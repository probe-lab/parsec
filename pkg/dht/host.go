package dht

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/fullrt"
	"github.com/libp2p/go-libp2p-kad-dht/metrics"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/routing"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	mh "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"

	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/dennis-tra/parsec/pkg/config"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
)

const ipfsProtocolPrefix = "/ipfs"

type Host struct {
	host.Host
	DHT           routing.Routing
	BasicHost     *basichost.BasicHost
	indexer       *Indexer
	multihashesLk sync.RWMutex
	multihashes   map[string]multiHashEntry
}

type multiHashEntry struct {
	ts  time.Time
	mhs []mh.Multihash
}

func New(ctx context.Context, fh *firehose.Firehose, conf config.ServerConfig) (*Host, error) {
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

	mode := kaddht.ModeClient
	if conf.DHTServer {
		mode = kaddht.ModeServer
	}

	origProvStore, err := providers.NewProviderManager(ctx, basicHost.ID(), basicHost.Peerstore(), ds)
	if err != nil {
		return nil, fmt.Errorf("initializing default provider manager: %w", err)
	}

	var provStore providers.ProviderStore
	if conf.FirehoseRPCs {
		log.Infoln("Using wrapped provider store")
		wrapProvStore, err := NewProviderStore(ctx, origProvStore, basicHost, fh, conf)
		if err != nil {
			return nil, fmt.Errorf("initializing wrapped provider store: %w", err)
		}
		provStore = wrapProvStore
	} else {
		provStore = origProvStore
	}

	var dht routing.Routing
	if conf.FullRT {
		log.Infoln("Using full accelerated DHT client")
		dht, err = fullrt.NewFullRT(basicHost, ipfsProtocolPrefix, fullrt.DHTOption(
			kaddht.BootstrapPeers(kaddht.GetDefaultBootstrapPeerAddrInfos()...),
			kaddht.BucketSize(20),
			kaddht.Mode(mode),
			kaddht.Datastore(ds),
			kaddht.ProviderStore(provStore),
		))
	} else {
		log.Infoln("Using standard DHT client")
		opts := []kaddht.Option{kaddht.Mode(mode), kaddht.Datastore(ds)}
		if conf.OptProv {
			opts = append(opts,
				kaddht.EnableOptimisticProvide(),
				kaddht.ProviderStore(provStore))
		}
		dht, err = kaddht.New(ctx, basicHost, opts...)
	}
	if err != nil {
		return nil, fmt.Errorf("new router: %w", err)
	}

	newHost := &Host{
		Host:        routedhost.Wrap(basicHost, dht),
		BasicHost:   basicHost.(*basichost.BasicHost),
		DHT:         dht,
		multihashes: map[string]multiHashEntry{},
	}

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

	log.WithField("localID", newHost.ID()).Info("Initialized new libp2p host")

	if err = newHost.subscribeForEvents(); err != nil {
		return nil, fmt.Errorf("subscribe for events: %w", err)
	}

	return newHost, nil
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
	if h.indexer != nil {
		if err := h.indexer.engine.Shutdown(); err != nil {
			log.WithError(err).WithField("indexer", h.indexer.hostname).Warnln("Failed to shut down indexer engine")
		}
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
