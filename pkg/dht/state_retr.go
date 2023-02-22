package dht

import (
	"context"
	"sync"

	"github.com/ipfs/go-cid"

	pb "github.com/libp2p/go-libp2p-kad-dht/pb"
	log "github.com/sirupsen/logrus"

	"github.com/dennis-tra/parsec/pkg/wrap"
)

type RetrievalState struct {
	*CommonState

	getProvidersLk sync.RWMutex
	GetProviders   []*GetProvidersSpan
}

func NewRetrievalState(h *Host, c cid.Cid) *RetrievalState {
	return &RetrievalState{
		CommonState:    NewCommonState(h, c),
		getProvidersLk: sync.RWMutex{},
		GetProviders:   []*GetProvidersSpan{},
	}
}

func (rs *RetrievalState) Register(ctx context.Context) context.Context {
	ctx, rpcEvents := wrap.RegisterForRPCEvents(ctx)
	go rs.consumeRPCEvents(rpcEvents)

	return rs.CommonState.Register(ctx)
}

func (rs *RetrievalState) consumeRPCEvents(rpcEvents <-chan interface{}) {
	for event := range rpcEvents {
		switch evt := event.(type) {
		case *wrap.RPCSendRequestStartedEvent:
		case *wrap.RPCSendRequestEndedEvent:
			switch evt.Request.Type {
			case pb.Message_GET_PROVIDERS:
				rs.trackGetProvidersRequest(evt)
			default:
				log.Warn(evt)
			}
		case *wrap.RPCSendMessageStartedEvent:
		case *wrap.RPCSendMessageEndedEvent:
			switch evt.Message.Type {
			default:
				log.Warn(evt)
			}
		default:
			log.Warn(event)
		}
	}
}

func (rs *RetrievalState) trackGetProvidersRequest(evt *wrap.RPCSendRequestEndedEvent) {
	gps := &GetProvidersSpan{
		QueryID:      evt.QueryID,
		RemotePeerID: evt.RemotePeer,
		Start:        evt.StartedAt,
		End:          evt.EndedAt,
		Error:        evt.Error,
	}
	if evt.Response != nil {
		gps.Providers = pb.PBPeersToPeerInfos(evt.Response.ProviderPeers)
		gps.CloserPeers = pb.PBPeersToPeerInfos(evt.Response.CloserPeers)
	}

	rs.getProvidersLk.Lock()
	defer rs.getProvidersLk.Unlock()

	rs.GetProviders = append(rs.GetProviders, gps)
	rs.storeRelevantPeer(evt.RemotePeer)
	for _, p := range gps.Providers {
		rs.storeRelevantPeer(p.ID)
	}
}
