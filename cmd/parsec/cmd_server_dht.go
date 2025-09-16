package main

import (
	"context"
	"strconv"

	"github.com/ipfs/go-datastore"
	"github.com/urfave/cli/v2"

	"github.com/probe-lab/parsec/pkg/server"
)

var serverDHTConfig = struct {
	FullRT     bool
	OptProv    bool
	ServerMode bool
	Badbits    string
	DeniedCIDs string
}{
	FullRT:     false,
	OptProv:    false,
	ServerMode: false,
	Badbits:    "",
	DeniedCIDs: "",
}

// ServerDHTCommand contains the crawl sub-command configuration.
var ServerDHTCommand = &cli.Command{
	Name:   "dht",
	Action: ServerDHTAction,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:        "fullrt",
			Usage:       "Whether to enable the full routing table setting on the DHT",
			EnvVars:     []string{"PARSEC_SERVER_DHT_FULLRT"},
			DefaultText: strconv.FormatBool(serverDHTConfig.FullRT),
			Value:       serverDHTConfig.FullRT,
			Destination: &serverDHTConfig.FullRT,
		},
		&cli.BoolFlag{
			Name:        "optprov",
			Usage:       "Whether to enable optimistic provide",
			EnvVars:     []string{"PARSEC_SERVER_DHT_OPTPROV"},
			DefaultText: strconv.FormatBool(serverDHTConfig.OptProv),
			Value:       serverDHTConfig.OptProv,
			Destination: &serverDHTConfig.OptProv,
		},
		&cli.BoolFlag{
			Name:        "server-mode",
			Usage:       "Whether to enable DHT server mode",
			EnvVars:     []string{"PARSEC_SERVER_DHT_SERVER_MODE"},
			DefaultText: strconv.FormatBool(serverDHTConfig.ServerMode),
			Value:       serverDHTConfig.ServerMode,
			Destination: &serverDHTConfig.ServerMode,
		},
		&cli.StringFlag{
			Name:        "badbits",
			EnvVars:     []string{"PARSEC_SERVER_DHT_BADBITS"},
			DefaultText: serverDHTConfig.Badbits,
			Value:       serverDHTConfig.Badbits,
			Destination: &serverDHTConfig.Badbits,
		},
		&cli.StringFlag{
			Name:        "denied-cids",
			EnvVars:     []string{"PARSEC_SERVER_DHT_DENIED_CIDS"},
			DefaultText: serverDHTConfig.DeniedCIDs,
			Value:       serverDHTConfig.DeniedCIDs,
			Destination: &serverDHTConfig.DeniedCIDs,
		},
	},
}

// ServerDHTAction is the function that is called when running `nebula crawl`.
func ServerDHTAction(c *cli.Context) error {
	return serverAction(c, func(ctx context.Context, h *server.Host, ds datastore.Batching) (server.IServer, error) {
		return server.InitDHTServer(ctx, h, ds, dhtServerConfig())
	})
}

func dhtServerConfig() *server.DHTServerConfig {
	return &server.DHTServerConfig{
		Badbits:           serverDHTConfig.Badbits,
		DeniedCIDs:        serverDHTConfig.DeniedCIDs,
		ServerMode:        serverDHTConfig.ServerMode,
		FullRT:            serverDHTConfig.FullRT,
		OptProv:           serverDHTConfig.OptProv,
		FirehoseRPCEvents: serverConfig.FirehoseRPCEvents,
	}
}
