package main

import (
	"context"
	"fmt"

	"github.com/dennis-tra/parsec/pkg/config"
	"github.com/dennis-tra/parsec/pkg/parsec"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// NodeCommand contains the crawl sub-command configuration.
var NodeCommand = &cli.Command{
	Name:   "node",
	Action: NodeAction,
	Flags:  []cli.Flag{},
}

// NodeAction is the function that is called when running `nebula crawl`.
func NodeAction(c *cli.Context) error {
	log.Infoln("Starting Parsec node...")

	// Load configuration
	conf, err := config.Init(c)
	if err != nil {
		return err
	}

	_ = conf

	n, err := parsec.NewServer("localhost", 7070)
	if err != nil {
		return fmt.Errorf("new node: %w", n)
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
