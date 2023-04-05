package server

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	log "github.com/sirupsen/logrus"

	"github.com/dennis-tra/parsec/pkg/config"
)

type ConnectionEvent struct {
	RemotePeer   string
	RemoteMaddr  multiaddr.Multiaddr
	OpenedAt     time.Time
	Timestamp    time.Time
	Direction    string
	Transient    bool
	ConnectionID string
	PartitionKey string
	AgentVersion string
	Transport    string
	Address      net.IP
	Fleet        string
	DBNodeID     int
	LocalPeer    string
	Region       string
	Type         string
}

func (s *Server) Listen(n network.Network, multiaddr multiaddr.Multiaddr) {
}

func (s *Server) ListenClose(n network.Network, multiaddr multiaddr.Multiaddr) {
}

func (s *Server) Connected(n network.Network, conn network.Conn) {
	go s.trackConnectionEvent(conn, "connect")
}

func (s *Server) Disconnected(n network.Network, conn network.Conn) {
	// Don't track disconnect events for now
	// go s.trackConnectionEvent(conn, "disconnect")
}

func (s Server) trackConnectionEvent(conn network.Conn, evtType string) {
	now := time.Now()
	s.host.BasicHost.IDService().IdentifyConn(conn)

	avStr := ""
	agentVersion, err := s.host.Peerstore().Get(conn.RemotePeer(), "AgentVersion")
	if err == nil {
		if str, ok := agentVersion.(string); ok {
			avStr = str
		}
	}

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
		Fleet:        s.conf.Fleet,
		DBNodeID:     s.dbNode.ID,
		Region:       config.Global.AWSRegion,
		LocalPeer:    conn.LocalPeer().String(),
		ConnectionID: conn.ID(),
		RemotePeer:   conn.RemotePeer().String(),
		RemoteMaddr:  conn.RemoteMultiaddr(),
		OpenedAt:     stat.Opened,
		Direction:    stat.Direction.String(),
		Transient:    stat.Transient,
		AgentVersion: avStr,
		Transport:    trpt,
		Address:      ipnet,
		Timestamp:    now,
		PartitionKey: fmt.Sprintf("%s-%s", config.Global.AWSRegion, s.conf.Fleet),
		Type:         evtType,
	}

	data, err := json.Marshal(ce)
	if err != nil {
		log.WithError(err).Warnln("Couldn't marshal connection event")
		return
	}

	log.Infoln("Put record!")
	_, err = s.fh.PutRecord(&firehose.PutRecordInput{
		Record:             &firehose.Record{Data: data},
		DeliveryStreamName: aws.String(s.stream),
	})
	if err != nil {
		log.WithError(err).Warnln("Couldn't put connection event")
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
