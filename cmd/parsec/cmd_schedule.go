package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/dennis-tra/parsec/pkg/models"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"golang.org/x/sync/errgroup"

	"github.com/dennis-tra/parsec/pkg/config"
	"github.com/dennis-tra/parsec/pkg/db"
	"github.com/dennis-tra/parsec/pkg/parsec"
	"github.com/dennis-tra/parsec/pkg/util"
	"github.com/libp2p/go-libp2p/core/peer"
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

	conf := config.DefaultScheduleConfig.Apply(c)

	// Acquire database handle
	var (
		dbc *db.DBClient
		err error
	)
	if !c.Bool("dry-run") {
		if dbc, err = db.InitDBClient(c.Context, conf.DatabaseHost, conf.DatabasePort, conf.DatabaseName, conf.DatabaseUser, conf.DatabasePassword, conf.DatabaseSSLMode); err != nil {
			return fmt.Errorf("init db client: %w", err)
		}
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("read build info: %w", err)
	}

	var dbRun *models.Run
	if dbc != nil {
		dbRun, err = dbc.InitRun(c.Context, bi)
		if err != nil {
			return fmt.Errorf("init run: %w", err)
		}
	}

	log.Infoln("Waiting for parsec node APIs to become available...")

	dbNodesLk := sync.RWMutex{}
	dbNodes := make([]*models.Node, len(nodes))

	errCtx, cancel := context.WithTimeout(c.Context, 20*time.Minute)
	errg, errCtx := errgroup.WithContext(errCtx)
	for i, n := range nodes {
		n2 := n
		i2 := i
		errg.Go(func() error {
			info, err := n2.WaitForAPI(errCtx)
			if err != nil {
				return err
			}

			if dbc == nil {
				return nil
			}

			dbNode, err := dbc.InsertNode(errCtx, dbRun.ID, info.PeerID, n.Cluster().Region, n.Cluster().InstanceType, info.BuildInfo)
			if err != nil {
				return fmt.Errorf("insert db node: %w", err)
			}

			dbNodesLk.Lock()
			dbNodes[i2] = dbNode
			dbNodesLk.Unlock()

			return nil
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
			log.WithField("nodeID", dbNodes[provNode].ID).WithError(err).Warnln("Failed to provide record")

			if dbc == nil {
				continue
			}

			m := models.Measurement{
				NodeID: dbNodes[provNode].ID,
				Metric: string(MetricProvideProvideError),
				Error:  null.StringFrom(err.Error()),
			}

			if err = m.Insert(c.Context, dbc.Handle(), boil.Infer()); err != nil {
				log.WithField("nodeID", nodes[provNode].ID()).WithError(err).Warnln("Failed to save measurement entry")
			}

			continue
		}

		if dbc != nil {
			if err = saveProvide(c.Context, dbc.Handle(), dbNodes[provNode].ID, provide); err != nil {
				log.WithField("nodeID", nodes[provNode].ID()).WithError(err).Errorln("Failed to save measurement entry")
				return err
			}
		}

		// Loop through remaining nodes (len(nodes) - 1)
		var wg sync.WaitGroup
		for i := 0; i < len(nodes)-1; i++ {
			wg.Add(1)

			// Start at current provNode + 1 and roll over after len(nodes) was reached
			retrNode := (provNode + i + 1) % len(nodes)

			go func() {
				defer wg.Done()
				node := nodes[retrNode]
				dbNode := dbNodes[retrNode]

				retrieval, err := node.Retrieve(c.Context, content.CID, 1)
				if err != nil {
					log.WithField("nodeID", node.ID()).WithError(err).Warnln("Failed to retrieve record")

					if dbc == nil {
						return
					}

					m := models.Measurement{
						NodeID: dbNode.ID,
						Metric: string(MetricRetrievalError),
						Error:  null.StringFrom(err.Error()),
					}

					if err = m.Insert(c.Context, dbc.Handle(), boil.Infer()); err != nil {
						log.WithField("nodeID", node.ID()).WithError(err).Warnln("Failed to save measurement entry")
					}

					return
				}

				log.WithField("nodeID", node.ID()).WithField("dur", retrieval.TimeToFirstProviderRecord()).Infoln("Time to first provider record")

				if dbc != nil {
					if err = saveRetrieval(c.Context, dbc.Handle(), dbNode.ID, retrieval); err != nil {
						log.WithField("nodeID", node.ID()).WithError(err).Errorln("Failed to save measurement entry")
						return
					}
				}
			}()
		}
		wg.Wait()

		provNode += 1
		provNode %= len(nodes)
	}
}

