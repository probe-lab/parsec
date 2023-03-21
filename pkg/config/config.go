package config

import (
	"github.com/urfave/cli/v2"
)

type GlobalConfig struct {
	Version       string
	TelemetryHost string
	TelemetryPort int
	Debug         bool
	LogLevel      int
}

var DefaultGlobalConfig = GlobalConfig{
	TelemetryHost: "0.0.0.0",
	TelemetryPort: 6666,
	Debug:         false,
	LogLevel:      4,
}

func (gc GlobalConfig) Apply(c *cli.Context) GlobalConfig {
	newConfig := gc

	newConfig.Version = c.App.Version

	if c.IsSet("debug") {
		newConfig.Debug = c.Bool("debug")
	}
	if c.IsSet("log-level") {
		newConfig.LogLevel = c.Int("log-level")
	}
	if c.IsSet("telemetry-host") {
		newConfig.TelemetryHost = c.String("telemetry-host")
	}
	if c.IsSet("telemetry-port") {
		newConfig.TelemetryPort = c.Int("telemetry-port")
	}

	return newConfig
}

type ServerConfig struct {
	GlobalConfig
	ServerHost string
	ServerPort int
	PeerHost   string
	PeerPort   int
	FullRT     bool
}

var DefaultServerConfig = ServerConfig{
	GlobalConfig: DefaultGlobalConfig,
	ServerHost:   "localhost",
	ServerPort:   7070,
	PeerPort:     4001,
}

func (sc ServerConfig) Apply(c *cli.Context) ServerConfig {
	newConfig := sc

	newConfig.GlobalConfig = newConfig.GlobalConfig.Apply(c)

	if c.IsSet("server-host") {
		newConfig.ServerHost = c.String("server-host")
	}
	if c.IsSet("server-port") {
		newConfig.ServerPort = c.Int("server-port")
	}
	if c.IsSet("peer-port") {
		newConfig.PeerPort = c.Int("peer-port")
	}
	if c.IsSet("fullrt") {
		newConfig.FullRT = c.Bool("fullrt")
	}

	return newConfig
}

type SchedulerConfig struct {
	GlobalConfig
	ServerPort       int
	DryRun           bool
	DatabaseHost     string
	DatabasePort     int
	DatabaseName     string
	DatabasePassword string
	DatabaseUser     string
	DatabaseSSLMode  string
	FullRT           bool
}

var DefaultSchedulerConfig = SchedulerConfig{
	GlobalConfig:     DefaultGlobalConfig,
	ServerPort:       7070,
	DatabaseHost:     "localhost",
	DatabasePort:     5432,
	DatabaseName:     "parsec",
	DatabasePassword: "password",
	DatabaseUser:     "parsec",
	DatabaseSSLMode:  "disable",
}

func (sc SchedulerConfig) Apply(c *cli.Context) SchedulerConfig {
	newConfig := sc

	newConfig.GlobalConfig = newConfig.GlobalConfig.Apply(c)

	if c.IsSet("server-port") {
		newConfig.ServerPort = c.Int("server-port")
	}
	if c.IsSet("dry-run") {
		newConfig.DryRun = c.Bool("dry-run")
	}
	if c.IsSet("db-host") {
		newConfig.DatabaseHost = c.String("db-host")
	}
	if c.IsSet("db-port") {
		newConfig.DatabasePort = c.Int("db-port")
	}
	if c.IsSet("db-name") {
		newConfig.DatabaseName = c.String("db-name")
	}
	if c.IsSet("db-password") {
		newConfig.DatabasePassword = c.String("db-password")
	}
	if c.IsSet("db-user") {
		newConfig.DatabaseUser = c.String("db-user")
	}
	if c.IsSet("db-sslmode") {
		newConfig.DatabaseSSLMode = c.String("db-sslmode")
	}
	if c.IsSet("fullrt") {
		newConfig.FullRT = c.Bool("fullrt")
	}

	return newConfig
}
