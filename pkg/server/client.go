package server

import (
	"fmt"
	"net/http"
)

type Client struct {
	client *http.Client
	addr   string
}

func NewClient(host string, port int16) *Client {
	return &Client{
		addr:   fmt.Sprintf("%s:%d", host, port),
		client: http.DefaultClient,
	}
}
