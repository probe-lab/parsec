package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"

	"github.com/probe-lab/parsec/pkg/db"
	"github.com/probe-lab/parsec/pkg/server"
	"github.com/probe-lab/parsec/pkg/util"
)

var schedulerConfig = struct {
	Fleets   *cli.StringSlice
	Interval time.Duration
	Retries  int
}{
	Fleets:   cli.NewStringSlice(),
	Interval: time.Minute,
	Retries:  1,
}

var SchedulerCommand = &cli.Command{
	Name: "scheduler",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "fleets",
			Usage:       "The fleets to experiment with",
			EnvVars:     []string{"PARSEC_SCHEDULER_FLEETS"},
			DefaultText: schedulerConfig.Fleets.String(),
			Value:       schedulerConfig.Fleets,
			Destination: schedulerConfig.Fleets,
		},
		&cli.DurationFlag{
			Name:        "interval",
			Usage:       "The interval between two consecutive runs of the scheduler",
			EnvVars:     []string{"PARSEC_SCHEDULER_INTERVAL"},
			DefaultText: schedulerConfig.Interval.String(),
			Value:       schedulerConfig.Interval,
			Destination: &schedulerConfig.Interval,
		},
		&cli.IntFlag{
			Name:        "probes",
			Usage:       "how often many retrievals should be performed",
			EnvVars:     []string{"PARSEC_SCHEDULER_PROBES"},
			DefaultText: strconv.Itoa(schedulerConfig.Retries),
			Value:       schedulerConfig.Retries,
			Destination: &schedulerConfig.Retries,
		},
	},
	Before: schedulerBefore,
	Action: schedulerAction,
}

func schedulerBefore(c *cli.Context) error {
	prometheus.MustRegister(activeNodes)
	prometheus.MustRegister(issuedProvides)
	prometheus.MustRegister(issuedRetrievals)
	return nil
}

func schedulerAction(c *cli.Context) error {
	log.Infoln("Starting Parsec scheduler...")

	// Acquire database handle
	dbc := db.NewDummyClient()
	var err error
	if !c.Bool("dry-run") {
		if dbc, err = db.InitPGClient(c.Context, pgConfig()); err != nil {
			return fmt.Errorf("init db client: %w", err)
		}
	}

	dbScheduler, err := dbc.InsertScheduler(c.Context, schedulerConfig.Fleets.Value())
	if err != nil {
		return fmt.Errorf("insert scheduler: %w", err)
	}

	throttle := time.NewTimer(0)
	provNodeIdx := 0

outer:
	for {
		// If context was canceled, stop here
		select {
		case <-throttle.C:
			throttle.Reset(schedulerConfig.Interval)
		case <-c.Context.Done():
			return c.Err()
		}
		log.Infoln("Starting a new round of scheduling...")

		// Get all dbNodes from database
		dbNodes, err := dbc.GetNodes(c.Context, schedulerConfig.Fleets.Value())
		if err != nil {
			return fmt.Errorf("get nodes: %w", err)
		}

		if len(dbNodes) < 2 {
			log.WithFields(log.Fields{
				"fleets": schedulerConfig.Fleets.Value(),
				"nodes":  len(dbNodes),
			}).Infof("Fewer than two nodes in database. Waiting %s and then trying again...", schedulerConfig.Interval)
			continue
		}

		activeNodes.Set(float64(len(dbNodes)))

		var clients []*server.Client
		for _, node := range dbNodes {
			client := server.NewClient(node.IPAddress, node.ServerPort, strings.Join(schedulerConfig.Fleets.Value(), ","))

			if err = client.Readiness(c.Context); err != nil {
				log.WithField("nodeID", node.ID).WithError(err).Warnln("Node not ready")
				if err := dbc.UpdateOfflineSince(c.Context, node); err != nil {
					log.WithField("nodeID", node.ID).WithError(err).Warnln("Couldn't put node offline")
				}
				continue
			}

			clients = append(clients, client)
		}

		if len(clients) < 2 {
			log.WithField("fleets", schedulerConfig.Fleets.Value()).Infof("Fewer than two nodes ready. Waiting %s and then trying again...", schedulerConfig.Interval)
			continue
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

		nonNilErrs := 0
		for _, measurement := range provide.Measurements {
			if measurement.Error == "" {
				continue
			}

			nonNilErrs += 1

			if strings.Contains(measurement.Error, "Too Many Requests") {
				log.WithField("nodeID", providerNode.ID).Warnln("Failed to provide record due to too many requests. Trying again in a bit...")
				continue outer
			}
		}

		for _, measurement := range provide.Measurements {
			if _, err := dbc.InsertProvide(c.Context, providerNode.ID, provide.CID, measurement.Duration.Seconds(), provide.RoutingTableSize, measurement.Error, dbScheduler.ID, measurement.Step); err != nil {
				return fmt.Errorf("insert provide: %w", err)
			}
		}

		if nonNilErrs > 0 {
			err := ""
			for _, measurement := range provide.Measurements {
				if measurement.Error != "" {
					err = measurement.Error
					break
				}
			}
			log.WithField("err", err).Infoln("Failed to provide content")
			continue
		}

		// Loop through remaining nodes (len(nodes) - 1)
		errg, errCtx := errgroup.WithContext(c.Context)
		for i := 0; i < len(dbNodes)-1; i++ {

			// Start at current provNodeIdx + 1 and roll over after len(nodes) was reached
			idx := (provNodeIdx + 1 + i) % len(dbNodes)

			retrievalNode := dbNodes[idx]
			retrievalClient := clients[idx]

			errg.Go(func() error {
				for i := 0; i < schedulerConfig.Retries; i++ {
					retrieval, err := retrievalClient.Retrieve(errCtx, content.CID)
					issuedRetrievals.WithLabelValues(strconv.FormatBool(err == nil)).Inc()
					if err != nil {
						log.WithField("nodeID", retrievalNode.ID).WithError(err).Warnln("Failed to retrieve record")
						if err := dbc.UpdateOfflineSince(c.Context, retrievalNode); err != nil {
							log.WithField("nodeID", retrievalNode.ID).WithError(err).Warnln("Couldn't put retrieval node offline")
						}
						return nil
					}

					for _, measurement := range retrieval.Measurements {
						if _, err := dbc.InsertRetrieval(errCtx, retrievalNode.ID, retrieval.CID, measurement.Duration.Seconds(), retrieval.RoutingTableSize, measurement.Error, dbScheduler.ID); err != nil {
							return fmt.Errorf("insert retrieval: %w", err)
						}
					}
				}

				return nil
			})
		}
		if err = errg.Wait(); err != nil {
			return fmt.Errorf("waitgroup retrieve: %w", err)
		}

		provNodeIdx += 1
		provNodeIdx %= len(dbNodes)

		log.Infoln("Finished scheduling round. Starting next round soon.")
	}
}
