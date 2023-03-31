package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

func (g GlobalConfig) ECSMetadata() (*ECSMetadata, error) {
	if g.ecsMetadata != nil {
		return g.ecsMetadata, nil
	}

	var data []byte
	if g.ECSContainerMetadata == "" {

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
	} else {
		data = []byte(g.ECSContainerMetadata)
	}

	log.Debugln("ECS Metadata:")
	log.Debugln(string(data))

	var md ECSMetadata
	if err := json.Unmarshal(data, &md); err != nil {
		return nil, fmt.Errorf("unmarshal ecs metadata: %w", err)
	}

	g.ecsMetadata = &md

	return g.ecsMetadata, nil
}

type ServerConfig struct {
	ServerHost string
	ServerPort int
	PeerHost   string
	PeerPort   int
	FullRT     bool
	DHTServer  bool
	Fleet      string
	LevelDB    string
}

var Server = ServerConfig{
	ServerHost: "localhost",
	ServerPort: 7070,
	PeerPort:   4001,
	Fleet:      "",
	FullRT:     false,
	DHTServer:  false,
	LevelDB:    "./leveldb",
}

type SchedulerConfig struct {
	Fleets *cli.StringSlice
}

var Scheduler = SchedulerConfig{
	Fleets: cli.NewStringSlice(),
}
