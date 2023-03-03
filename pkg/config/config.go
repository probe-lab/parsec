package config

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/arn"
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

	return newConfig
}

type ScheduleConfig struct {
	GlobalConfig
	ServerHost       string
	ServerPort       int
	DryRun           bool
	DatabaseHost     string
	DatabasePort     int
	DatabaseName     string
	DatabasePassword string
	DatabaseUser     string
	DatabaseSSLMode  string
}

var DefaultScheduleConfig = ScheduleConfig{
	GlobalConfig:     DefaultGlobalConfig,
	ServerHost:       "localhost",
	ServerPort:       7070,
	DatabaseHost:     "localhost",
	DatabasePort:     5432,
	DatabaseName:     "parsec",
	DatabasePassword: "password",
	DatabaseUser:     "parsec",
	DatabaseSSLMode:  "disable",
}

func (sc ScheduleConfig) Apply(c *cli.Context) ScheduleConfig {
	newConfig := sc

	newConfig.GlobalConfig = newConfig.GlobalConfig.Apply(c)

	if c.IsSet("server-host") {
		newConfig.ServerHost = c.String("server-host")
	}
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

	return newConfig
}

type ScheduleDockerConfig struct {
	ScheduleConfig
	Nodes int
}

var DefaultScheduleDockerConfig = ScheduleDockerConfig{
	ScheduleConfig: DefaultScheduleConfig,
	Nodes:          5,
}

func (sdc ScheduleDockerConfig) Apply(c *cli.Context) ScheduleDockerConfig {
	newConfig := sdc

	newConfig.ScheduleConfig = newConfig.ScheduleConfig.Apply(c)

	if c.IsSet("nodes") {
		newConfig.Nodes = c.Int("nodes")
	}

	return newConfig
}

type ScheduleAWSConfig struct {
	ScheduleConfig
	NodeAgent                string
	InstanceType             string
	KeyNames                 []string
	Regions                  []string
	PublicSubnetIDs          []string
	InstanceProfileARNs      []arn.ARN
	InstanceSecurityGroupIDs []string
	S3BucketARNs             []arn.ARN
}

var DefaultScheduleAWSConfig = ScheduleAWSConfig{
	ScheduleConfig: DefaultScheduleConfig,
}

func (sac ScheduleAWSConfig) Apply(c *cli.Context) (ScheduleAWSConfig, error) {
	newConfig := sac

	newConfig.ScheduleConfig = newConfig.ScheduleConfig.Apply(c)

	if c.IsSet("nodeagent") {
		newConfig.NodeAgent = c.String("nodeagent")
	}
	if c.IsSet("instance-type") {
		newConfig.InstanceType = c.String("instance-type")
	}
	if c.IsSet("key-names") {
		newConfig.KeyNames = c.StringSlice("key-names")
	}
	if c.IsSet("regions") {
		newConfig.Regions = c.StringSlice("regions")
	}
	if c.IsSet("public-subnet-ids") {
		newConfig.PublicSubnetIDs = c.StringSlice("public-subnet-ids")
	}

	if c.IsSet("instance-profile-arns") {
		for _, arnStr := range c.StringSlice("instance-profile-arns") {
			iparn, err := arn.Parse(arnStr)
			if err != nil {
				return ScheduleAWSConfig{}, fmt.Errorf("error parsing instnace profile arn: %w", err)
			}
			newConfig.InstanceProfileARNs = append(newConfig.InstanceProfileARNs, iparn)
		}
	}

	if c.IsSet("s3-bucket-arns") {
		for _, arnStr := range c.StringSlice("s3-bucket-arns") {
			s3arn, err := arn.Parse(arnStr)
			if err != nil {
				return ScheduleAWSConfig{}, fmt.Errorf("error parsing s3 bucket arn: %w", err)
			}
			newConfig.S3BucketARNs = append(newConfig.S3BucketARNs, s3arn)
		}
	}

	if c.IsSet("instance-security-group-ids") {
		newConfig.InstanceSecurityGroupIDs = c.StringSlice("instance-security-group-ids")
	}

	return newConfig, nil
}
