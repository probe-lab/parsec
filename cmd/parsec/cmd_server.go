package main

import (
	"context"
	"fmt"

	"github.com/dennis-tra/parsec/pkg/config"
	"github.com/dennis-tra/parsec/pkg/parsec"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// ServerCommand contains the crawl sub-command configuration.
var ServerCommand = &cli.Command{
	Name:   "server",
	Action: ServerAction,
	Flags:  []cli.Flag{},
}

// ServerAction is the function that is called when running `nebula crawl`.
func ServerAction(c *cli.Context) error {
	log.SetFormatter(&log.JSONFormatter{})

	log.Infoln("Starting Parsec server...")

	// Load configuration
	conf, err := config.Init(c)
	if err != nil {
		return err
	}

	_ = conf

	n, err := parsec.NewServer("localhost", 7070)
	if err != nil {
		return fmt.Errorf("new node: %w", err)
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
