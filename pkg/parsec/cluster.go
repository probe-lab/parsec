package parsec

import (
	"fmt"

	"github.com/guseggert/clustertest/cluster/basic"
)

type Cluster struct {
	*basic.Cluster
	Region       string
	InstanceType string
	ServerHost   string
	ServerPort   int
	FullRT       bool
}

func NewCluster(bc *basic.Cluster, region string, instanceType string, serverHost string, serverPort int, fullRT bool) *Cluster {
	return &Cluster{
		Cluster:      bc,
		Region:       region,
		InstanceType: instanceType,
		ServerHost:   serverHost,
		ServerPort:   serverPort,
		FullRT:       fullRT,
	}
}

func (c *Cluster) NewNodes(n int) ([]*Node, error) {
	clusterNodes, err := c.Cluster.NewNodes(n)
	if err != nil {
		return nil, err
	}

	parsecNodes := make([]*Node, len(clusterNodes))
	for i, cn := range clusterNodes {
		n, err := NewNode(c, cn.Context(c.Ctx), fmt.Sprintf("node-%d", i), c.ServerHost, c.ServerPort)
		if err != nil {
			return nil, fmt.Errorf("new parsec node: %w", err)
		}
		parsecNodes[i] = n
	}

	return parsecNodes, nil
}

func (c *Cluster) NewNode(num int) (*Node, error) {
	cn, err := c.Cluster.NewNode()
	if err != nil {
		return nil, fmt.Errorf("new cluster node: %w", err)
	}

	n, err := NewNode(c, cn.Context(c.Ctx), fmt.Sprintf("node-%d", num), c.ServerHost, c.ServerPort)
	if err != nil {
		return nil, fmt.Errorf("new parsec node: %w", err)
	}

	return n, err
}
