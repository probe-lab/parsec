package main

import (
	"context"
	"strconv"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/urfave/cli/v2"

	"github.com/probe-lab/parsec/pkg/server"
)

var serverIPNIConfig = struct {
	IndexerHost                string
	IndexerIndexTimeout        time.Duration
	IndexerAvailabilityTimeout time.Duration
	IndexerInterval            time.Duration
	PublisherHost              string
	PublisherPort              int
}{
	IndexerHost:                "",
	IndexerIndexTimeout:        1 * time.Minute,
	IndexerAvailabilityTimeout: 2 * time.Minute,
	IndexerInterval:            time.Second,
	PublisherHost:              "127.0.0.1",
	PublisherPort:              2020,
}

// ServerIPNICommand contains the crawl sub-command configuration.
var ServerIPNICommand = &cli.Command{
	Name:   "ipni",
	Action: ServerIPNIAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "indexer-host",
			EnvVars:     []string{"PARSEC_SERVER_IPNI_INDEXER_HOST"},
			DefaultText: serverIPNIConfig.IndexerHost,
			Value:       serverIPNIConfig.IndexerHost,
			Destination: &serverIPNIConfig.IndexerHost,
		},
		&cli.DurationFlag{
			Name:        "indexer-index-timeout",
			Usage:       "The maximum time to wait for the indexer to reach out to us after an announcement",
			EnvVars:     []string{"PARSEC_SERVER_IPNI_INDEXER_INDEX_TIMEOUT"},
			DefaultText: serverIPNIConfig.IndexerIndexTimeout.String(),
			Value:       serverIPNIConfig.IndexerIndexTimeout,
			Destination: &serverIPNIConfig.IndexerIndexTimeout,
		},
		&cli.DurationFlag{
			Name:        "indexer-availability-timeout",
			Usage:       "The maximum time to wait for content to become available",
			EnvVars:     []string{"PARSEC_SERVER_IPNI_INDEXER_AVAILABILITY_TIMEOUT"},
			DefaultText: serverIPNIConfig.IndexerAvailabilityTimeout.String(),
			Value:       serverIPNIConfig.IndexerAvailabilityTimeout,
			Destination: &serverIPNIConfig.IndexerAvailabilityTimeout,
		},
		&cli.DurationFlag{
			Name:        "indexer-interval",
			Usage:       "The time interval between two consecutive availability checks of the indexer",
			EnvVars:     []string{"PARSEC_SERVER_IPNI_INDEXER_INTERVAL"},
			DefaultText: serverIPNIConfig.IndexerInterval.String(),
			Value:       serverIPNIConfig.IndexerInterval,
			Destination: &serverIPNIConfig.IndexerInterval,
		},
		&cli.StringFlag{
			Name:        "publisher-host",
			Usage:       "The host address of the http server that the indexer will reach out to to fetch the advertisement",
			EnvVars:     []string{"PARSEC_SERVER_IPNI_PUBLISHER_HOST"},
			DefaultText: serverIPNIConfig.PublisherHost,
			Value:       serverIPNIConfig.PublisherHost,
			Destination: &serverIPNIConfig.PublisherHost,
		},
		&cli.IntFlag{
			Name:        "publisher-port",
			Usage:       "The port of the http server that the indexer will reach out to to fetch the advertisement",
			EnvVars:     []string{"PARSEC_SERVER_IPNI_PUBLISHER_PORT"},
			DefaultText: strconv.Itoa(serverIPNIConfig.PublisherPort),
			Value:       serverIPNIConfig.PublisherPort,
			Destination: &serverIPNIConfig.PublisherPort,
		},
	},
}

// ServerIPNIAction is the function that is called when running `nebula crawl`.
func ServerIPNIAction(c *cli.Context) error {
	return serverAction(c, func(ctx context.Context, h *server.Host, ds datastore.Batching) (server.IServer, error) {
		return server.InitIPNIServer(ctx, h, ds, ipniServerConfig())
	})
}

func ipniServerConfig() *server.IPNIServerConfig {
	return &server.IPNIServerConfig{
		IndexerHost:                serverIPNIConfig.IndexerHost,
		PublisherHost:              serverIPNIConfig.PublisherHost,
		PublisherPort:              serverIPNIConfig.PublisherPort,
		IndexerInterval:            serverIPNIConfig.IndexerInterval,
		IndexerIndexTimeout:        serverIPNIConfig.IndexerIndexTimeout,
		IndexerAvailabilityTimeout: serverIPNIConfig.IndexerAvailabilityTimeout,
	}
}
