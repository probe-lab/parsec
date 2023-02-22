package parsec

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type InfoResponse struct{}

func (s *Server) info(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
	resp := InfoResponse{}
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
