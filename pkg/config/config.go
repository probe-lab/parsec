package config

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// DefaultConfig the default configuration.
var DefaultConfig = Config{

	TelemetryHost:    "0.0.0.0",
	TelemetryPort:    6666,
	DatabaseHost:     "0.0.0.0",
	DatabasePort:     5432,
	DatabaseName:     "nebula",
	DatabasePassword: "password",
	DatabaseUser:     "nebula",
	DatabaseSSLMode:  "disable",
}

// Config contains general user configuration.
type Config struct {
	// The version string of nebula
	Version string `json:"-"`

	// Determines where the prometheus and pprof hosts should bind to.
	TelemetryHost string

	// Determines the port where prometheus and pprof serve the metrics endpoint.
	TelemetryPort int

	// Determines the host address of the database.
	DatabaseHost string

	// Determines the port of the database.
	DatabasePort int

	// Determines the name of the database that should be used.
	DatabaseName string

	// Determines the password with which we access the database.
	DatabasePassword string

	// Determines the username with which we access the database.
	DatabaseUser string

	// Postgres SSL mode (should be one supported in https://www.postgresql.org/docs/current/libpq-ssl.html)
	DatabaseSSLMode string
}

// Init takes the command line argument and tries to read the config file from that directory.
func Init(c *cli.Context) (*Config, error) {
	conf := DefaultConfig

	// Apply command line argument configurations.
	conf.apply(c)

	// Print full configuration.
	log.Debugln("Configuration (CLI params overwrite file config):\n", conf)

	// Populate the context with the configuration.
	return &conf, nil
}

// String prints the configuration as a json string
func (c *Config) String() string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("%s", data)
}

// DatabaseSourceName returns the data source name string to be put into the sql.Open method.
func (c *Config) DatabaseSourceName() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.DatabaseHost,
		c.DatabasePort,
		c.DatabaseName,
		c.DatabaseUser,
		c.DatabasePassword,
		c.DatabaseSSLMode,
	)
}

// apply takes command line arguments and overwrites the respective configurations.
func (c *Config) apply(ctx *cli.Context) {
	c.Version = ctx.App.Version

	if ctx.IsSet("telemetry-host") {
		c.TelemetryHost = ctx.String("telemetry-host")
	}
	if ctx.IsSet("telemetry-port") {
		c.TelemetryPort = ctx.Int("telemetry-port")
	}
	if ctx.IsSet("db-host") {
		c.DatabaseHost = ctx.String("db-host")
	}
	if ctx.IsSet("db-port") {
		c.DatabasePort = ctx.Int("db-port")
	}
	if ctx.IsSet("db-name") {
		c.DatabaseName = ctx.String("db-name")
	}
	if ctx.IsSet("db-password") {
		c.DatabasePassword = ctx.String("db-password")
	}
	if ctx.IsSet("db-user") {
		c.DatabaseUser = ctx.String("db-user")
	}
	if ctx.IsSet("db-sslmode") {
		c.DatabaseSSLMode = ctx.String("db-sslmode")
	}
}
