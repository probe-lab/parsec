package main

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"

	"github.com/dennis-tra/parsec/pkg/config"
	"github.com/dennis-tra/parsec/pkg/parsec"
	"github.com/dennis-tra/parsec/pkg/util"
)

var ScheduleCommand = &cli.Command{
	Name: "schedule",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "server-host",
			Usage:       "On which host address can the server be reached",
			EnvVars:     []string{"PARSEC_SCHEDULE_SERVER_HOST"},
			DefaultText: config.DefaultScheduleConfig.ServerHost,
			Value:       config.DefaultScheduleConfig.ServerHost,
		},
		&cli.IntFlag{
			Name:        "server-port",
			Usage:       "On which port can the server be reached",
			EnvVars:     []string{"PARSEC_SCHEDULE_SERVER_PORT"},
			DefaultText: strconv.Itoa(config.DefaultScheduleConfig.ServerPort),
			Value:       config.DefaultScheduleConfig.ServerPort,
		},
		&cli.StringFlag{
			Name:        "db-host",
			Usage:       "On which host address can nebula reach the database",
			EnvVars:     []string{"PARSEC_SCHEDULE_DATABASE_HOST"},
			DefaultText: config.DefaultScheduleConfig.DatabaseHost,
			Value:       config.DefaultScheduleConfig.DatabaseHost,
		},
		&cli.IntFlag{
			Name:        "db-port",
			Usage:       "On which port can nebula reach the database",
			EnvVars:     []string{"PARSEC_SCHEDULE_DATABASE_PORT"},
			DefaultText: strconv.Itoa(config.DefaultScheduleConfig.DatabasePort),
			Value:       config.DefaultScheduleConfig.DatabasePort,
		},
		&cli.StringFlag{
			Name:        "db-name",
			Usage:       "The name of the database to use",
			EnvVars:     []string{"PARSEC_SCHEDULE_DATABASE_NAME"},
			DefaultText: config.DefaultScheduleConfig.DatabaseName,
			Value:       config.DefaultScheduleConfig.DatabaseName,
		},
		&cli.StringFlag{
			Name:        "db-password",
			Usage:       "The password for the database to use",
			EnvVars:     []string{"PARSEC_SCHEDULE_DATABASE_PASSWORD"},
			DefaultText: config.DefaultScheduleConfig.DatabasePassword,
			Value:       config.DefaultScheduleConfig.DatabasePassword,
		},
		&cli.StringFlag{
			Name:        "db-user",
			Usage:       "The user with which to access the database to use",
			EnvVars:     []string{"PARSEC_SCHEDULE_DATABASE_USER"},
			DefaultText: config.DefaultScheduleConfig.DatabaseUser,
			Value:       config.DefaultScheduleConfig.DatabaseUser,
		},
		&cli.StringFlag{
			Name:        "db-sslmode",
			Usage:       "The sslmode to use when connecting the the database",
			EnvVars:     []string{"PARSEC_SCHEDULE_DATABASE_SSL_MODE"},
			DefaultText: config.DefaultScheduleConfig.DatabaseSSLMode,
			Value:       config.DefaultScheduleConfig.DatabaseSSLMode,
		},
	},
	Subcommands: []*cli.Command{
		ScheduleDockerCommand,
		ScheduleAWSCommand,
	},
}

func ScheduleAction(c *cli.Context, nodes []*parsec.Node) error {
	log.Infoln("Starting Parsec scheduler...")

	// Acquire database handle
	//var dbc *db.Client
	//if !c.Bool("dry-run") {
	//	if dbc, err = db.InitClient(c.Context, conf); err != nil {
	//		return err
	//	}
	//}
	//_ = dbc

	log.Infoln("Waiting for parsec node APIs to become available...")
	ctx, cancel := context.WithTimeout(c.Context, time.Minute)
	errg, ctx := errgroup.WithContext(ctx)
	for _, n := range nodes {
		n2 := n
		errg.Go(func() error {
			return n2.WaitForAPI(ctx)
		})
	}
	if err := errg.Wait(); err != nil {
		cancel()
		return fmt.Errorf("waiting for parsec node APIs: %w", err)
	}
	cancel()

	// The node index that is currently providing
	provNode := 0
	for {
		select {
		case <-c.Context.Done():
			return c.Context.Err()
		default:
		}

		content, err := util.NewRandomContent()
		if err != nil {
			return fmt.Errorf("new random content: %w", err)
		}

		provide, err := nodes[provNode].Provide(c.Context, content)
		if err != nil {
			return err
		}
		_ = provide

		// Loop through remaining nodes (len(nodes) - 1)
		var wg sync.WaitGroup
		for i := 0; i < len(nodes)-1; i++ {
			wg.Add(1)

			// Start at current provNode + 1 and roll over after len(nodes) was reached
			retrNode := (provNode + i + 1) % len(nodes)

			go func() {
				defer wg.Done()
				retrieval, err := nodes[retrNode].Retrieve(c.Context, content.CID, 1)
				if err != nil {
					log.WithError(err).Infoln("asdfsdf")
				} else {
					log.WithField("nodeID", nodes[retrNode].ID()).WithField("dur", retrieval.TimeToFirstProviderRecord()).Infoln("Time to first provider record")
				}
			}()
		}
		wg.Wait()

		provNode += 1
		provNode %= len(nodes)
	}
}
