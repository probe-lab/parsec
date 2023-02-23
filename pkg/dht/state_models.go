package dht

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
)

type DialSpan struct {
	RemotePeerID peer.ID
	Maddr        ma.Multiaddr
	Start        time.Time
	End          time.Time
	Trpt         string
	Error        error
}

type ConnectionSpan struct {
	RemotePeerID peer.ID
	Maddr        ma.Multiaddr
	Start        time.Time
	End          time.Time
}

type FindNodesSpan struct {
	QueryID      uuid.UUID
	RemotePeerID peer.ID
	Start        time.Time
	End          time.Time
	CloserPeers  []*peer.AddrInfo
	Error        error
}

type GetProvidersSpan struct {
	QueryID      uuid.UUID
	RemotePeerID peer.ID
	Start        time.Time
	End          time.Time
	Providers    []*peer.AddrInfo
	CloserPeers  []*peer.AddrInfo
	Error        error
}

type AddProvidersSpan struct {
	QueryID       uuid.UUID
	RemotePeerID  peer.ID
	CID           cid.Cid
	Start         time.Time
	ProviderAddrs []ma.Multiaddr
	End           time.Time
	Error         error
}

type PeerInfo struct {
	PeerID       peer.ID
	AgentVersion string
	Protocols    []protocol.ID
	Addrs        []ma.Multiaddr
}

func NewPeerInfo(pid peer.ID, ps peerstore.Peerstore) *PeerInfo {
	pi := &PeerInfo{PeerID: pid}
	pi.SetFromPeerstore(ps)
	return pi
}

func (pi *PeerInfo) SetFromPeerstore(ps peerstore.Peerstore) bool {
	changed := false

	av := ""
	if agent, err := ps.Get(pi.PeerID, "AgentVersion"); err == nil {
		av = agent.(string)
	}

	if av != "" {
		pi.AgentVersion = av
		changed = true
	}

	protocols := []protocol.ID{}
	if prots, err := ps.GetProtocols(pi.PeerID); err == nil {
		protocols = prots
	}

	sort.Slice(protocols, func(i, j int) bool {
		return string(protocols[i]) < string(protocols[j])
	})

	if len(protocols) != len(pi.Protocols) {
		pi.Protocols = protocols
		changed = true
	} else {
		for i := 0; i < len(protocols); i++ {
			if pi.Protocols[i] != protocols[i] {
				pi.Protocols = protocols
				changed = true
				break
			}
		}
	}

	maddrs := ps.Addrs(pi.PeerID)

	sort.Slice(maddrs, func(i, j int) bool {
		return maddrs[i].String() < maddrs[j].String()
	})

	if len(maddrs) != len(pi.Addrs) {
		pi.Addrs = maddrs
		changed = true
	} else {
		for i := 0; i < len(maddrs); i++ {
			if pi.Addrs[i].String() != maddrs[i].String() {
				pi.Addrs = maddrs
				changed = true
				break
			}
		}
	}

	return changed
}
