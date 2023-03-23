package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/dennis-tra/parsec/pkg/dht"
	"github.com/dennis-tra/parsec/pkg/util"
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

	latencies.WithLabelValues("provide_duration", strconv.FormatBool(err == nil)).Observe(end.Sub(start).Seconds())
	log.WithField("cid", content.CID.String()).Infoln("Done providing content...")

	resp := ProvideResponse{
		CID:              content.CID.String(),
		Duration:         end.Sub(start),
		RoutingTableSize: dht.RoutingTableSize(s.host.DHT),
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

func (c *Client) Provide(ctx context.Context, content *util.Content) (*ProvideResponse, error) {
	pr := &ProvideRequest{
		Content: content.Raw,
	}

	data, err := json.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("marshal provide request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/provide", c.addr), bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create retrieve request: %w", err)
	}
	req = req.WithContext(ctx)

	req.Header.Add("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("start provide: %w", err)
	}

	dat, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read provide response: %w", err)
	}

	provide := ProvideResponse{}
	if err = json.Unmarshal(dat, &provide); err != nil {
		return nil, fmt.Errorf("unmarshal provide response: %w", err)
	}

	return &provide, nil
}

type ProvideResponse struct {
	CID              string
	Duration         time.Duration
	Error            string
	RoutingTableSize int
}
