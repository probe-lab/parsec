package dht

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/fullrt"
	"github.com/libp2p/go-libp2p-kad-dht/metrics"
	pb "github.com/libp2p/go-libp2p-kad-dht/pb"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
	mh "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"

	"github.com/dennis-tra/parsec/pkg/config"
	"github.com/dennis-tra/parsec/pkg/firehose"
)

const ipfsProtocolPrefix = "/ipfs"

type Host struct {
	host.Host
	conf          config.ServerConfig
	fhClient      *firehose.Client
	DHT           routing.Routing
	BasicHost     *basichost.BasicHost
	indexer       *Indexer
	multihashesLk sync.RWMutex
	multihashes   map[string]multiHashEntry

	mapMu           sync.RWMutex
	badbitsMap      map[string]struct{}
	deniedCIDsMap   map[string]string
	badbitsStats    os.FileInfo
	deniedCIDsStats os.FileInfo
}

type multiHashEntry struct {
	ts  time.Time
	mhs []mh.Multihash
}

func New(ctx context.Context, fhClient *firehose.Client, conf config.ServerConfig) (*Host, error) {
	// Don't listen on quic-v1 since it's not supported by IPNI at the moment
	addrs := []string{
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", conf.PeerPort),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", conf.PeerPort),
		fmt.Sprintf("/ip6/::/tcp/%d", conf.PeerPort),
		fmt.Sprintf("/ip6/::/udp/%d/quic", conf.PeerPort),

		// fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", conf.PeerPort),
		// fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1/webtransport", conf.PeerPort),
		// fmt.Sprintf("/ip6/::/udp/%d/quic-v1", conf.PeerPort),
		// fmt.Sprintf("/ip6/::/udp/%d/quic-v1/webtransport", conf.PeerPort),
	}

	limiter := rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits)
	rm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		return nil, errors.Wrap(err, "new resource manager")
	}

	if err = view.Register(metrics.DefaultViews...); err != nil {
		return nil, fmt.Errorf("register metric views: %w", err)
	}

	ds, err := leveldb.NewDatastore(conf.LevelDB, nil)
	if err != nil {
		return nil, fmt.Errorf("leveldb datastore: %w", err)
	}

	basicHost, err := libp2p.New(
		libp2p.ResourceManager(rm),
		libp2p.ListenAddrStrings(addrs...),
	)
	if err != nil {
		return nil, fmt.Errorf("new libp2p host: %w", err)
	}

	badbitsStats, err := os.Stat(conf.Badbits)
	if err != nil {
		return nil, fmt.Errorf("stats from badbits file %s: %w", conf.Badbits, err)
	}

	badbitsMap, err := loadBadbits(conf.Badbits)
	if err != nil {
		return nil, fmt.Errorf("load badbits: %w", err)
	}

	deniedCIDsStats, err := os.Stat(conf.DeniedCIDs)
	if err != nil {
		return nil, fmt.Errorf("stats from denied CIDs file %s: %w", conf.Badbits, err)
	}

	deniedCIDsMap, err := loadDeniedCIDs(conf.DeniedCIDs)
	if err != nil {
		return nil, fmt.Errorf("load denied CIDs: %w", err)
	}

	mode := kaddht.ModeClient
	if conf.DHTServer {
		mode = kaddht.ModeServer
	}

	newHost := &Host{
		conf:            conf,
		BasicHost:       basicHost.(*basichost.BasicHost),
		fhClient:        fhClient,
		multihashes:     map[string]multiHashEntry{},
		badbitsStats:    badbitsStats,
		deniedCIDsStats: deniedCIDsStats,
		badbitsMap:      badbitsMap,
		deniedCIDsMap:   deniedCIDsMap,
	}

	var dht routing.Routing
	if conf.FullRT {
		log.Infoln("Using full accelerated DHT client")
		dht, err = fullrt.NewFullRT(basicHost, ipfsProtocolPrefix, fullrt.DHTOption(
			kaddht.BootstrapPeers(kaddht.GetDefaultBootstrapPeerAddrInfos()...),
			kaddht.BucketSize(20),
			kaddht.Mode(mode),
			kaddht.Datastore(ds),
			kaddht.DhtHandlerWrapper(newHost.handlerWrapper),
		))
	} else {
		log.Infoln("Using standard DHT client")
		opts := []kaddht.Option{
			kaddht.Mode(mode),
			kaddht.Datastore(ds),
			kaddht.DhtHandlerWrapper(newHost.handlerWrapper),
		}
		if conf.OptProv {
			opts = append(opts, kaddht.EnableOptimisticProvide())
		}
		dht, err = kaddht.New(ctx, basicHost, opts...)
	}
	if err != nil {
		return nil, fmt.Errorf("new router: %w", err)
	}

	newHost.Host = routedhost.Wrap(basicHost, dht)
	newHost.DHT = dht

	if config.Server.IndexerHost != "" {
		newHost.indexer, err = newHost.initIndexer(ctx, ds, config.Server.IndexerHost)
		if err != nil {
			return nil, fmt.Errorf("init indexer: %w", err)
		}
	} else {
		log.Infoln("No indexer configured")
	}

	go newHost.measureNetworkSize(ctx)
	go newHost.measureDiskUsage(ctx, ds)
	go newHost.gcMultihashEntries(ctx)
	go newHost.reloadMaps(ctx)

	log.WithField("localID", newHost.ID()).Info("Initialized new libp2p host")

	if err = newHost.subscribeForEvents(); err != nil {
		return nil, fmt.Errorf("subscribe for events: %w", err)
	}

	return newHost, nil
}

