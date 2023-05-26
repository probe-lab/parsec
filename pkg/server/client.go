package server

import (
	"fmt"
	"net/http"

	"github.com/dennis-tra/parsec/pkg/config"
)

type Client struct {
	client      *http.Client
	addr        string
	schedulerID string
	routing     config.Routing
}

func NewClient(host string, port int16, schedulerID string, routing config.Routing) *Client {
	return &Client{
		schedulerID: schedulerID,
		addr:        fmt.Sprintf("%s:%d", host, port),
		client:      http.DefaultClient,
		routing:     routing,
	}
}
