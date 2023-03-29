package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/julienschmidt/httprouter"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"

	"github.com/dennis-tra/parsec/pkg/dht"
	"github.com/dennis-tra/parsec/pkg/util"
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

	resp := RetrievalResponse{
		CID:              c.String(),
		RoutingTableSize: dht.RoutingTableSize(s.host.DHT),
	}
	logEntry := log.WithField("cid", c.String()).WithField("rtSize", resp.RoutingTableSize)

	logEntry.Infoln("Start finding providers")

	// here's where the magic happens
	start := time.Now()
	provider := <-s.host.DHT.FindProvidersAsync(ctx, c, 1)
	resp.Duration = time.Since(start)

	latencies.WithLabelValues("retrieval_ttfpr", strconv.FormatBool(ctx.Err() == nil), r.Header.Get(headerSchedulerID)).Observe(resp.Duration.Seconds())

	logEntry = logEntry.WithField("dur", resp.Duration.Seconds())

	if errors.Is(provider.ID.Validate(), peer.ErrEmptyPeerID) {
		resp.Error = "not found"
		logEntry.Infoln("Didn't find provider")
	} else {
		s.host.Network().ClosePeer(provider.ID)
		s.host.Peerstore().RemovePeer(provider.ID)
		s.host.Peerstore().ClearAddrs(provider.ID)
		logEntry.WithField("provider", util.FmtPeerID(provider.ID)).Infoln("Found provider")
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

func (c *Client) Retrieve(ctx context.Context, content cid.Cid) (*RetrievalResponse, error) {
	rr := &RetrieveRequest{}

	data, err := json.Marshal(rr)
	if err != nil {
		return nil, fmt.Errorf("marshal retrieval request: %w", err)
	}

	endpoint := fmt.Sprintf("http://%s/retrieve/%s", c.addr, content.String())

	log.Infoln("POST", endpoint)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create retrieve request: %w", err)
	}
	req = req.WithContext(ctx)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(headerSchedulerID, c.schedulerID)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post retrieval request: %w", err)
	}

	dat, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read retrieval response: %w", err)
	}

	retrieval := RetrievalResponse{}
	if err = json.Unmarshal(dat, &retrieval); err != nil {
		return nil, fmt.Errorf("unmarshal retrieval response: %w", err)
	}

	return &retrieval, nil
}

type RetrievalResponse struct {
	CID              string
	Duration         time.Duration
	RoutingTableSize int
	Error            string
}
