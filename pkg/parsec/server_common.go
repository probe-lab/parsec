package parsec

import (
	"math/big"
	"time"

	"github.com/dennis-tra/parsec/pkg/dht"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type CommonResponse struct {
	Start       time.Time
	End         time.Time
	Peers       []Peer
	Dials       []Dial
	Connections []Connection
	PeerSet     []PeerSet
}

type Connection struct {
	RemotePeerID peer.ID
	Maddr        string
	Start        time.Time
	End          time.Time
}

type Dial struct {
	RemotePeerID peer.ID
	Maddr        string
	Start        time.Time
	End          time.Time
	Trpt         string
	Error        string
}

type PeerSet struct {
	QueryID      uuid.UUID
	RemotePeerID peer.ID
	Distance     *big.Int
	State        int
	ReferredBy   peer.ID
}

type Peer struct {
	PeerID       peer.ID
	AgentVersion string
	Protocols    []protocol.ID
	Addrs        []string
}

func CommonResponseFromState(state *dht.CommonState) CommonResponse {
	resp := CommonResponse{
		Start:       state.Start,
		End:         state.End,
		Peers:       []Peer{},
		Dials:       make([]Dial, len(state.Dials)),
		Connections: make([]Connection, len(state.Connections)),
		PeerSet:     []PeerSet{},
	}

	state.RelevantPeers.Range(func(key, value any) bool {
		pi := value.(*dht.PeerInfo)
		maddrs := make([]string, len(pi.Addrs))
		for i, addr := range pi.Addrs {
			maddrs[i] = addr.String()
		}
		resp.Peers = append(resp.Peers, Peer{
			PeerID:       pi.PeerID,
			AgentVersion: pi.AgentVersion,
			Protocols:    pi.Protocols,
			Addrs:        maddrs,
		})
		return true
	})

	for id, ps := range state.PeerSet {
		for _, peerState := range ps.AllStates() {
			resp.PeerSet = append(resp.PeerSet, PeerSet{
				QueryID:      id,
				RemotePeerID: peerState.ID,
				Distance:     peerState.Distance,
				State:        int(peerState.State),
				ReferredBy:   peerState.ReferredBy,
			})
		}
	}

	for i, d := range state.Dials {
		errStr := ""
		if d.Error != nil {
			errStr = d.Error.Error()
		}
		resp.Dials[i] = Dial{
			RemotePeerID: d.RemotePeerID,
			Maddr:        d.Maddr.String(),
			Start:        d.Start,
			End:          d.End,
			Trpt:         d.Trpt,
			Error:        errStr,
		}
	}

	for i, c := range state.Connections {
		resp.Connections[i] = Connection{
			RemotePeerID: c.RemotePeerID,
			Maddr:        c.Maddr.String(),
			Start:        c.Start,
			End:          c.End,
		}
	}

	return resp
}
