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
	"github.com/dennis-tra/parsec/pkg/db"
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
		&cli.BoolFlag{
			Name:        "dry-run",
			Usage:       "Whether to save data to the database or not",
			EnvVars:     []string{"PARSEC_SCHEDULE_DRY_RUN"},
			DefaultText: strconv.FormatBool(config.DefaultScheduleConfig.DryRun),
			Value:       config.DefaultScheduleConfig.DryRun,
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

	defer Cleanup(nodes)

	conf := config.DefaultScheduleConfig.Apply(c)

	// Acquire database handle
	var dbc db.Client
	var err error
	if c.Bool("dry-run") {
		dbc = db.NewDummyClient()
	} else {
		if dbc, err = db.InitDBClient(c.Context, conf.DatabaseHost, conf.DatabasePort, conf.DatabaseName, conf.DatabaseUser, conf.DatabasePassword, conf.DatabaseSSLMode); err != nil {
			return fmt.Errorf("init db client: %w", err)
		}
	}
	dbRun, err := dbc.InsertRun(c.Context)
	if err != nil {
		return fmt.Errorf("init run: %w", err)
	}

	log.Infoln("Waiting for parsec node APIs to become available...")

	errCtx, cancel := context.WithTimeout(c.Context, 5*time.Minute)
	errg, errCtx := errgroup.WithContext(errCtx)
	for _, node := range nodes {
		node := node
		errg.Go(func() error {
			info, err := node.WaitForAPI(errCtx)
			if err != nil {
				return err
			}

			dbNode, err := dbc.InsertNode(errCtx, dbRun.ID, info.PeerID, node.Cluster().Region, node.Cluster().InstanceType, info.BuildInfo)
			if err != nil {
				return fmt.Errorf("insert db node: %w", err)
			}

			node.Assign(dbNode)

			return nil
		})
	}
	if err := errg.Wait(); err != nil {
		cancel()
		return fmt.Errorf("waiting for parsec node APIs: %w", err)
	}
	cancel()

	// The node index that is currently providing
	provNodeIdx := 0
	for {
		select {
		case <-c.Context.Done():
			return c.Context.Err()
		default:
		}

		providerNode := nodes[provNodeIdx]

		content, err := util.NewRandomContent()
		if err != nil {
			return fmt.Errorf("new random content: %w", err)
		}

		provide, err := providerNode.Provide(c.Context, content)
		if err != nil {

			log.WithField("nodeID", providerNode.ID).WithError(err).Warnln("Failed to provide record")
			return fmt.Errorf("provide content: %w", err)
		}

		if _, err := dbc.InsertProvide(c.Context, providerNode.DatabaseID(), provide); err != nil {
			return fmt.Errorf("insert provide: %w", err)
		}

		// let everyone take a breath
		time.Sleep(10 * time.Second)

		// Loop through remaining nodes (len(nodes) - 1)
		errg, errCtx := errgroup.WithContext(c.Context)
		for i := 0; i < len(nodes)-1; i++ {
			// Start at current provNodeIdx + 1 and roll over after len(nodes) was reached
			retrievalNode := nodes[(provNodeIdx+1+i)%len(nodes)]

			errg.Go(func() error {
				retrieval, err := retrievalNode.Retrieve(errCtx, content.CID, 1)
				if err != nil {
					return fmt.Errorf("api retrieve: %w", err)
				}

				if _, err := dbc.InsertRetrieval(errCtx, retrievalNode.DatabaseID(), retrieval); err != nil {
					return fmt.Errorf("insert retrieve: %w", err)
				}

				return nil
			})
		}
		if err = errg.Wait(); err != nil {
			return fmt.Errorf("waitgroup retrieve: %w", err)
		}

		provNodeIdx += 1
		provNodeIdx %= len(nodes)
	}
}

func Cleanup(nodes []*parsec.Node) {
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		node := node
		go func() {
			defer wg.Done()
			if err := node.Cluster().Cleanup(); err != nil {
				log.WithField("nodeID", node.ID()).Warnln("Error cleaning up cluster")
			}
		}()
	}
	wg.Wait()
}
