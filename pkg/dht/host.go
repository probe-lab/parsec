package dht

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/fullrt"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/routing"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const ipfsProtocolPrefix = "/ipfs"

type Host struct {
	host.Host
	DHT routing.Routing
}

func New(ctx context.Context, port int, fullRT bool, dhtServer bool) (*Host, error) {
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

	var dht routing.Routing
	h, err := libp2p.New(
		libp2p.ResourceManager(rm),
		libp2p.ListenAddrStrings(addrs...),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			mode := kaddht.ModeClient
			if dhtServer {
				mode = kaddht.ModeServer
			}

			var err error
			if fullRT {
				log.Infoln("Using full accelerated DHT client")
				dht, err = fullrt.NewFullRT(h, ipfsProtocolPrefix, fullrt.DHTOption(
					kaddht.BootstrapPeers(kaddht.GetDefaultBootstrapPeerAddrInfos()...),
					kaddht.BucketSize(20),
					kaddht.Mode(mode),
				))
			} else {
				log.Infoln("Using standard DHT client")
				dht, err = kaddht.New(ctx, h, kaddht.Mode(mode))
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
