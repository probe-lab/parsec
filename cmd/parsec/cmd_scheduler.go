package main

import (
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

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

	pc := parsec.NewCluster(basic.New(cl).Context(c.Context))

	log.Infoln("Initializing nodes")
	nodes, err := pc.NewNodes(5)
	if err != nil {
		return fmt.Errorf("new nodes: %w", err)
	}

	errg := errgroup.Group{}
	for _, n := range nodes {
		n2 := n
		errg.Go(func() error {
			return n2.WaitForAPI(c.Context)
		})
	}
	if err = errg.Wait(); err != nil {
		return fmt.Errorf("errgroup: %w", err)
	}

	i := 0
	for {
		select {
		case <-c.Context.Done():
			return c.Err()
		default:
		}

		content, err := util.NewRandomContent()
		if err != nil {
			return fmt.Errorf("new random content: %w", err)
		}

		_, err = nodes[i].Provide(c.Context, content)
		if err != nil {
			return err
		}

		var wg sync.WaitGroup
		for j := 0; j < len(nodes); j++ {
			wg.Add(1)
			nidx := (i + j + 1) % len(nodes)

			go func() {
				defer wg.Done()
				retrieval, err := nodes[nidx].Retrieve(c.Context, content.CID, 1)
				if err != nil {
					log.WithError(err).Infoln("asdfsdf")
				} else {
					log.WithField("nodeID", nodes[nidx].ID()).WithField("dur", retrieval.TimeToFirstProviderRecord()).Infoln("Time to first provider record")
				}
			}()
		}
		wg.Wait()

		i += 1
		i %= len(nodes)
	}
}
