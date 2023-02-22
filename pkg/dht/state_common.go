package dht

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/qpeerset"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	ma "github.com/multiformats/go-multiaddr"
)

type CommonState struct {
	h      *Host
	CID    cid.Cid
	cancel context.CancelFunc

	Start time.Time
	End   time.Time

	dialsLk sync.RWMutex
	Dials   []*DialSpan

	connectionsStartedLk sync.RWMutex
	ConnectionsStarted   map[peer.ID]time.Time

	connectionsLk sync.RWMutex
	Connections   []*ConnectionSpan

	PeerSet map[uuid.UUID]*qpeerset.QueryPeerset

	RelevantPeers sync.Map
}

func NewCommonState(h *Host, c cid.Cid) *CommonState {
	return &CommonState{
		h:                    h,
		CID:                  c,
		dialsLk:              sync.RWMutex{},
		Dials:                []*DialSpan{},
		connectionsStartedLk: sync.RWMutex{},
		ConnectionsStarted:   map[peer.ID]time.Time{},
		connectionsLk:        sync.RWMutex{},
		Connections:          []*ConnectionSpan{},
		PeerSet:              map[uuid.UUID]*qpeerset.QueryPeerset{},
		RelevantPeers:        sync.Map{},
	}
}

func (cs *CommonState) Register(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	ctx, lookupEvents := kaddht.RegisterForLookupEvents(ctx)
	ctx, queryEvents := routing.RegisterForQueryEvents(ctx)

	go cs.consumeLookupEvents(lookupEvents)
	go cs.consumeQueryEvents(queryEvents)

	for _, notifier := range cs.h.Transports {
		notifier.Notify(cs)
	}

	cs.h.Host.Network().Notify(cs)

	cs.cancel = cancel

	return ctx
}

func (cs *CommonState) Unregister() {
	cs.h.Host.Network().StopNotify(cs)

	for _, notifier := range cs.h.Transports {
		notifier.StopNotify(cs)
	}

	cs.cancel()
	cs.cancel = nil

	cs.filterDials()
	cs.filterConnections()
}

func (cs *CommonState) storeRelevantPeer(pid peer.ID) {
	newPi := NewPeerInfo(pid, cs.h.Peerstore())
	pi, loaded := cs.RelevantPeers.LoadOrStore(pid, newPi)
	if !loaded {
		return
	}

	changed := pi.(*PeerInfo).SetFromPeerstore(cs.h.Peerstore())
	if changed {
		cs.RelevantPeers.Store(pid, pi)
	}
}

func (cs *CommonState) PeerInfos() map[peer.ID]*PeerInfo {
	pis := map[peer.ID]*PeerInfo{}

	cs.RelevantPeers.Range(func(pid, value interface{}) bool {
		pi, ok := value.(*PeerInfo)
		if !ok {
			return false
		}

		pis[pi.PeerID] = pi

		return true
	})

	return pis
}

func (cs *CommonState) consumeLookupEvents(lookupEvents <-chan *kaddht.LookupEvent) {
	for event := range lookupEvents {
		cs.ensurePeerset(event)
		if event.Request != nil {
			cs.trackLookupRequest(event)
		} else if event.Response != nil {
			cs.trackLookupResponse(event)
		}
	}
}

func (cs *CommonState) consumeQueryEvents(queryEvents <-chan *routing.QueryEvent) {
	for event := range queryEvents {
		switch event.Type {
		case routing.DialingPeer:
			cs.trackConnectionStart(event.ID)
		}
	}
}

func (cs *CommonState) ensurePeerset(evt *kaddht.LookupEvent) {
	if _, found := cs.PeerSet[evt.ID]; !found {
		cs.PeerSet[evt.ID] = qpeerset.NewQueryPeerset(string(cs.CID.Hash()))
	}
}

