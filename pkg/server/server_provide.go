package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

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

	log.WithField("cid", content.CID.String()).Infoln("Start providing content...")
	timeoutCtx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()

	start := time.Now()
	err = s.host.DHT.Provide(timeoutCtx, content.CID, true)
	end := time.Now()

	log.WithField("cid", content.CID.String()).Infoln("Done providing content...")

	resp := ProvideResponse{
		CID:              content.CID.String(),
		Duration:         end.Sub(start),
		RoutingTableSize: s.host.DHT.RoutingTable().Size(),
	}

	if err != nil {
		resp.Error = err.Error()
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

type ProvideResponse struct {
	CID              string
	Duration         time.Duration
	Error            string
	RoutingTableSize int
}
