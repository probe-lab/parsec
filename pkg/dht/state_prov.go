package dht

import (
	"context"
	"sync"

	"github.com/dennis-tra/parsec/pkg/wrap"
	"github.com/ipfs/go-cid"
	pb "github.com/libp2p/go-libp2p-kad-dht/pb"
	log "github.com/sirupsen/logrus"
)

type ProvideState struct {
	*CommonState

	findNodesLk sync.RWMutex
	FindNodes   []*FindNodesSpan

	addProvidersLk sync.RWMutex
	AddProviders   []*AddProvidersSpan
}

func NewProvideState(h *Host, c cid.Cid) *ProvideState {
	return &ProvideState{
		CommonState:    NewCommonState(h, c),
		findNodesLk:    sync.RWMutex{},
		FindNodes:      []*FindNodesSpan{},
		addProvidersLk: sync.RWMutex{},
		AddProviders:   []*AddProvidersSpan{},
	}
}

func (ps *ProvideState) Register(ctx context.Context) context.Context {
	ctx, rpcEvents := wrap.RegisterForRPCEvents(ctx)
	go ps.consumeRPCEvents(rpcEvents)

	return ps.CommonState.Register(ctx)
}

func (ps *ProvideState) consumeRPCEvents(rpcEvents <-chan interface{}) {
	for event := range rpcEvents {
		switch evt := event.(type) {
		case *wrap.RPCSendRequestStartedEvent:
		case *wrap.RPCSendRequestEndedEvent:
			switch evt.Request.Type {
			case pb.Message_FIND_NODE:
				ps.trackFindNodeRequest(evt)
			default:
				log.Warn(evt)
			}
		case *wrap.RPCSendMessageStartedEvent:
		case *wrap.RPCSendMessageEndedEvent:
			switch evt.Message.Type {
			case pb.Message_ADD_PROVIDER:
				ps.trackAddProvidersRequest(evt)
			default:
				log.Warn(evt)
			}
		default:
			log.Warn(event)
		}
	}
}

func (ps *ProvideState) trackFindNodeRequest(evt *wrap.RPCSendRequestEndedEvent) {
	log.Infoln("TRACK FIND_NODE")
	fns := &FindNodesSpan{
		QueryID:      evt.QueryID,
		RemotePeerID: evt.RemotePeer,
		Start:        evt.StartedAt,
		End:          evt.EndedAt,
		Error:        evt.Error,
	}
	if evt.Response != nil {
		fns.CloserPeers = pb.PBPeersToPeerInfos(evt.Response.CloserPeers)
	}

	ps.findNodesLk.Lock()
	defer ps.findNodesLk.Unlock()

	ps.FindNodes = append(ps.FindNodes, fns)
	ps.storeRelevantPeer(evt.RemotePeer)
	for _, p := range fns.CloserPeers {
		ps.storeRelevantPeer(p.ID)
	}
}

func (ps *ProvideState) trackAddProvidersRequest(evt *wrap.RPCSendMessageEndedEvent) {
	log.Infoln("TRACK ADD_PROVIDERS")
	aps := &AddProvidersSpan{
		QueryID:       evt.QueryID,
		CID:           ps.CID,
		RemotePeerID:  evt.RemotePeer,
		Start:         evt.StartedAt,
		End:           evt.EndedAt,
		ProviderAddrs: pb.PBPeersToPeerInfos(evt.Message.ProviderPeers)[0].Addrs,
		Error:         evt.Error,
	}

	ps.addProvidersLk.Lock()
	defer ps.addProvidersLk.Unlock()

	ps.AddProviders = append(ps.AddProviders, aps)
	ps.storeRelevantPeer(evt.RemotePeer)
}
