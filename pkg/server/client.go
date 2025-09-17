package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ipfs/go-cid"
	log "github.com/sirupsen/logrus"

	"github.com/probe-lab/parsec/pkg/util"
)

type Client struct {
	client      *http.Client
	addr        string
	schedulerID string
	region      string
}

func NewClient(host string, port int16, schedulerID string, region string) *Client {
	return &Client{
		schedulerID: schedulerID,
		addr:        fmt.Sprintf("%s:%d", host, port),
		client:      http.DefaultClient,
		region:      region,
	}
}

type Measurement struct {
	Step     string
	Duration time.Duration
	Error    string
}

type ProvideResponse struct {
	CID              string
	Measurements     []*Measurement
	RoutingTableSize int
}

func (c *Client) Provide(ctx context.Context, content *util.Content) (*ProvideResponse, error) {
	pr := &ProvideRequest{
		Content: content.Raw,
	}

	data, err := json.Marshal(pr)
	if err != nil {
		return nil, fmt.Errorf("marshal provide request: %w", err)
	}

	endpoint := fmt.Sprintf("http://%s/provide", c.addr)
	log.WithField("cid", content.CID.String()).WithField("region", c.region).Infoln("POST", endpoint)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create retrieve request: %w", err)
	}
	req = req.WithContext(ctx)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(headerSchedulerID, c.schedulerID)

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

func (c *Client) Readiness(ctx context.Context) error {
	endpoint := fmt.Sprintf("http://%s/readiness", c.addr)

	log.WithField("region", c.region).Infoln("GET", endpoint)
	req, err := http.Get(endpoint)
	if err != nil {
		return fmt.Errorf("create retrieve request: %w", err)
	}

	if req.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", req.StatusCode)
	}

	return nil
}

type RetrievalResponse struct {
	CID              string
	Measurements     []*Measurement
	RoutingTableSize int
}

func (c *Client) Retrieve(ctx context.Context, content cid.Cid) (*RetrievalResponse, error) {
	rr := &RetrieveRequest{}

	data, err := json.Marshal(rr)
	if err != nil {
		return nil, fmt.Errorf("marshal retrieval request: %w", err)
	}

	endpoint := fmt.Sprintf("http://%s/retrieve/%s", c.addr, content.String())

	log.WithField("region", c.region).Infoln("POST", endpoint)
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
