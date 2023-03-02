package dht

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/routing"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Host struct {
	host.Host

	DHT       *kaddht.IpfsDHT
	StartedAt *time.Time
}

func New(ctx context.Context, port int) (*Host, error) {
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

	var dht *kaddht.IpfsDHT
	h, err := libp2p.New(
		libp2p.ResourceManager(rm),
		libp2p.ListenAddrStrings(addrs...),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			dht, err = kaddht.New(ctx, h)
			return dht, err
		}),
	)
	if err != nil {
		return nil, errors.Wrap(err, "new libp2p host")
	}

	now := time.Now()
	newHost := &Host{
		Host:      h,
		DHT:       dht,
		StartedAt: &now,
	}

	log.WithField("localID", h.ID()).Info("Initialized new libp2p host")

	return newHost, nil
}
