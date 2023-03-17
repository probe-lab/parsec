package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dennis-tra/parsec/pkg/server"

	"github.com/dennis-tra/parsec/pkg/config"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
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
			DefaultText: config.DefaultServerConfig.ServerHost,
			Value:       config.DefaultServerConfig.ServerHost,
		},
		&cli.IntFlag{
			Name:        "server-port",
			Usage:       "On which port can the server be reached",
			EnvVars:     []string{"PARSEC_SERVER_SERVER_PORT"},
			DefaultText: strconv.Itoa(config.DefaultServerConfig.ServerPort),
			Value:       config.DefaultServerConfig.ServerPort,
		},
		&cli.IntFlag{
			Name:        "peer-port",
			Usage:       "On which port can the peer be reached",
			EnvVars:     []string{"PARSEC_SERVER_PEER_PORT"},
			DefaultText: strconv.Itoa(config.DefaultServerConfig.PeerPort),
			Value:       config.DefaultServerConfig.PeerPort,
		},
	},
}

// ServerAction is the function that is called when running `nebula crawl`.
func ServerAction(c *cli.Context) error {
	log.SetFormatter(&log.JSONFormatter{})

	log.Infoln("Starting Parsec server...")

	conf := config.DefaultServerConfig.Apply(c)

	n, err := server.NewServer(conf.ServerHost, conf.ServerPort, conf.PeerPort)
	if err != nil {
		return fmt.Errorf("new server: %w", err)
	}

	log.Infoln("Listening and serving on", n.ListenAddr())
	go func() {
		if err := n.ListenAndServe(); err != nil {
			log.WithError(err).Warnln("Stopped listen and serve")
		}
	}()

	<-c.Context.Done()

	return n.Shutdown(context.Background())
}
