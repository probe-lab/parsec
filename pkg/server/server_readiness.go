package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

func (s *Server) readiness(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
	rw.WriteHeader(http.StatusOK)
}

func (c *Client) Readiness(ctx context.Context) error {
	endpoint := fmt.Sprintf("http://%s/readiness", c.addr)

	log.Infoln("GET", endpoint)
	req, err := http.Get(endpoint)
	if err != nil {
		return fmt.Errorf("create retrieve request: %w", err)
	}

	if req.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", req.StatusCode)
	}

	return nil
}
