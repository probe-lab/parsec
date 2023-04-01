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
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"
)

const ipfsProtocolPrefix = "/ipfs"

type Host struct {
	host.Host
	DHT routing.Routing
}

func New(ctx context.Context, port int, fullRT bool, dhtServer bool, ldb string) (*Host, error) {
	addrs := []string{
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", port),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1/webtransport", port),
		fmt.Sprintf("/ip6/::/tcp/%d", port),
		fmt.Sprintf("/ip6/::/udp/%d/quic", port),
		fmt.Sprintf("/ip6/::/udp/%d/quic-v1", port),
		fmt.Sprintf("/ip6/::/udp/%d/quic-v1/webtransport", port),
	}

	limiter := rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits)
	rm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		return nil, errors.Wrap(err, "new resource manager")
	}

	if err = view.Register(metrics.DefaultViews...); err != nil {
		return nil, fmt.Errorf("register metric views: %w", err)
	}

	ds, err := leveldb.NewDatastore(ldb, nil)
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

			diskUsage.Set(float64(usage))
		}
	}()

	var dht routing.Routing
	h, err := libp2p.New(
		libp2p.ResourceManager(rm),
		libp2p.ListenAddrStrings(addrs...),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			mode := kaddht.ModeClient
			if dhtServer {
				mode = kaddht.ModeServer
			}

			if fullRT {
				log.Infoln("Using full accelerated DHT client")
				dht, err = fullrt.NewFullRT(h, ipfsProtocolPrefix, fullrt.DHTOption(
					kaddht.BootstrapPeers(kaddht.GetDefaultBootstrapPeerAddrInfos()...),
					kaddht.BucketSize(20),
					kaddht.Mode(mode),
					kaddht.Datastore(ds),
				))
			} else {
				log.Infoln("Using standard DHT client")
				dht, err = kaddht.New(ctx, h, kaddht.Mode(mode), kaddht.Datastore(ds))
			}
			return dht, err
		}),
	)
	if err != nil {
		return nil, errors.Wrap(err, "new libp2p host")
	}

	newHost := &Host{
		Host: h,
		DHT:  dht,
	}

	log.WithField("localID", h.ID()).Info("Initialized new libp2p host")

	return newHost, nil
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
