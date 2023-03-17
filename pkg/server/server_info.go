package server

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/julienschmidt/httprouter"
)

type InfoResponse struct {
	PeerID    peer.ID
	BuildInfo *debug.BuildInfo
}

func (s *Server) info(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp := InfoResponse{
		PeerID:    s.host.ID(),
		BuildInfo: bi,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err = rw.Write(data); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