func loadDeniedCIDs(filename string) (map[string]string, error) {
	if filename == "" {
		log.Infoln("No denied CIDs file configured")
		return map[string]string{}, nil
	}

	log.Infoln("Parsing Denied CIDs file")
	// Open the provided filename
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create a new CSV reader reading from the opened file
	reader := csv.NewReader(file)

	// Assume we have a header line, read it
	_, err = reader.Read()
	if err != nil {
		return nil, err
	}

	// Now, process the rest of the CSV records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	denyMap := map[string]string{}
	for _, record := range records {
		if len(record) != 2 {
			continue // Not enough fields in the record
		}
		CID := record[0]
		source := record[1]
		denyMap[CID] = source
	}

	return denyMap, nil
}

func loadBadbits(filename string) (map[string]struct{}, error) {
	if filename == "" {
		log.Infoln("No Badbits file configured")
		return map[string]struct{}{}, nil
	}

	log.Infoln("Parsing Badbits file")
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open bad bits file: %w", err)
	}

	denyMap := map[string]struct{}{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "//") {
			continue
		}

		denyMap[line[2:]] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("closing bad bits file: %w", err)
	}

	return denyMap, nil
}

func (h *Host) reloadMaps(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	for {
		select {
		case <-ctx.Done():
		case <-t.C:

			log.Infoln("Reloading Maps")
			badbitsStats, err := os.Stat(h.conf.Badbits)
			if err != nil {
				log.WithError(err).Warnln("Couldn't get badbits file stats")
			} else if badbitsStats.ModTime().After(h.badbitsStats.ModTime()) {
				log.Infoln("Reloading badbits map because it has changed")
				badbitsMap, err := loadBadbits(h.conf.Badbits)
				if err != nil {
					log.WithError(err).Warnln("Couldn't reload badbits")
				} else {
					h.mapMu.Lock()
					h.badbitsStats = badbitsStats
					h.badbitsMap = badbitsMap
					h.mapMu.Unlock()
				}
			} else {
				log.Infoln("Not reloading badbits map because it's unchanged")
			}

			deniedCIDsStats, err := os.Stat(h.conf.DeniedCIDs)
			if err != nil {
				log.WithError(err).Warnln("Couldn't get denied CIDs file stats")
			} else if deniedCIDsStats.ModTime().After(h.deniedCIDsStats.ModTime()) {
				log.Infoln("Reloading badbits map because it has changed")
				deniedCIDsMap, err := loadDeniedCIDs(h.conf.DeniedCIDs)
				if err != nil {
					log.WithError(err).Warnln("Couldn't reload denied CIDs")
				} else {
					h.mapMu.Lock()
					h.deniedCIDsStats = deniedCIDsStats
					h.deniedCIDsMap = deniedCIDsMap
					h.mapMu.Unlock()
				}
			} else {
				log.Infoln("Not reloading badbits map because it's unchanged")
			}

		}
	}
}

type RPCRequest struct {
	MessageType string
	Multihash   string
	Match       string
	Source      string
}

var codecs = []multicodec.Code{
	multicodec.Raw,
	multicodec.DagPb,
	multicodec.DagCbor,
	multicodec.DagJose,
}

