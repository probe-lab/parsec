package main

import (
	"fmt"
	"time"

	"github.com/guseggert/clustertest/cluster/basic"
	"github.com/guseggert/clustertest/cluster/docker"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/dennis-tra/parsec/pkg/parsec"
	"github.com/dennis-tra/parsec/pkg/util"
)

// ScheduleCommand contains the crawl sub-command configuration.
var ScheduleCommand = &cli.Command{
	Name:   "schedule",
	Action: ScheduleAction,
	Flags:  []cli.Flag{},
}

// ScheduleAction is the function that is called when running `nebula crawl`.
func ScheduleAction(c *cli.Context) error {
	log.Infoln("Starting Parsec scheduler...")

	// Load configuration file
	//conf, err := config.Init(c)
	//if err != nil {
	//	return err
	//}

	// Acquire database handle
	//var dbc *db.Client
	//if !c.Bool("dry-run") {
	//	if dbc, err = db.InitClient(c.Context, conf); err != nil {
	//		return err
	//	}
	//}
	//_ = dbc

	log.Infoln("Initializing new cluster")
	cl, err := docker.NewCluster()
	if err != nil {
		return fmt.Errorf("new docker cluster: %w", err)
	}

	pc := parsec.NewCluster(basic.New(cl))

	log.Infoln("Initializing nodes")
	nodes, err := pc.NewNodes(2)
	if err != nil {
		return fmt.Errorf("new nodes: %w", err)
	}

	log.Infoln("sleeping...")
	time.Sleep(5 * time.Second)

	content, err := util.NewRandomContent()
	if err != nil {
		return fmt.Errorf("new random content: %w", err)
	}
	provide, err := nodes[0].Provide(content)
	if err != nil {
		return err
	}
	log.Infoln(provide)

	retrieval, err := nodes[1].Retrieve(content.CID, 1)
	if err != nil {
		return err
	}
	log.Infoln("TimeToFirstProviderRecord", retrieval.TimeToFirstProviderRecord())

	<-c.Context.Done()

	return nil
}
