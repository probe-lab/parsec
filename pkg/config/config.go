package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	log "github.com/sirupsen/logrus"
)

type GlobalConfig struct {
	Debug                     bool
	LogLevel                  int
	TelemetryHost             string
	TelemetryPort             int
	DryRun                    bool
	DatabaseHost              string
	DatabasePort              int
	DatabaseName              string
	DatabasePassword          string
	DatabaseUser              string
	DatabaseSSLMode           string
	ECSContainerMetadataURIV4 string
	ECSContainerMetadata      string
	ecsMetadata               *ECSMetadata
	AWSRegion                 string
}

var Global = GlobalConfig{
	TelemetryHost:    "0.0.0.0",
	TelemetryPort:    6666,
	Debug:            false,
	LogLevel:         4,
	DatabaseHost:     "localhost",
	DatabasePort:     5432,
	DatabaseName:     "parsec",
	DatabasePassword: "password",
	DatabaseUser:     "parsec",
	DatabaseSSLMode:  "disable",
}

func (g GlobalConfig) ServerProcess() (*ServerProcess, error) {
	if g.ecsMetadata != nil {
		return &ServerProcess{
			CPU:       g.ecsMetadata.Limits.CPU,
			Memory:    g.ecsMetadata.Limits.Memory,
			PrivateIP: g.ecsMetadata.GetPrivateIP(),
		}, nil
	}

	var data []byte
	if g.ECSContainerMetadataURIV4 != "" && g.ECSContainerMetadata == "" {

		resp, err := http.Get(g.ECSContainerMetadataURIV4)
		if err != nil {
			return nil, fmt.Errorf("get metadata from %s: %w", g.ECSContainerMetadataURIV4, err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("resp status code %d", resp.StatusCode)
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading ecs metadata request body: %w", err)
		}
	} else if g.ECSContainerMetadata != "" {
		data = []byte(g.ECSContainerMetadata)
	} else {

		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("get hostname: %w", err)
		}
		addrs, err := net.LookupHost(hostname)
		if err != nil {
			return nil, fmt.Errorf("lookup host %s: %w", hostname, err)
		}

		for _, addr := range addrs {
			log.Infof("Found addr: %s\n", addr)
		}

		ip := net.ParseIP(addrs[0])
		if ip == nil {
			return nil, fmt.Errorf("parse ip: %s", addrs[0])
		}

		return &ServerProcess{
			CPU:       1,
			Memory:    1,
			PrivateIP: ip,
		}, nil
	}

	log.Debugln("ECS Metadata:")
	log.Debugln(string(data))

	var md ECSMetadata
	if err := json.Unmarshal(data, &md); err != nil {
		return nil, fmt.Errorf("unmarshal ecs metadata: %w", err)
	}

	g.ecsMetadata = &md

	return &ServerProcess{
		CPU:       g.ecsMetadata.Limits.CPU,
		Memory:    g.ecsMetadata.Limits.Memory,
		PrivateIP: g.ecsMetadata.GetPrivateIP(),
	}, nil
}

type ServerConfig struct {
	ServerHost               string
	ServerPort               int
	PeerHost                 string
	PeerPort                 int
	FullRT                   bool
	DHTServer                bool
	Fleet                    string
	LevelDB                  string
	OptProv                  bool
	FirehoseStream           string
	FirehoseRegion           string
	FirehoseBatchSize        int
	FirehoseBatchTime        time.Duration
	StartupDelay             time.Duration
	IndexerHost              string
	Badbits                  string
	DeniedCIDs               string
	FirehoseConnectionEvents bool
	FirehoseRPCEvents        bool
}

var Server = ServerConfig{
	ServerHost:               "localhost",
	ServerPort:               7070,
	PeerPort:                 4001,
	Fleet:                    "",
	FullRT:                   false,
	DHTServer:                false,
	LevelDB:                  "./leveldb",
	FirehoseRegion:           "us-east-1",
	StartupDelay:             3 * time.Minute,
	IndexerHost:              "",
	Badbits:                  "badbits.deny",
	DeniedCIDs:               "cids.deny",
	FirehoseBatchTime:        30 * time.Second,
	FirehoseBatchSize:        500,
	FirehoseConnectionEvents: true,
	FirehoseRPCEvents:        true,
}

type Routing string

const (
	RoutingDHT  Routing = "DHT"
	RoutingIPNI Routing = "IPNI"
)

type SchedulerConfig struct {
	Fleets  *cli.StringSlice
	Routing string
}

var Scheduler = SchedulerConfig{
	Fleets:  cli.NewStringSlice(),
	Routing: string(RoutingDHT),
}