type Metric string

const (
	MetricProvideProvideError             Metric = "provide_error"
	MetricProvideTotalTime                Metric = "provide_total_time"
	MetricProvideFindNodesCount           Metric = "provide_find_nodes_count"
	MetricProvideAddProvidersCount        Metric = "provide_add_providers_count"
	MetricProvideAddProvidersSuccessCount Metric = "provide_add_providers_success_count"

	MetricRetrievalError   Metric = "retrieval_error"
	MetricRetrievalTTFPR   Metric = "retrieval_ttfpr"
	MetricRetrievaHopCount Metric = "retrieval_hop_count"
)

func saveProvide(ctx context.Context, handle *sql.DB, dbNodeID int, provide *parsec.ProvideResponse) error {
	if err := saveMeasurement(ctx, handle, dbNodeID, MetricProvideTotalTime, provide.End.Sub(provide.Start).Seconds()); err != nil {
		return err
	}

	if err := saveMeasurement(ctx, handle, dbNodeID, MetricProvideFindNodesCount, float64(len(provide.FindNodes))); err != nil {
		return err
	}

	if err := saveMeasurement(ctx, handle, dbNodeID, MetricProvideAddProvidersCount, float64(len(provide.AddProviders))); err != nil {
		return err
	}

	addProvidersSuccessesCount := 0
	for _, addProvider := range provide.AddProviders {
		if addProvider.Error == "" {
			addProvidersSuccessesCount += 1
		}
	}

	if err := saveMeasurement(ctx, handle, dbNodeID, MetricProvideAddProvidersSuccessCount, float64(addProvidersSuccessesCount)); err != nil {
		return err
	}

	return nil
}

func saveRetrieval(ctx context.Context, handle *sql.DB, dbNodeID int, retrieval *parsec.RetrievalResponse) error {
	if err := saveMeasurement(ctx, handle, dbNodeID, MetricRetrievalTTFPR, retrieval.TimeToFirstProviderRecord().Seconds()); err != nil {
		return err
	}

	for _, getProvider := range retrieval.GetProviders {
		if len(getProvider.Providers) == 0 {
			continue
		}
		provider := getProvider.Providers[0]
		if err := saveMeasurement(ctx, handle, dbNodeID, MetricRetrievaHopCount, float64(hopCount(retrieval.PeerSet, provider))); err != nil {
			return err
		}
	}

	return nil
}

func hopCount(peerSet []parsec.PeerSet, peerID peer.ID) int {
	for _, ps := range peerSet {
		if ps.RemotePeerID != peerID {
			continue
		}

		if errors.Is(ps.ReferredBy.Validate(), peer.ErrEmptyPeerID) {
			continue
		}
		return hopCount(peerSet, ps.ReferredBy) + 1
	}
	return 0
}

func saveMeasurement(ctx context.Context, handle *sql.DB, dbNodeID int, metric Metric, value float64) error {
	m := models.Measurement{
		NodeID: dbNodeID,
		Metric: string(metric),
		Value:  null.Float64From(value),
	}

	if err := m.Insert(ctx, handle, boil.Infer()); err != nil {
		return fmt.Errorf("saving measurement %s: %w", MetricRetrievaHopCount, err)
	}

	return nil
}
