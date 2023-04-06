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

	ocprom "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/prometheus/client_golang/prometheus"
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
	app := &cli.App{
		Name: "parsec",
		Authors: []*cli.Author{
			{
				Name:  "Dennis Trautwein",
				Email: "dennis@protocol.ai",
			},
		},
		Version: fmt.Sprintf("v%s+%s", RawVersion, ShortCommit),
		Before:  Before,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "Set this flag to enable debug logging",
				EnvVars:     []string{"PARSEC_DEBUG"},
				DefaultText: strconv.FormatBool(config.Global.Debug),
				Destination: &config.Global.Debug,
				Value:       config.Global.Debug,
			},
			&cli.IntFlag{
				Name:        "log-level",
				Usage:       "Set this flag to a value from 0 (least verbose) to 6 (most verbose). Overrides the --debug flag",
				EnvVars:     []string{"PARSEC_LOG_LEVEL"},
				DefaultText: strconv.Itoa(config.Global.LogLevel),
				Destination: &config.Global.LogLevel,
				Value:       config.Global.LogLevel,
			},
			&cli.StringFlag{
				Name:        "telemetry-host",
				Usage:       "To which network address should the telemetry (prometheus, pprof) server bind",
				EnvVars:     []string{"PARSEC_TELEMETRY_HOST"},
				DefaultText: config.Global.TelemetryHost,
				Destination: &config.Global.TelemetryHost,
				Value:       config.Global.TelemetryHost,
			},
			&cli.IntFlag{
				Name:        "telemetry-port",
				Usage:       "On which port should the telemetry (prometheus, pprof) server listen",
				EnvVars:     []string{"PARSEC_TELEMETRY_PORT"},
				DefaultText: strconv.Itoa(config.Global.TelemetryPort),
				Destination: &config.Global.TelemetryPort,
				Value:       config.Global.TelemetryPort,
			},
			&cli.BoolFlag{
				Name:        "dry-run",
				Usage:       "Whether to save data to the database or not",
				EnvVars:     []string{"PARSEC_SCHEDULE_DRY_RUN"},
				DefaultText: strconv.FormatBool(config.Global.DryRun),
				Value:       config.Global.DryRun,
				Destination: &config.Global.DryRun,
			},
			&cli.StringFlag{
				Name:        "db-host",
				Usage:       "On which host address can nebula reach the database",
				EnvVars:     []string{"PARSEC_DATABASE_HOST"},
				DefaultText: config.Global.DatabaseHost,
				Value:       config.Global.DatabaseHost,
				Destination: &config.Global.DatabaseHost,
			},
			&cli.IntFlag{
				Name:        "db-port",
				Usage:       "On which port can nebula reach the database",
				EnvVars:     []string{"PARSEC_DATABASE_PORT"},
				DefaultText: strconv.Itoa(config.Global.DatabasePort),
				Value:       config.Global.DatabasePort,
				Destination: &config.Global.DatabasePort,
			},
			&cli.StringFlag{
				Name:        "db-name",
				Usage:       "The name of the database to use",
				EnvVars:     []string{"PARSEC_DATABASE_NAME"},
				DefaultText: config.Global.DatabaseName,
				Value:       config.Global.DatabaseName,
				Destination: &config.Global.DatabaseName,
			},
			&cli.StringFlag{
				Name:        "db-password",
				Usage:       "The password for the database to use",
				EnvVars:     []string{"PARSEC_DATABASE_PASSWORD"},
				DefaultText: config.Global.DatabasePassword,
				Value:       config.Global.DatabasePassword,
				Destination: &config.Global.DatabasePassword,
			},
			&cli.StringFlag{
				Name:        "db-user",
				Usage:       "The user with which to access the database to use",
				EnvVars:     []string{"PARSEC_DATABASE_USER"},
				DefaultText: config.Global.DatabaseUser,
				Value:       config.Global.DatabaseUser,
				Destination: &config.Global.DatabaseUser,
			},
			&cli.StringFlag{
				Name:        "db-sslmode",
				Usage:       "The sslmode to use when connecting the the database",
				EnvVars:     []string{"PARSEC_DATABASE_SSL_MODE"},
				DefaultText: config.Global.DatabaseSSLMode,
				Value:       config.Global.DatabaseSSLMode,
				Destination: &config.Global.DatabaseSSLMode,
			},
			&cli.StringFlag{
				// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html
				// https://stackoverflow.com/questions/55718332/how-do-i-get-my-ip-address-from-inside-an-ecs-container-running-with-the-awsvpc
				Name:        "ecs-metadata-uri-v4",
				Usage:       "The URL to the metadata endpoint",
				EnvVars:     []string{"ECS_CONTAINER_METADATA_URI_V4"},
				DefaultText: config.Global.ECSContainerMetadataURIV4,
				Value:       config.Global.ECSContainerMetadataURIV4,
				Destination: &config.Global.ECSContainerMetadataURIV4,
			},
			&cli.StringFlag{
				Name:        "ecs-metadata",
				Usage:       "The ECS container metadata JSON (skips querying the URIv4 endpoint)",
				EnvVars:     []string{"ECS_CONTAINER_METADATA"},
				DefaultText: config.Global.ECSContainerMetadata,
				Value:       config.Global.ECSContainerMetadata,
				Destination: &config.Global.ECSContainerMetadata,
			},
			&cli.StringFlag{
				// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html
				// https://stackoverflow.com/questions/55718332/how-do-i-get-my-ip-address-from-inside-an-ecs-container-running-with-the-awsvpc
				Name:        "aws-region",
				Usage:       "The AWS region that this server is running in",
				EnvVars:     []string{"AWS_REGION"},
				DefaultText: config.Global.AWSRegion,
				Value:       config.Global.AWSRegion,
				Destination: &config.Global.AWSRegion,
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

	pe, err := ocprom.NewExporter(ocprom.Options{
		Namespace:  "parsec",
		Registerer: prometheus.DefaultRegisterer,
		Gatherer:   prometheus.DefaultGatherer,
	})
	if err != nil {
		log.Fatalf("Failed to create the Prometheus stats exporter: %v", err)
	}

	mux := http.DefaultServeMux
	mux.Handle("/metrics", pe)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.WithError(err).Warnln("Error serving prometheus")
	}
}
