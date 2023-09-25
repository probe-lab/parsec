package main

import (
	"context"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/dennis-tra/parsec/pkg/config"
	"github.com/dennis-tra/parsec/pkg/db"
	"github.com/dennis-tra/parsec/pkg/server"
)

// ServerCommand contains the crawl sub-command configuration.
var ServerCommand = &cli.Command{
	Name:   "server",
	Action: ServerAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "server-host",
			Usage:       "On which host address can the server be reached",
			EnvVars:     []string{"PARSEC_SERVER_SERVER_HOST"},
			DefaultText: config.Server.ServerHost,
			Value:       config.Server.ServerHost,
			Destination: &config.Server.ServerHost,
		},
		&cli.IntFlag{
			Name:        "server-port",
			Usage:       "On which port can the server be reached",
			EnvVars:     []string{"PARSEC_SERVER_SERVER_PORT"},
			DefaultText: strconv.Itoa(config.Server.ServerPort),
			Value:       config.Server.ServerPort,
			Destination: &config.Server.ServerPort,
		},
		&cli.IntFlag{
			Name:        "peer-port",
			Usage:       "On which port can the peer be reached",
			EnvVars:     []string{"PARSEC_SERVER_PEER_PORT"},
			DefaultText: strconv.Itoa(config.Server.PeerPort),
			Value:       config.Server.PeerPort,
			Destination: &config.Server.PeerPort,
		},
		&cli.BoolFlag{
			Name:        "fullrt",
			Usage:       "Whether to enable the full routing table setting on the DHT",
			EnvVars:     []string{"PARSEC_SERVER_FULLRT"},
			DefaultText: strconv.FormatBool(config.Server.FullRT),
			Value:       config.Server.FullRT,
			Destination: &config.Server.FullRT,
		},
		&cli.BoolFlag{
			Name:        "dht-server",
			Usage:       "Whether to enable DHT server mode",
			EnvVars:     []string{"PARSEC_SERVER_DHT_SERVER"},
			DefaultText: strconv.FormatBool(config.Server.DHTServer),
			Value:       config.Server.DHTServer,
			Destination: &config.Server.DHTServer,
		},
		&cli.BoolFlag{
			Name:        "optprov",
			Usage:       "Whether to enable optimistic provide",
			EnvVars:     []string{"PARSEC_SERVER_OPTPROV"},
			DefaultText: strconv.FormatBool(config.Server.OptProv),
			Value:       config.Server.OptProv,
			Destination: &config.Server.OptProv,
		},
		&cli.StringFlag{
			Name:        "fleet",
			Usage:       "A fleet identifier",
			EnvVars:     []string{"PARSEC_SERVER_FLEET"},
			DefaultText: config.Server.Fleet,
			Value:       config.Server.Fleet,
			Destination: &config.Server.Fleet,
		},
		&cli.StringFlag{
			Name:        "level-db",
			Usage:       "Path to the level DB datastore",
			EnvVars:     []string{"PARSEC_SERVER_LEVELDB"},
			DefaultText: config.Server.LevelDB,
			Value:       config.Server.LevelDB,
			Destination: &config.Server.LevelDB,
		},
		&cli.StringFlag{
			Name:        "firehose-region",
			EnvVars:     []string{"PARSEC_SERVER_FIREHOSE_REGION"},
			DefaultText: config.Server.FirehoseRegion,
			Value:       config.Server.FirehoseRegion,
			Destination: &config.Server.FirehoseRegion,
		},
		&cli.StringFlag{
			Name:        "firehose-stream",
			EnvVars:     []string{"PARSEC_SERVER_FIREHOSE_STREAM"},
			DefaultText: config.Server.FirehoseStream,
			Value:       config.Server.FirehoseStream,
			Destination: &config.Server.FirehoseStream,
		},
		&cli.BoolFlag{
			Name:        "firehose-rpcs",
			EnvVars:     []string{"PARSEC_SERVER_FIREHOSE_RPCS"},
			DefaultText: strconv.FormatBool(config.Server.FirehoseRPCs),
			Value:       config.Server.FirehoseRPCs,
			Destination: &config.Server.FirehoseRPCs,
		},
		&cli.BoolFlag{
			Name:        "firehose-conns",
			EnvVars:     []string{"PARSEC_SERVER_FIREHOSE_CONNS"},
			DefaultText: strconv.FormatBool(config.Server.FirehoseConns),
			Value:       config.Server.FirehoseConns,
			Destination: &config.Server.FirehoseConns,
		},
		&cli.DurationFlag{
			Name:        "startup-delay",
			EnvVars:     []string{"PARSEC_SERVER_STARTUP_DELAY"},
			DefaultText: config.Server.StartupDelay.String(),
			Value:       config.Server.StartupDelay,
			Destination: &config.Server.StartupDelay,
		},
		&cli.StringFlag{
			Name:        "indexer-host",
			EnvVars:     []string{"PARSEC_SERVER_INDEXER_HOST"},
			DefaultText: config.Server.IndexerHost,
			Value:       config.Server.IndexerHost,
			Destination: &config.Server.IndexerHost,
		},
	},
}

// ServerAction is the function that is called when running `nebula crawl`.
func ServerAction(c *cli.Context) error {
	log.Infoln("Starting Parsec server...")

	dbc := db.NewDummyClient()
	if !c.Bool("dry-run") {
		var err error
		if dbc, err = db.InitDBClient(c.Context, config.Global); err != nil {
			return fmt.Errorf("init db client: %w", err)
		}
	}

	n, err := server.NewServer(c.Context, dbc, config.Server)
	if err != nil {
		return fmt.Errorf("new server: %w", err)
	}

	log.Infoln("Listening and serving on", n.ListenAddr())
	go func() {
		if err := n.ListenAndServe(c.Context); err != nil {
			log.WithError(err).Warnln("Stopped listen and serve")
		}
	}()

	<-c.Context.Done()

	log.Infoln("Shutting server down")
	return n.Shutdown(context.Background())
}
