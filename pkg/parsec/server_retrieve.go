package parsec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/julienschmidt/httprouter"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
)

type RetrieveRequest struct{}

func (s *Server) retrieve(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
	ctx := r.Context()
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

	log.WithField("cid", c.String()).Infoln("Start finding providers")

	provider, duration, err := s.measureRetrieval(ctx, c)
	resp := RetrievalResponse{
		Duration:         duration,
		RoutingTableSize: s.host.DHT.RoutingTable().Size(),
	}
	if err != nil {
		resp.Error = err.Error()
	} else {
		s.host.Network().ClosePeer(provider)
		s.host.Peerstore().RemovePeer(provider)
	}

	log.WithField("cid", c.String()).WithError(err).Infoln("Done finding providers")

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

func (s *Server) measureRetrieval(ctx context.Context, c cid.Cid) (peer.ID, time.Duration, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	start := time.Now()
	for provider := range s.host.DHT.FindProvidersAsync(timeoutCtx, c, 1) {
		return provider.ID, time.Since(start), nil
	}
	return "", time.Since(start), fmt.Errorf("not found")
}

type RetrievalResponse struct {
	Duration         time.Duration
	RoutingTableSize int
	Error            string
}