func (h *Host) handlerWrapper(handler func(context.Context, peer.ID, *pb.Message) (*pb.Message, error), ctx context.Context, id peer.ID, req *pb.Message) (*pb.Message, error) {
	switch req.GetType() {
	case pb.Message_ADD_PROVIDER, pb.Message_GET_PROVIDERS:
		go func() {
			_, mh, err := multihash.MHFromBytes(req.GetKey())
			if err != nil {
				log.WithError(err).Warnln("failed to parse multihash from key")
				return
			}

			rec := &RPCRequest{
				MessageType: req.GetType().String(),
				Multihash:   mh.String(),
			}

			h.mapMu.RLock()
			for _, codec := range codecs {
				v1 := cid.NewCidV1(uint64(codec), mh)

				if source, found := h.deniedCIDsMap[v1.String()]; found {
					rec.Match = v1.String()
					rec.Source = source
					log.WithField("source", source).Infof("Found CID in %s!", source)
					break
				}

				preimage := v1.String() + "/"
				hsh := sha256.Sum256([]byte(preimage))
				matchStr := hex.EncodeToString(hsh[:])

				if _, found := h.badbitsMap[matchStr]; found {
					rec.Match = preimage
					rec.Source = "badbits"
					log.WithField("preimage", preimage).Infoln("Preimage found!", id)
					break
				}
			}
			h.mapMu.RUnlock()

			if err := h.fhClient.Submit("add_provider", id, rec); err != nil {
				log.WithError(err).Warnln("Couldn't submit add_provider event")
			}
		}()
	default:
	}

	return handler(ctx, id, req)
}

func (h *Host) subscribeForEvents() error {
	sub, err := h.EventBus().Subscribe([]interface{}{new(event.EvtLocalAddressesUpdated), new(event.EvtLocalReachabilityChanged)})
	if err != nil {
		return fmt.Errorf("event bus subscription: %w", err)
	}

	go func() {
		for evt := range sub.Out() {
			switch evt := evt.(type) {
			case event.EvtLocalAddressesUpdated:
				log.Infoln("libp2p host Multiaddresses updated:")
				for i, update := range evt.Current {
					log.Infof("  [%d] %s (%d)\n", i, update.Address, update.Action)
				}
			case event.EvtLocalReachabilityChanged:
				log.Infoln("New reachability:", evt.Reachability.String())
			}
		}
	}()

	return nil
}

func (h *Host) Close() error {
	if err := h.indexer.engine.Shutdown(); err != nil {
		log.WithError(err).WithField("indexer", h.indexer.hostname).Warnln("Failed to shut down indexer engine")
	}

	return h.Host.Close()
}

func (h *Host) measureDiskUsage(ctx context.Context, ds *leveldb.Datastore) {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		usage, err := ds.DiskUsage(ctx)
		if err != nil {
			log.WithError(err).Warnln("Failed getting disk usage")
			continue
		}

		diskUsageGauge.Set(float64(usage))
	}
}

func (h *Host) measureNetworkSize(ctx context.Context) {
	idht, ok := h.DHT.(*kaddht.IpfsDHT)
	if !ok {
		return
	}

	t := time.NewTicker(time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		netSize, err := idht.NetworkSize()
		if err != nil {
			continue
		}
		netSizeGauge.Set(float64(netSize))
	}
}

func (h *Host) gcMultihashEntries(ctx context.Context) {
	t := time.NewTicker(time.Hour)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		h.multihashesLk.Lock()
		for contextID, entry := range h.multihashes {
			if entry.ts.After(time.Now().Add(-time.Hour)) {
				continue
			}
			delete(h.multihashes, contextID)
		}
		h.multihashesLk.Unlock()
	}
}

func RoutingTableSize(dht routing.Routing) int {
	frt, ok := dht.(*fullrt.FullRT)
	if ok {
		return len(frt.Stat())
	}

	ipfsdht, ok := dht.(*kaddht.IpfsDHT)
	if ok {
		return ipfsdht.RoutingTable().Size()
	}

	panic("unrecognise DHT client implementation")
}

func genProbes(start mh.Multihash, count int) ([]mh.Multihash, error) {
	probes := make([]mh.Multihash, count)
	hash := start
	for i := 0; i < count; i++ {
		probe, err := mh.Sum(hash, mh.SHA2_256, -1)
		if err != nil {
			return nil, fmt.Errorf("gen probe digest: %w", err)
		}
		probes[i] = probe
		hash = probe
	}

	return probes, nil
}

func fmtContextID(contextID []byte) string {
	str := base64.StdEncoding.EncodeToString(contextID)
	if len(str) >= 16 {
		return str[:16]
	}
	return str
}

func fmtMultihash(m mh.Multihash) string {
	str := m.B58String()
	if len(str) >= 16 {
		return str[:16]
	}
	return str
}
