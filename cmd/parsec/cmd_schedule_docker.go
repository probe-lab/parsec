package main

import (
	"fmt"
	"strconv"

	"github.com/dennis-tra/parsec/pkg/config"
	"github.com/dennis-tra/parsec/pkg/parsec"
	"github.com/guseggert/clustertest/cluster/basic"
	"github.com/guseggert/clustertest/cluster/docker"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var ScheduleDockerCommand = &cli.Command{
	Name:   "docker",
	Action: ScheduleDockerAction,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "nodes",
			Usage:       "The number of nodes to spawn",
			EnvVars:     []string{"PARSEC_SCHEDULE_DOCKER_NODES"},
			DefaultText: strconv.Itoa(config.DefaultScheduleDockerConfig.Nodes),
			Value:       config.DefaultScheduleDockerConfig.Nodes,
		},
	},
}

func ScheduleDockerAction(c *cli.Context) error {
	log.Infoln("Starting Parsec docker scheduler...")

	conf := config.DefaultScheduleDockerConfig.Apply(c)

	cl, err := docker.NewCluster()
	if err != nil {
		return fmt.Errorf("new docker cluster: %w", err)
	}

	pc := parsec.NewCluster(basic.New(cl).Context(c.Context), conf.ServerHost, conf.ServerPort)

	log.Infoln("Initializing docker nodes")
	nodes, err := pc.NewNodes(conf.Nodes)
	if err != nil {
		return fmt.Errorf("new docker nodes: %w", err)
	}

	return ScheduleAction(c, nodes)
}
