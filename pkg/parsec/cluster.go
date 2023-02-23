package parsec

import (
	"fmt"

	"github.com/guseggert/clustertest/cluster/basic"
)

type Cluster struct {
	*basic.Cluster
	ServerHost string
	ServerPort int
}

func NewCluster(bc *basic.Cluster, serverHost string, serverPort int) *Cluster {
	return &Cluster{
		Cluster:    bc,
		ServerHost: serverHost,
		ServerPort: serverPort,
	}
}

func (c *Cluster) NewNodes(n int) ([]*Node, error) {
	clusterNodes, err := c.Cluster.NewNodes(n)
	if err != nil {
		return nil, err
	}

	parsecNodes := make([]*Node, len(clusterNodes))
	for i, cn := range clusterNodes {
		n, err := NewNode(cn.Context(c.Ctx), fmt.Sprintf("node-%d", i), c.ServerHost, c.ServerPort)
		if err != nil {
			return nil, fmt.Errorf("new parsec node: %w", err)
		}
		parsecNodes[i] = n
	}

	return parsecNodes, nil
}

func (c *Cluster) NewNode() (*Node, error) {
	nodes, err := c.NewNodes(1)
	if err != nil {
		return nil, err
	}
	if len(nodes) != 1 {
		return nil, fmt.Errorf("expected 1 node, got %d", len(nodes))
	}
	return nodes[0], err
}
