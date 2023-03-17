package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/volatiletech/sqlboiler/v4/boil"

	"github.com/dennis-tra/parsec/pkg/config"
)

var (
	// RawVersion and build tag of the
	// PCP command line tool. This is
	// replaced on build via e.g.:
	// -ldflags "-X main.RawVersion=${VERSION}"
	RawVersion  = "dev"
	ShortCommit = "5f3759df" // quake
)

func main() {
	// ShortCommit version tag
	verTag := fmt.Sprintf("v%s+%s", RawVersion, ShortCommit)

	app := &cli.App{
		Name: "parsec",
		Authors: []*cli.Author{
			{
				Name:  "Dennis Trautwein",
				Email: "dennis@protocol.ai",
			},
		},
		Version: verTag,
		Before:  Before,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "Set this flag to enable debug logging",
				EnvVars:     []string{"PARSEC_DEBUG"},
				Value:       config.DefaultGlobalConfig.Debug,
				DefaultText: strconv.FormatBool(config.DefaultGlobalConfig.Debug),
			},
			&cli.IntFlag{
				Name:        "log-level",
				Usage:       "Set this flag to a value from 0 (least verbose) to 6 (most verbose). Overrides the --debug flag",
				EnvVars:     []string{"PARSEC_LOG_LEVEL"},
				Value:       config.DefaultGlobalConfig.LogLevel,
				DefaultText: strconv.Itoa(config.DefaultGlobalConfig.LogLevel),
			},
			&cli.StringFlag{
				Name:        "telemetry-host",
				Usage:       "To which network address should the telemetry (prometheus, pprof) server bind",
				EnvVars:     []string{"PARSEC_TELEMETRY_HOST"},
				Value:       config.DefaultGlobalConfig.TelemetryHost,
				DefaultText: config.DefaultGlobalConfig.TelemetryHost,
			},
			&cli.IntFlag{
				Name:        "telemetry-port",
				Usage:       "On which port should the telemetry (prometheus, pprof) server listen",
				EnvVars:     []string{"PARSEC_TELEMETRY_PORT"},
				Value:       config.DefaultGlobalConfig.TelemetryPort,
				DefaultText: strconv.Itoa(config.DefaultGlobalConfig.TelemetryPort),
			},
		},
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			SchedulerCommand,
			ServerCommand,
		},
	}

	sigs := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	go func() {
		sig := <-sigs
		log.Printf("Received %s signal - Stopping...\n", sig.String())
		signal.Stop(sigs)
		cancel()
	}()

	if err := app.RunContext(ctx, os.Args); err != nil && !errors.Is(err, context.Canceled) {
		log.Errorf("error: %v\n", err)
		os.Exit(1)
	}
	log.Infoln("Parsec stopped.")
}

// Before is executed before any subcommands are run, but after the context is ready
// If a non-nil error is returned, no subcommands are run.
func Before(c *cli.Context) error {
	if c.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}

	if c.IsSet("log-level") {
		ll := c.Int("log-level")
		log.SetLevel(log.Level(ll))
		if ll == int(log.TraceLevel) {
			boil.DebugMode = true
		}
	}

	// Start prometheus metrics endpoint
	go metricsListenAndServe(c.String("telemetry-host"), c.Int("telemetry-port"))

	return nil
}

func metricsListenAndServe(host string, port int) {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.WithField("addr", addr).Debugln("Starting telemetry endpoint")
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.WithError(err).Warnln("Error serving prometheus")
	}
}
