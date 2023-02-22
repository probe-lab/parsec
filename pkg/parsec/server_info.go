package parsec

import (
	"net/http"
	"runtime/debug"

	"github.com/julienschmidt/httprouter"
)

type InfoResponse struct{}

func (s *Server) info(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = bi
}
