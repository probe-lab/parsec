package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"

	"github.com/dennis-tra/parsec/pkg/config"
	"github.com/dennis-tra/parsec/pkg/db"
	"github.com/dennis-tra/parsec/pkg/server"
	"github.com/dennis-tra/parsec/pkg/util"
)

var SchedulerCommand = &cli.Command{
	Name: "scheduler",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "fleets",
			Usage:       "The fleets to experiment with",
			EnvVars:     []string{"PARSEC_SCHEDULER_FLEETS"},
			DefaultText: config.Scheduler.Fleets.String(),
			Value:       config.Scheduler.Fleets,
			Destination: config.Scheduler.Fleets,
		},
	},
	Action: SchedulerAction,
}

func SchedulerAction(c *cli.Context) error {
	log.Infoln("Starting Parsec scheduler...")

	// Acquire database handle
	dbc := db.NewDummyClient()
	var err error
	if !c.Bool("dry-run") {
		if dbc, err = db.InitDBClient(c.Context, config.Global); err != nil {
			return fmt.Errorf("init db client: %w", err)
		}
	}

	dbScheduler, err := dbc.InsertScheduler(c.Context, config.Scheduler.Fleets.Value())
	if err != nil {
		return fmt.Errorf("insert scheduler: %w", err)
	}

	provNodeIdx := 0
	for {
		// If context was cancelled stop here
		select {
		case <-c.Context.Done():
			return c.Context.Err()
		default:
		}

		// Get all dbNodes from database
		dbNodes, err := dbc.GetNodes(c.Context, config.Scheduler.Fleets.Value())
		if err != nil {
			return fmt.Errorf("get nodes: %w", err)
		}

		if len(dbNodes) < 2 {
			log.WithField("fleets", config.Scheduler.Fleets.Value()).Infoln("Less than two nodes in database. Waiting 10s and then trying again...")
			select {
			case <-time.After(10 * time.Second):
				continue
			case <-c.Context.Done():
				return c.Err()
			}
		}

		activeNodes.Set(float64(len(dbNodes)))

		clients := make([]*server.Client, len(dbNodes))
		for i, node := range dbNodes {
			client := server.NewClient(node.IPAddress, node.ServerPort, strings.Join(config.Scheduler.Fleets.Value(), ","))

			if err = client.Readiness(c.Context); err != nil {
				log.WithField("nodeID", node.ID).WithError(err).Warnln("Node not ready")
				if err := dbc.UpdateOfflineSince(c.Context, node); err != nil {
					log.WithField("nodeID", node.ID).WithError(err).Warnln("Couldn't put node offline")
				}
				continue
			}

			clients[i] = client
		}

		if len(clients) == 0 {
			log.WithField("fleets", config.Scheduler.Fleets.Value()).Infoln("Less than two nodes ready. Waiting 10s and then trying again...")
			select {
			case <-time.After(10 * time.Second):
				continue
			case <-c.Context.Done():
				return c.Err()
			}
		}

		// If nodes leave the network
		provNodeIdx %= len(dbNodes)

		providerNode := dbNodes[provNodeIdx]
		providerClient := clients[provNodeIdx]

		content, err := util.NewRandomContent()
		if err != nil {
			return fmt.Errorf("new random content: %w", err)
		}

		provide, err := providerClient.Provide(c.Context, content)
		issuedProvides.WithLabelValues(strconv.FormatBool(err == nil)).Inc()
		if err != nil {
			log.WithField("nodeID", providerNode.ID).WithError(err).Warnln("Failed to provide record")
			if err := dbc.UpdateOfflineSince(c.Context, providerNode); err != nil {
				log.WithField("nodeID", providerNode.ID).WithError(err).Warnln("Couldn't put node offline")
			}
			continue
		}

		if _, err := dbc.InsertProvide(c.Context, providerNode.ID, provide.CID, provide.Duration.Seconds(), provide.RoutingTableSize, provide.Error, dbScheduler.ID); err != nil {
			return fmt.Errorf("insert provide: %w", err)
		}

		// let everyone take a breath
		time.Sleep(10 * time.Second)

		// Loop through remaining nodes (len(nodes) - 1)
		errg, errCtx := errgroup.WithContext(c.Context)
		for i := 0; i < len(dbNodes)-1; i++ {

			// Start at current provNodeIdx + 1 and roll over after len(nodes) was reached
			idx := (provNodeIdx + 1 + i) % len(dbNodes)

			retrievalNode := dbNodes[idx]
			retrievalClient := clients[idx]

			errg.Go(func() error {
				retrieval, err := retrievalClient.Retrieve(errCtx, content.CID)
				issuedRetrievals.WithLabelValues(strconv.FormatBool(err == nil)).Inc()
				if err != nil {
					log.WithField("nodeID", retrievalNode.ID).WithError(err).Warnln("Failed to retrieve record")
					if err := dbc.UpdateOfflineSince(c.Context, retrievalNode); err != nil {
						log.WithField("nodeID", retrievalNode.ID).WithError(err).Warnln("Couldn't put retrieval node offline")
					}
					return nil
				}

				if _, err := dbc.InsertRetrieval(errCtx, retrievalNode.ID, retrieval.CID, retrieval.Duration.Seconds(), retrieval.RoutingTableSize, retrieval.Error, dbScheduler.ID); err != nil {
					return fmt.Errorf("insert retrieval: %w", err)
				}

				return nil
			})
		}
		if err = errg.Wait(); err != nil {
			return fmt.Errorf("waitgroup retrieve: %w", err)
		}

		provNodeIdx += 1
		provNodeIdx %= len(dbNodes)
	}
}
