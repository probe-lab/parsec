package parsec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/ipfs/go-cid"

	"github.com/guseggert/clustertest/cluster"
	"github.com/guseggert/clustertest/cluster/basic"
	log "github.com/sirupsen/logrus"
)

type Node struct {
	*basic.Node

	ctx    context.Context
	client http.Client
	host   string
	port   int
}

func NewNode(n *basic.Node, host string, port int) (*Node, error) {
	log.Infoln("Reading parsec binary...")
	parsecBinary, err := os.ReadFile("parsec")
	if err != nil {
		return nil, fmt.Errorf("read parsec binary: %w", err)
	}

	log.Infoln("Sending parsec binary...")
	if err = n.SendFile("/parsec", bytes.NewReader(parsecBinary)); err != nil {
		return nil, fmt.Errorf("send parsec binary: %w", err)
	}

	log.Infoln("Setting parsec permissions...")
	proc, err := n.StartProc(cluster.StartProcRequest{
		Command: "chmod",
		Args:    []string{"+x", "/parsec"},
	})
	if err != nil {
		return nil, fmt.Errorf("setting parsec permissions: %w", err)
	}
	if _, err := proc.Wait(); err != nil {
		return nil, fmt.Errorf("set parsec permissions: %w", err)
	}

	log.Infoln("Starting parsec server...")
	proc, err = n.StartProc(cluster.StartProcRequest{
		Command: "/parsec",
		Args:    []string{"node"},
	})
	if err != nil {
		return nil, fmt.Errorf("start parsec node: %w", err)
	}
	go func() {
		procRes, err := proc.Wait()
		log.WithError(err).WithField("exitCode", procRes.ExitCode).Infoln("Parsec server exited")
	}()

	newTransport := http.DefaultTransport.(*http.Transport).Clone()
	newTransport.DialContext = n.Node.Dial
	httpClient := http.Client{Transport: newTransport}

	return &Node{
		Node:   n,
		ctx:    context.Background(),
		client: httpClient,
		host:   host,
		port:   port,
	}, nil
}

func (n *Node) Context(ctx context.Context) *Node {
	newN := *n
	newN.Ctx = ctx
	newN.Node = newN.Node.Context(ctx)
	return &newN
}

func (n *Node) Retrieve(c cid.Cid, count int) error {
	rr := &RetrieveRequest{
		Count: 1,
	}

	data, err := json.Marshal(rr)
	if err != nil {
		return fmt.Errorf("marshal retrieve request: %w", err)
	}

	res, err := n.client.Post(fmt.Sprintf("http://%s:%d/retrieve/%s", n.host, n.port, c.String()), "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	dat, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	_ = dat
	return nil
}

func (n *Node) Provide(c *Content) error {
	pr := &ProvideRequest{
		CID: c.raw,
	}

	data, err := json.Marshal(pr)
	if err != nil {
		return fmt.Errorf("marshal provide request: %w", err)
	}

	res, err := n.client.Post(fmt.Sprintf("http://%s:%d/provide", n.host, n.port), "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	dat, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	_ = dat

	return nil
}
