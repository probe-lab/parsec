package dht

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/dennis-tra/parsec/pkg/wrap"
)

type Host struct {
	host.Host

	DHT        *kaddht.IpfsDHT
	StartedAt  *time.Time
	Transports []*wrap.Notifier
	MsgSender  *wrap.MessageSenderImpl
}

func New(ctx context.Context, port int) (*Host, error) {
	tcp, tcpTrpt := wrap.NewTCPTransport()
	quic, quicTrpt := wrap.NewQuicTransport()
	msgSender := wrap.NewMessageSenderImpl()

	newHost := &Host{
		MsgSender:  msgSender,
		Transports: []*wrap.Notifier{tcp.Notifier, quic.Notifier},
	}

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

	var dht *kaddht.IpfsDHT
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(addrs...),
		libp2p.Transport(tcpTrpt),
		libp2p.Transport(quicTrpt),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			dht, err = kaddht.New(ctx, h, kaddht.MessageSenderImpl(msgSender.Init))
			return dht, err
		}),
	)
	if err != nil {
		return nil, errors.Wrap(err, "new libp2p host")
	}

	now := time.Now()
	newHost.Host = h
	newHost.DHT = dht
	newHost.StartedAt = &now

	log.WithField("localID", h.ID()).Info("Initialized new libp2p host")

	return newHost, nil
}
