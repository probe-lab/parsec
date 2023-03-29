package server

import (
	"fmt"
	"net/http"
)

type Client struct {
	client      *http.Client
	addr        string
	schedulerID string
}

func NewClient(host string, port int16, schedulerID string) *Client {
	return &Client{
		schedulerID: schedulerID,
		addr:        fmt.Sprintf("%s:%d", host, port),
		client:      http.DefaultClient,
	}
}
