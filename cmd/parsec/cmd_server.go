package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ipfs/go-datastore"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"github.com/probe-lab/parsec/pkg/db"
	"github.com/probe-lab/parsec/pkg/server"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/probe-lab/parsec/pkg/firehose"
)

var serverConfig = struct {
	ServerHost               string
	ServerPort               int
	PeerHost                 string
	PeerPort                 int
	Fleet                    string
	LevelDB                  string
	FirehoseStream           string
	FirehoseRegion           string
	FirehoseBatchSize        int
	FirehoseBatchTime        time.Duration
	StartupDelay             time.Duration
	FirehoseConnectionEvents bool
	FirehoseRPCEvents        bool
}{
	ServerHost:               "localhost",
	ServerPort:               7070,
	PeerHost:                 "127.0.0.1",
	PeerPort:                 4001,
	Fleet:                    "",
	LevelDB:                  "./leveldb",
	FirehoseRegion:           "us-east-1",
	StartupDelay:             3 * time.Minute,
	FirehoseBatchTime:        30 * time.Second,
	FirehoseBatchSize:        500,
	FirehoseConnectionEvents: true,
	FirehoseRPCEvents:        true,
}

// ServerCommand contains the crawl sub-command configuration.
var ServerCommand = &cli.Command{
	Name: "server",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "server-host",
			Usage:       "On which host address can the server be reached",
			EnvVars:     []string{"PARSEC_SERVER_SERVER_HOST"},
			DefaultText: serverConfig.ServerHost,
			Value:       serverConfig.ServerHost,
			Destination: &serverConfig.ServerHost,
		},
		&cli.IntFlag{
			Name:        "server-port",
			Usage:       "On which port can the server be reached",
			EnvVars:     []string{"PARSEC_SERVER_SERVER_PORT"},
			DefaultText: strconv.Itoa(serverConfig.ServerPort),
			Value:       serverConfig.ServerPort,
			Destination: &serverConfig.ServerPort,
		},
		&cli.StringFlag{
			Name:        "peer-host",
			Usage:       "To which network interface should the peer bind",
			EnvVars:     []string{"PARSEC_SERVER_PEER_HOST"},
			DefaultText: serverConfig.PeerHost,
			Value:       serverConfig.PeerHost,
			Destination: &serverConfig.PeerHost,
		},
		&cli.IntFlag{
			Name:        "peer-port",
			Usage:       "On which port can the peer be reached",
			EnvVars:     []string{"PARSEC_SERVER_PEER_PORT"},
			DefaultText: strconv.Itoa(serverConfig.PeerPort),
			Value:       serverConfig.PeerPort,
			Destination: &serverConfig.PeerPort,
		},
		&cli.StringFlag{
			Name:        "fleet",
			Usage:       "A fleet identifier",
			EnvVars:     []string{"PARSEC_SERVER_FLEET"},
			DefaultText: serverConfig.Fleet,
			Value:       serverConfig.Fleet,
			Destination: &serverConfig.Fleet,
		},
		&cli.StringFlag{
			Name:        "level-db",
			Usage:       "Path to the level DB datastore",
			EnvVars:     []string{"PARSEC_SERVER_LEVELDB"},
			DefaultText: serverConfig.LevelDB,
			Value:       serverConfig.LevelDB,
			Destination: &serverConfig.LevelDB,
		},
		&cli.StringFlag{
			Name:        "firehose-region",
			EnvVars:     []string{"PARSEC_SERVER_FIREHOSE_REGION"},
			DefaultText: serverConfig.FirehoseRegion,
			Value:       serverConfig.FirehoseRegion,
			Destination: &serverConfig.FirehoseRegion,
		},
		&cli.StringFlag{
			Name:        "firehose-stream",
			EnvVars:     []string{"PARSEC_SERVER_FIREHOSE_STREAM"},
			DefaultText: serverConfig.FirehoseStream,
			Value:       serverConfig.FirehoseStream,
			Destination: &serverConfig.FirehoseStream,
		},
		&cli.DurationFlag{
			Name:        "firehose-batch-time",
			EnvVars:     []string{"PARSEC_SERVER_FIREHOSE_BATCH_TIME"},
			DefaultText: serverConfig.FirehoseBatchTime.String(),
			Value:       serverConfig.FirehoseBatchTime,
			Destination: &serverConfig.FirehoseBatchTime,
		},
		&cli.IntFlag{
			Name:        "firehose-batch-size",
			EnvVars:     []string{"PARSEC_SERVER_FIREHOSE_BATCH_SIZE"},
			DefaultText: strconv.Itoa(serverConfig.FirehoseBatchSize),
			Value:       serverConfig.FirehoseBatchSize,
			Destination: &serverConfig.FirehoseBatchSize,
		},
		&cli.BoolFlag{
			Name:        "firehose-connection-events",
			EnvVars:     []string{"PARSEC_SERVER_FIREHOSE_CONNECTION_EVENTS"},
			DefaultText: strconv.FormatBool(serverConfig.FirehoseConnectionEvents),
			Value:       serverConfig.FirehoseConnectionEvents,
			Destination: &serverConfig.FirehoseConnectionEvents,
		},
		&cli.BoolFlag{
			Name:        "firehose-rpc-events",
			EnvVars:     []string{"PARSEC_SERVER_FIREHOSE_RPC_EVENTS"},
			DefaultText: strconv.FormatBool(serverConfig.FirehoseRPCEvents),
			Value:       serverConfig.FirehoseRPCEvents,
			Destination: &serverConfig.FirehoseRPCEvents,
		},
		&cli.DurationFlag{
			Name:        "startup-delay",
			EnvVars:     []string{"PARSEC_SERVER_STARTUP_DELAY"},
			DefaultText: serverConfig.StartupDelay.String(),
			Value:       serverConfig.StartupDelay,
			Destination: &serverConfig.StartupDelay,
		},
	},
	Subcommands: []*cli.Command{
		ServerDHTCommand,
		ServerIPNICommand,
	},
	Before: serverBefore,
}

