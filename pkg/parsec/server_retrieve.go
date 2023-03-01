package parsec

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/dennis-tra/parsec/pkg/util"

	"github.com/ipfs/go-cid"
	"github.com/julienschmidt/httprouter"
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
	start := time.Now()
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	provider, ok := <-s.host.DHT.FindProvidersAsync(timeoutCtx, c, 1)
	end := time.Now()

	resp := RetrievalResponse{
		Duration: end.Sub(start),
	}

	if ok {
		resp.Error = "not found"
		log.WithField("cid", c.String()).Infoln("Could not find providers")
	} else {
		log.WithField("cid", c.String()).WithField("provider", util.FmtPeerID(provider.ID)).Infoln("Done finding providers")
	}

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
	Duration time.Duration
	Error    string
}
