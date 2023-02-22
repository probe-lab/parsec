package parsec

import (
	"fmt"

	"github.com/guseggert/clustertest/cluster/basic"
)

type Cluster struct {
	*basic.Cluster
}

func NewCluster(bc *basic.Cluster) *Cluster {
	return &Cluster{
		Cluster: bc,
	}
}

func (c *Cluster) NewNodes(n int) ([]*Node, error) {
	clusterNodes, err := c.Cluster.NewNodes(n)
	if err != nil {
		return nil, err
	}

	parsecNodes := make([]*Node, len(clusterNodes))
	for i, cn := range clusterNodes {
		n, err := NewNode(cn.Context(c.Ctx), fmt.Sprintf("node-%d", i), "localhost", 7070)
		if err != nil {
			return nil, fmt.Errorf("new parsec node: %w", err)
		}
		parsecNodes[i] = n
	}

	return parsecNodes, nil
}
