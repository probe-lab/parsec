package server

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/fullrt"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/multiformats/go-multicodec"
	mh "github.com/multiformats/go-multihash"
	log "github.com/sirupsen/logrus"
	"go.uber.org/fx"

	"github.com/probe-lab/parsec/pkg/firehose"
	"github.com/probe-lab/parsec/pkg/util"
)

const ipfsProtocolPrefix = "/ipfs"

type HostConfig struct {
	Host                     string
	Port                     int
	FirehoseConnectionEvents bool
}

type Host struct {
	host.Host
	conf      *HostConfig
	fhClient  firehose.Submitter
	IdService identify.IDService
}

func InitHost(ctx context.Context, fhClient firehose.Submitter, conf *HostConfig) (*Host, error) {
	addrs := []string{
		fmt.Sprintf("/ip4/%s/tcp/%d", conf.Host, conf.Port),
		fmt.Sprintf("/ip4/%s/udp/%d/quic-v1", conf.Host, conf.Port),
		fmt.Sprintf("/ip4/%s/udp/%d/quic-v1/webtransport", conf.Host, conf.Port),
		fmt.Sprintf("/ip4/%s/udp/%d/webrtc-direct", conf.Host, conf.Port),
	}

	var id identify.IDService
	libp2pHost, err := libp2p.New(
		libp2p.ResourceManager(&network.NullResourceManager{}),
		libp2p.ListenAddrStrings(addrs...),
		libp2p.WithFxOption(fx.Populate(&id)),
		libp2p.DisableMetrics(),
		// TODO: remove
		libp2p.AddrsFactory(func(maddrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			return append(maddrs, multiaddr.StringCast(fmt.Sprintf("/ip4/2.202.214.161/tcp/%d", conf.Port)))
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("new libp2p host: %w", err)
	}

	h := &Host{
		Host:      libp2pHost,
		fhClient:  fhClient,
		IdService: id,
	}

	if conf.FirehoseConnectionEvents {
		libp2pHost.Network().Notify(h)
	}

	log.Infoln("Connecting to bootstrap peers...")
	for _, bp := range kaddht.GetDefaultBootstrapPeerAddrInfos() {
		log.WithField("peerID", util.FmtPeerID(bp.ID)).Infoln("Connecting...")
		if err = h.Connect(ctx, bp); err != nil {
			log.WithError(err).WithField("peerID", util.FmtPeerID(bp.ID)).Warnln("Could not connect to bootstrap peer")
		}
	}

	log.WithField("localID", h.ID()).Info("Initialized new libp2p host")

	if err = h.subscribeForEvents(); err != nil {
		return nil, fmt.Errorf("subscribe for events: %w", err)
	}

	return h, nil
}

type multiHashEntry struct {
	ts  time.Time
	mhs []mh.Multihash
}

func (h *Host) AgentVersion(p peer.ID) string {
	agentVersion, err := h.Peerstore().Get(p, "AgentVersion")
	if err == nil {
		if str, ok := agentVersion.(string); ok {
			return str
		}
	}
	return ""
}

func (h *Host) waitForPublicIPv4Address(ctx context.Context) (string, error) {
	sub, err := h.EventBus().Subscribe(new(event.EvtLocalAddressesUpdated))
	if err != nil {
		return "", fmt.Errorf("event bus subscription: %w", err)
	}
	defer func() {
		if err := sub.Close(); err != nil {
			log.WithError(err).Warnln("Failed to close event bus subscription")
		}
	}()

	timeout := time.Minute
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.WithField("timeout", timeout).Infoln("Waiting for the libp2p host to detect the public address")
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case evt := <-sub.Out():
			switch tevt := evt.(type) {
			case event.EvtLocalAddressesUpdated:
				for _, update := range tevt.Current {
					if !manet.IsPublicAddr(update.Address) {
						continue
					}

					ip, err := update.Address.ValueForProtocol(multiaddr.P_IP4)
					if err != nil {
						continue
					}

					log.Infoln("Detected public IPv4 address:", ip)
					return ip, nil
				}
			}

		}
	}
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

	log.WithField("size", len(denyMap)).Infoln("Loaded denied CIDs file")

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

	log.WithField("size", len(denyMap)).Infoln("Loaded badbits file")

	return denyMap, nil
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

func (i *IPNIServer) gcMultihashEntries(ctx context.Context) {
	t := time.NewTicker(time.Hour)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		i.multihashesLk.Lock()
		for contextID, entry := range i.multihashes {
			if entry.ts.After(time.Now().Add(-time.Hour)) {
				continue
			}
			delete(i.multihashes, contextID)
		}
		i.multihashesLk.Unlock()
	}
}

func RoutingTableSize(dht routing.Routing) int {
	if dht == nil {
		return 0
	}
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
