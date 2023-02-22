package parsec

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/dennis-tra/parsec/pkg/dht"
	"github.com/dennis-tra/parsec/pkg/util"
	"github.com/julienschmidt/httprouter"
)

import (
	log "github.com/sirupsen/logrus"
)

type ProvideRequest struct {
	Content []byte
}

func (s *Server) provide(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var pr ProvideRequest
	data, err := io.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = json.Unmarshal(data, &pr); err != nil {
		rw.Write([]byte(err.Error()))
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	content, err := util.ContentFrom(pr.Content)
	if err != nil {
		rw.Write([]byte(err.Error()))
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	state := dht.NewProvideState(s.host, content.CID)
	ctx := state.Register(r.Context())

	log.WithField("cid", content.CID.String()).Infoln("Start providing content...")
	state.Start = time.Now()
	err = s.host.DHT.Provide(ctx, content.CID, true)
	state.End = time.Now()
	log.WithField("cid", content.CID.String()).Infoln("Done providing content...")

	state.Unregister()

	resp := provideResponseFromState(state)
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

type ProvideResponse struct {
	CommonResponse

	FindNodes    []FindNode
	AddProviders []AddProvider
}

type FindNode struct {
	QueryID      uuid.UUID
	RemotePeerID peer.ID
	Start        time.Time
	End          time.Time
	CloserPeers  []peer.ID
	Error        string
}

type AddProvider struct {
	QueryID      uuid.UUID
	RemotePeerID peer.ID
	Start        time.Time
	End          time.Time
	Error        string
}

func provideResponseFromState(state *dht.ProvideState) *ProvideResponse {
	resp := ProvideResponse{
		CommonResponse: CommonResponseFromState(state.CommonState),
		FindNodes:      make([]FindNode, len(state.FindNodes)),
		AddProviders:   make([]AddProvider, len(state.AddProviders)),
	}

	for i, fn := range state.FindNodes {
		cps := make([]peer.ID, len(fn.CloserPeers))
		for i, cp := range fn.CloserPeers {
			cps[i] = cp.ID
		}

		errStr := ""
		if fn.Error != nil {
			errStr = fn.Error.Error()
		}

		resp.FindNodes[i] = FindNode{
			QueryID:      fn.QueryID,
			RemotePeerID: fn.RemotePeerID,
			Start:        fn.Start,
			End:          fn.End,
			CloserPeers:  cps,
			Error:        errStr,
		}
	}

	for i, ap := range state.AddProviders {
		errStr := ""
		if ap.Error != nil {
			errStr = ap.Error.Error()
		}

		resp.AddProviders[i] = AddProvider{
			QueryID:      ap.QueryID,
			RemotePeerID: ap.RemotePeerID,
			Start:        ap.Start,
			End:          ap.End,
			Error:        errStr,
		}
	}

	return &resp
}
