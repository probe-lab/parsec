package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

func (c *Client) Info(ctx context.Context) (*InfoResponse, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/info", c.addr), nil)
	if err != nil {
		return nil, fmt.Errorf("create info request: %w", err)
	}
	req = req.WithContext(ctx)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get info: %w", err)
	}

	dat, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read provide response: %w", err)
	}

	info := InfoResponse{}
	if err = json.Unmarshal(dat, &info); err != nil {
		return nil, fmt.Errorf("unmarshal info response: %w", err)
	}

	return &info, nil
}
