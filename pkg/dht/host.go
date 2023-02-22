package dht

import (
	"context"
	"time"

	"github.com/dennis-tra/parsec/pkg/util"
	"github.com/dennis-tra/parsec/pkg/wrap"
	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Host struct {
	host.Host

	DHT        *kaddht.IpfsDHT
	StartedAt  *time.Time
	Transports []*wrap.Notifier
	MsgSender  *wrap.MessageSenderImpl
}

func New(ctx context.Context) (*Host, error) {
	tcp, tcpTrpt := wrap.NewTCPTransport()
	quic, quicTrpt := wrap.NewQuicTransport()
	msgSender := wrap.NewMessageSenderImpl()

	newHost := &Host{
		MsgSender:  msgSender,
		Transports: []*wrap.Notifier{tcp.Notifier, quic.Notifier},
	}

	var dht *kaddht.IpfsDHT
	h, err := libp2p.New(
		libp2p.DefaultListenAddrs,
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

	log.WithField("localID", util.FmtPeerID(h.ID())).Info("Initialized new libp2p host")
	return newHost, nil
}
