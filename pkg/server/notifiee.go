package server

import (
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	log "github.com/sirupsen/logrus"
)

type ConnectionEvent struct {
	RemoteMaddr  multiaddr.Multiaddr
	OpenedAt     time.Time
	Direction    string
	Limited      bool
	ConnectionID string
	Transport    string
	Address      net.IP
}

func (s *Server) Listen(n network.Network, multiaddr multiaddr.Multiaddr) {
}

func (s *Server) ListenClose(n network.Network, multiaddr multiaddr.Multiaddr) {
}

func (s *Server) Connected(n network.Network, conn network.Conn) {
	go s.trackConnectionEvent(conn, "connect")
}

func (s *Server) Disconnected(n network.Network, conn network.Conn) {
	go s.trackConnectionEvent(conn, "disconnect")
}

func (s *Server) trackConnectionEvent(conn network.Conn, evtType string) {
	s.host.BasicHost.IDService().IdentifyConn(conn)

	ipnet, err := manet.ToIP(conn.RemoteMultiaddr())
	if err != nil || len(ipnet) == 0 {
		log.WithError(err).Warnln("Couldn't extract net addr")
	}

	trpt, err := toTransport(conn.RemoteMultiaddr())
	if err != nil || len(ipnet) == 0 {
		log.WithError(err).Warnln("Couldn't extract transport")
	}

	stat := conn.Stat()
	ce := ConnectionEvent{
		ConnectionID: conn.ID(),
		RemoteMaddr:  conn.RemoteMultiaddr(),
		OpenedAt:     stat.Opened,
		Direction:    stat.Direction.String(),
		Limited:      stat.Limited,
		Transport:    trpt,
		Address:      ipnet,
	}

	if err := s.fhClient.Submit(evtType, conn.RemotePeer(), ce); err != nil {
		log.WithError(err).Warnf("Couldn't submit %s event", evtType)
	}
}

func toTransport(addr multiaddr.Multiaddr) (string, error) {
	var transport string
	multiaddr.ForEach(addr, func(c multiaddr.Component) bool {
		switch c.Protocol().Code {
		case multiaddr.P_QUIC:
		case multiaddr.P_QUIC_V1:
		case multiaddr.P_TCP:
		case multiaddr.P_WS:
		case multiaddr.P_WSS:
		case multiaddr.P_WEBTRANSPORT:
		case multiaddr.P_WEBRTC:
		default:
			return true
		}

		transport = c.Protocol().Name

		return false
	})

	if transport == "" {
		return "", fmt.Errorf("no supported transport")
	}

	return transport, nil
}
