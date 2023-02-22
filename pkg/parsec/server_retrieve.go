package parsec

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/dennis-tra/parsec/pkg/dht"
	log "github.com/sirupsen/logrus"

	"github.com/ipfs/go-cid"

	"github.com/julienschmidt/httprouter"
)

type RetrieveRequest struct {
	Count int
}

func (s *Server) retrieve(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var rr RetrieveRequest
	data, err := io.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = json.Unmarshal(data, &rr); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	c, err := cid.Decode(params.ByName("cid"))
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	state := dht.NewRetrievalState(s.host, c)
	ctx := state.Register(r.Context())
	log.WithField("cid", c).Infoln("Start finding providers")
	state.Start = time.Now()
	for provider := range s.host.DHT.FindProvidersAsync(ctx, c, rr.Count) {
		log.WithField("cid", c).WithField("providerID", provider.ID).Infoln("Found Provider")
	}
	state.End = time.Now()
	log.WithField("cid", c).Infoln("Start finding providers")

	state.Unregister()

	resp := retrievalResponseFromState(state)

	data, err = json.Marshal(resp)
	if err != nil {
		rw.Write([]byte(err.Error()))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = rw.Write(data); err != nil {
		rw.Write([]byte(err.Error()))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type RetrievalResponse struct {
	CommonResponse
	GetProviders []GetProvider
}

type GetProvider struct {
	QueryID      uuid.UUID
	RemotePeerID peer.ID
	Start        time.Time
	End          time.Time
	CloserPeers  []peer.ID
	Providers    []peer.ID
	Error        string
}

func retrievalResponseFromState(state *dht.RetrievalState) *RetrievalResponse {
	resp := RetrievalResponse{
		CommonResponse: CommonResponseFromState(state.CommonState),
		GetProviders:   make([]GetProvider, len(state.GetProviders)),
	}

	for i, gp := range state.GetProviders {
		cps := make([]peer.ID, len(gp.CloserPeers))
		for i, cp := range gp.CloserPeers {
			cps[i] = cp.ID
		}

		providers := make([]peer.ID, len(gp.Providers))
		for i, provider := range gp.Providers {
			providers[i] = provider.ID
		}

		errStr := ""
		if gp.Error != nil {
			errStr = gp.Error.Error()
		}

		resp.GetProviders[i] = GetProvider{
			QueryID:      gp.QueryID,
			RemotePeerID: gp.RemotePeerID,
			Start:        gp.Start,
			End:          gp.End,
			CloserPeers:  cps,
			Providers:    providers,
			Error:        errStr,
		}
	}

	return &resp
}

func (r RetrievalResponse) TimeToFirstProviderRecord() time.Duration {
	firstProviderTime := time.Time{}
	for _, gp := range r.GetProviders {
		if len(gp.Providers) > 0 && (gp.End.Before(firstProviderTime) || firstProviderTime.IsZero()) {
			firstProviderTime = gp.End
		}
	}

	if firstProviderTime.IsZero() {
		return 0
	}

	return firstProviderTime.Sub(r.Start)
}