func serverBefore(c *cli.Context) error {
	prometheus.MustRegister(diskUsageGauge)
	if serverConfig.FirehoseStream == "" || serverConfig.FirehoseRegion == "" {
		serverConfig.FirehoseRPCEvents = false
		serverConfig.FirehoseConnectionEvents = false
	}
	return nil
}

type serverInitFunc func(ctx context.Context, h *server.Host, ds datastore.Batching) (server.IServer, error)

func serverAction(c *cli.Context, initServerFunc serverInitFunc) error {
	log.Infoln("Starting Parsec server...")

	dbc := db.NewDummyClient()
	if !c.Bool("dry-run") {
		var err error
		if dbc, err = db.InitPGClient(c.Context, pgConfig()); err != nil {
			return fmt.Errorf("init db client: %w", err)
		}
	}

	fhClient, err := firehose.NewClient(c.Context, fhConfig())
	if err != nil {
		return fmt.Errorf("new firehose client: %w", err)
	}

	hostConfig := &server.HostConfig{
		Host:                     serverConfig.PeerHost,
		Port:                     serverConfig.PeerPort,
		FirehoseConnectionEvents: serverConfig.FirehoseConnectionEvents,
	}
	h, err := server.InitHost(c.Context, fhClient, hostConfig)
	if err != nil {
		return fmt.Errorf("new host: %w", err)
	}

	ds, err := leveldb.NewDatastore(serverConfig.LevelDB, nil)
	if err != nil {
		return fmt.Errorf("leveldb datastore: %w", err)
	}

	log.Infoln("Deleting old datastore...")
	if err := ds.Delete(c.Context, datastore.NewKey("/")); err != nil {
		log.WithError(err).Warnln("Couldn't delete old datastore")
	}

	go measureDiskUsage(c.Context, ds)

	cpu, memory, privateIP, err := processMetadata()
	if err != nil {
		return fmt.Errorf("process metadata: %w", err)
	}

	nodeModel := &db.NodeModel{
		PeerID:     h.ID(),
		AWSRegion:  rootConfig.AWSRegion,
		Fleet:      serverConfig.Fleet,
		ServerPort: serverConfig.ServerPort,
		PeerPort:   serverConfig.PeerPort,
		CPU:        cpu,
		Memory:     memory,
		PrivateIP:  privateIP,
	}

	dbNode, err := dbc.InsertNode(c.Context, nodeModel)
	if err != nil {
		return fmt.Errorf("insert node: %w", err)
	}

	serverImpl, err := initServerFunc(c.Context, h, ds)
	if err != nil {
		return fmt.Errorf("new dht server: %w", err)
	}

	serverConf := &server.Config{
		Host:         serverConfig.ServerHost,
		Port:         serverConfig.ServerPort,
		StartupDelay: serverConfig.StartupDelay,
	}
	n, err := server.NewServer(dbc, dbNode, serverImpl, serverConf)
	if err != nil {
		return fmt.Errorf("new server: %w", err)
	}

	log.Infoln("Listening and serving on", serverConf.ListenAddr())
	go func() {
		if err := n.ListenAndServe(c.Context); err != nil {
			log.WithError(err).Warnln("Stopped listen and serve")
		}
	}()

	<-c.Context.Done()

	log.Infoln("Shutting server down")
	return n.Shutdown(context.Background())
}

func fhConfig() *firehose.Config {
	return &firehose.Config{
		Fleet:     serverConfig.Fleet,
		AWSRegion: serverConfig.FirehoseRegion,
		Stream:    serverConfig.FirehoseStream,
		BatchSize: serverConfig.FirehoseBatchSize,
		BatchTime: serverConfig.FirehoseBatchTime,
	}
}
