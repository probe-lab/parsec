package main

import (
	"net"
	"strconv"
	"time"

	"github.com/dennis-tra/parsec/pkg/config"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var SchedulerCommand = &cli.Command{
	Name: "scheduler",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "server-port",
			Usage:       "On which port can the server be reached",
			EnvVars:     []string{"PARSEC_SCHEDULE_SERVER_PORT"},
			DefaultText: strconv.Itoa(config.DefaultSchedulerConfig.ServerPort),
			Value:       config.DefaultSchedulerConfig.ServerPort,
		},
		&cli.BoolFlag{
			Name:        "dry-run",
			Usage:       "Whether to save data to the database or not",
			EnvVars:     []string{"PARSEC_SCHEDULE_DRY_RUN"},
			DefaultText: strconv.FormatBool(config.DefaultSchedulerConfig.DryRun),
			Value:       config.DefaultSchedulerConfig.DryRun,
		},
		&cli.StringFlag{
			Name:        "db-host",
			Usage:       "On which host address can nebula reach the database",
			EnvVars:     []string{"PARSEC_SCHEDULE_DATABASE_HOST"},
			DefaultText: config.DefaultSchedulerConfig.DatabaseHost,
			Value:       config.DefaultSchedulerConfig.DatabaseHost,
		},
		&cli.IntFlag{
			Name:        "db-port",
			Usage:       "On which port can nebula reach the database",
			EnvVars:     []string{"PARSEC_SCHEDULE_DATABASE_PORT"},
			DefaultText: strconv.Itoa(config.DefaultSchedulerConfig.DatabasePort),
			Value:       config.DefaultSchedulerConfig.DatabasePort,
		},
		&cli.StringFlag{
			Name:        "db-name",
			Usage:       "The name of the database to use",
			EnvVars:     []string{"PARSEC_SCHEDULE_DATABASE_NAME"},
			DefaultText: config.DefaultSchedulerConfig.DatabaseName,
			Value:       config.DefaultSchedulerConfig.DatabaseName,
		},
		&cli.StringFlag{
			Name:        "db-password",
			Usage:       "The password for the database to use",
			EnvVars:     []string{"PARSEC_SCHEDULE_DATABASE_PASSWORD"},
			DefaultText: config.DefaultSchedulerConfig.DatabasePassword,
			Value:       config.DefaultSchedulerConfig.DatabasePassword,
		},
		&cli.StringFlag{
			Name:        "db-user",
			Usage:       "The user with which to access the database to use",
			EnvVars:     []string{"PARSEC_SCHEDULE_DATABASE_USER"},
			DefaultText: config.DefaultSchedulerConfig.DatabaseUser,
			Value:       config.DefaultSchedulerConfig.DatabaseUser,
		},
		&cli.StringFlag{
			Name:        "db-sslmode",
			Usage:       "The sslmode to use when connecting the the database",
			EnvVars:     []string{"PARSEC_SCHEDULE_DATABASE_SSL_MODE"},
			DefaultText: config.DefaultSchedulerConfig.DatabaseSSLMode,
			Value:       config.DefaultSchedulerConfig.DatabaseSSLMode,
		},
	},
	Action: SchedulerAction,
}

func SchedulerAction(c *cli.Context) error {
	log.Infoln("Starting Parsec scheduler...")

	// conf := config.DefaultSchedulerConfig.Apply(c)

	// Acquire database handle
	//var dbc db.Client
	//var err error
	//if c.Bool("dry-run") {
	//	dbc = db.NewDummyClient()
	//} else {
	//	if dbc, err = db.InitDBClient(c.Context, conf.DatabaseHost, conf.DatabasePort, conf.DatabaseName, conf.DatabaseUser, conf.DatabasePassword, conf.DatabaseSSLMode); err != nil {
	//		return fmt.Errorf("init db client: %w", err)
	//	}
	//}

	for {
		time.Sleep(20 * time.Second)

		select {
		case <-c.Context.Done():
			return nil
		default:
		}

		ips, err := net.LookupIP("parsec.probe.lab")
		if err != nil {
			log.WithError(err).Warnln("Couldn't lookup parsec.probe.lab")
			continue
		}

		for _, ip := range ips {
			log.Infoln(ip)
		}

	}

	//dbRun, err := dbc.InsertRun(c.Context)
	//if err != nil {
	//	return fmt.Errorf("init run: %w", err)
	//}
	//
	//log.Infoln("Waiting for parsec node APIs to become available...")
	//
	//// The node index that is currently providing
	//provNodeIdx := 0
	//for {
	//	select {
	//	case <-c.Context.Done():
	//		return c.Context.Err()
	//	default:
	//	}
	//
	//	providerNode := nodes[provNodeIdx]
	//
	//	content, err := util.NewRandomContent()
	//	if err != nil {
	//		return fmt.Errorf("new random content: %w", err)
	//	}
	//
	//	provide, err := providerNode.Provide(c.Context, content)
	//	if err != nil {
	//
	//		log.WithField("nodeID", providerNode.ID).WithError(err).Warnln("Failed to provide record")
	//		return fmt.Errorf("provide content: %w", err)
	//	}
	//
	//	if _, err := dbc.InsertProvide(c.Context, providerNode.DatabaseID(), provide); err != nil {
	//		return fmt.Errorf("insert provide: %w", err)
	//	}
	//
	//	// let everyone take a breath
	//	time.Sleep(10 * time.Second)
	//
	//	// Loop through remaining nodes (len(nodes) - 1)
	//	errg, errCtx := errgroup.WithContext(c.Context)
	//	for i := 0; i < len(nodes)-1; i++ {
	//		// Start at current provNodeIdx + 1 and roll over after len(nodes) was reached
	//		retrievalNode := nodes[(provNodeIdx+1+i)%len(nodes)]
	//
	//		errg.Go(func() error {
	//			retrieval, err := retrievalNode.Retrieve(errCtx, content.CID, 1)
	//			if err != nil {
	//				return fmt.Errorf("api retrieve: %w", err)
	//			}
	//
	//			if _, err := dbc.InsertRetrieval(errCtx, retrievalNode.DatabaseID(), retrieval); err != nil {
	//				return fmt.Errorf("insert retrieve: %w", err)
	//			}
	//
	//			return nil
	//		})
	//	}
	//	if err = errg.Wait(); err != nil {
	//		return fmt.Errorf("waitgroup retrieve: %w", err)
	//	}
	//
	//	provNodeIdx += 1
	//	provNodeIdx %= len(nodes)
	//}
}
