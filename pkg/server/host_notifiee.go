package server

import (
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
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

func (h *Host) Listen(n network.Network, multiaddr multiaddr.Multiaddr) {
}

func (h *Host) ListenClose(n network.Network, multiaddr multiaddr.Multiaddr) {
}

func (h *Host) Connected(n network.Network, conn network.Conn) {
	go h.trackConnectionEvent(conn, "connect")
}

func (h *Host) Disconnected(n network.Network, conn network.Conn) {
	go h.trackConnectionEvent(conn, "disconnect")
}

func (h *Host) trackConnectionEvent(conn network.Conn, evtType string) {
	h.IdService.IdentifyConn(conn)

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

	addrInfo := peer.AddrInfo{
		ID:    conn.RemotePeer(),
		Addrs: []multiaddr.Multiaddr{conn.RemoteMultiaddr()},
	}

	if err := h.fhClient.Submit(evtType, h.ID(), addrInfo, h.AgentVersion(conn.RemotePeer()), ce); err != nil {
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