func (cs *CommonState) trackLookupResponse(evt *kaddht.LookupEvent) {
	for _, p := range evt.Response.Heard {
		if p.Peer == cs.h.ID() { // don't add self.
			continue
		}
		cs.PeerSet[evt.ID].TryAdd(p.Peer, evt.Response.Cause.Peer)
	}
	for _, p := range evt.Response.Queried {
		if p.Peer == cs.h.ID() { // don't add self.
			continue
		}
		if st := cs.PeerSet[evt.ID].GetState(p.Peer); st == qpeerset.PeerWaiting {
			cs.PeerSet[evt.ID].SetState(p.Peer, qpeerset.PeerQueried)
		} else {
			panic(fmt.Errorf("kademlia protocol error: tried to transition to the queried state from state %v", st))
		}
	}
	for _, p := range evt.Response.Unreachable {
		if p.Peer == cs.h.ID() { // don't add self.
			continue
		}

		if st := cs.PeerSet[evt.ID].GetState(p.Peer); st == qpeerset.PeerWaiting {
			cs.PeerSet[evt.ID].SetState(p.Peer, qpeerset.PeerUnreachable)
		} else {
			panic(fmt.Errorf("kademlia protocol error: tried to transition to the unreachable state from state %v", st))
		}
	}
}

func (cs *CommonState) trackLookupRequest(evt *kaddht.LookupEvent) {
	for _, p := range evt.Request.Waiting {
		cs.PeerSet[evt.ID].SetState(p.Peer, qpeerset.PeerWaiting)
	}
}

func (cs *CommonState) trackConnectionStart(p peer.ID) {
	cs.connectionsStartedLk.Lock()
	defer cs.connectionsStartedLk.Unlock()

	if _, found := cs.ConnectionsStarted[p]; found {
		return
	}

	cs.ConnectionsStarted[p] = time.Now()
}

func (cs *CommonState) trackConnectionEnd(p peer.ID, maddr ma.Multiaddr) {
	end := time.Now()

	cs.connectionsStartedLk.Lock()
	started, ok := cs.ConnectionsStarted[p]
	if !ok {
		cs.connectionsStartedLk.Unlock()
		return
	}
	delete(cs.ConnectionsStarted, p)
	cs.connectionsStartedLk.Unlock()

	cs.connectionsLk.Lock()
	cs.Connections = append(cs.Connections, &ConnectionSpan{
		RemotePeerID: p,
		Maddr:        maddr,
		Start:        started,
		End:          end,
	})
	cs.connectionsLk.Unlock()
}

func (cs *CommonState) filterDials() {
	cs.dialsLk.Lock()
	defer cs.dialsLk.Unlock()

	var relevantDials []*DialSpan
	for _, dial := range cs.Dials {
		if _, found := cs.RelevantPeers.Load(dial.RemotePeerID); found {
			relevantDials = append(relevantDials, dial)
		}
	}
	cs.Dials = relevantDials
}

func (cs *CommonState) filterConnections() {
	cs.connectionsLk.Lock()
	defer cs.connectionsLk.Unlock()

	var relevantConnections []*ConnectionSpan
	for _, conn := range cs.Connections {
		if _, found := cs.RelevantPeers.Load(conn.RemotePeerID); found {
			relevantConnections = append(relevantConnections, conn)
		}
	}
	cs.Connections = relevantConnections
}

////

func (cs *CommonState) Listen(network network.Network, multiaddr ma.Multiaddr) {
}

func (cs *CommonState) ListenClose(network network.Network, multiaddr ma.Multiaddr) {
}

func (cs *CommonState) Connected(network network.Network, conn network.Conn) {
	cs.trackConnectionEnd(conn.RemotePeer(), conn.RemoteMultiaddr())
}

func (cs *CommonState) Disconnected(network network.Network, conn network.Conn) {
}

////

func (cs *CommonState) DialStarted(trpt string, raddr ma.Multiaddr, p peer.ID, start time.Time) {
	cs.trackConnectionStart(p)
}

func (cs *CommonState) DialEnded(trpt string, raddr ma.Multiaddr, p peer.ID, start time.Time, end time.Time, err error) {
	d := &DialSpan{
		RemotePeerID: p,
		Maddr:        raddr,
		Start:        start,
		End:          end,
		Trpt:         trpt,
		Error:        err,
	}
	cs.dialsLk.Lock()
	defer cs.dialsLk.Unlock()

	cs.Dials = append(cs.Dials, d)
}
