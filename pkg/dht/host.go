package dht

import (
	"context"
	"fmt"
	"time"

	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/fullrt"
	"github.com/libp2p/go-libp2p-kad-dht/metrics"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/routing"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"

	"github.com/dennis-tra/parsec/pkg/config"
)

const ipfsProtocolPrefix = "/ipfs"

type Host struct {
	host.Host
	DHT       routing.Routing
	BasicHost *basichost.BasicHost
}

func New(ctx context.Context, conf config.ServerConfig) (*Host, error) {
	addrs := []string{
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", conf.PeerPort),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", conf.PeerPort),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", conf.PeerPort),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1/webtransport", conf.PeerPort),
		fmt.Sprintf("/ip6/::/tcp/%d", conf.PeerPort),
		fmt.Sprintf("/ip6/::/udp/%d/quic", conf.PeerPort),
		fmt.Sprintf("/ip6/::/udp/%d/quic-v1", conf.PeerPort),
		fmt.Sprintf("/ip6/::/udp/%d/quic-v1/webtransport", conf.PeerPort),
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

	go func() {
		ticker := time.NewTicker(time.Minute)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			usage, err := ds.DiskUsage(ctx)
			if err != nil {
				log.WithError(err).Warnln("Couldn't get disk usage")
				continue
			}

			diskUsageGauge.Set(float64(usage))
		}
	}()

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

	var dht routing.Routing
	if conf.FullRT {
		log.Infoln("Using full accelerated DHT client")
		dht, err = fullrt.NewFullRT(basicHost, ipfsProtocolPrefix, fullrt.DHTOption(
			kaddht.BootstrapPeers(kaddht.GetDefaultBootstrapPeerAddrInfos()...),
			kaddht.BucketSize(20),
			kaddht.Mode(mode),
			kaddht.Datastore(ds),
		))
	} else {
		log.Infoln("Using standard DHT client")
		opts := []kaddht.Option{kaddht.Mode(mode), kaddht.Datastore(ds)}
		if conf.OptProv {
			opts = append(opts, kaddht.EnableOptimisticProvide())
		}
		dht, err = kaddht.New(ctx, basicHost, opts...)
	}
	if err != nil {
		return nil, fmt.Errorf("new router: %w", err)
	}
	newHost := &Host{
		Host:      routedhost.Wrap(basicHost, dht),
		BasicHost: basicHost.(*basichost.BasicHost),
		DHT:       dht,
	}

	go newHost.measureNetworkSize(ctx)

	log.WithField("localID", newHost.ID()).Info("Initialized new libp2p host")

	return newHost, nil
}

func (h Host) measureNetworkSize(ctx context.Context) {
	idht, ok := h.DHT.(*kaddht.IpfsDHT)
	if ok {
		go func() {
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
		}()
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
