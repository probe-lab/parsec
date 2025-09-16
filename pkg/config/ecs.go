package config

import (
	"net"

	log "github.com/sirupsen/logrus"
)

type ECSMetadata struct {
	DockerID      string            `json:"DockerId"`
	Name          string            `json:"Name"`
	DockerName    string            `json:"DockerName"`
	Image         string            `json:"Image"`
	ImageID       string            `json:"ImageID"`
	Labels        map[string]string `json:"Labels"`
	DesiredStatus string            `json:"DesiredStatus"`
	KnownStatus   string            `json:"KnownStatus"`
	Limits        Limits            `json:"Limits"`
	CreatedAt     string            `json:"CreatedAt"`
	StartedAt     string            `json:"StartedAt"`
	Type          string            `json:"Type"`
	LogDriver     string            `json:"LogDriver"`
	LogOptions    LogOptions        `json:"LogOptions"`
	ContainerARN  string            `json:"ContainerARN"`
	Networks      []Network         `json:"Networks"`
}

type Limits struct {
	CPU    int64 `json:"CPU"`
	Memory int64 `json:"Memory"`
}

type LogOptions struct {
	AWSLogsCreateGroup string `json:"awslogs-create-group"`
	AWSLogsGroup       string `json:"awslogs-group"`
	AWSLogsRegion      string `json:"awslogs-region"`
	AWSLogsStream      string `json:"awslogs-stream"`
}

type Network struct {
	NetworkMode              string   `json:"NetworkMode"`
	IPv4Addresses            []string `json:"IPv4Addresses"`
	AttachmentIndex          int64    `json:"AttachmentIndex"`
	MACAddress               string   `json:"MACAddress"`
	IPv4SubnetCIDRBlock      string   `json:"IPv4SubnetCIDRBlock"`
	PrivateDNSName           string   `json:"PrivateDNSName"`
	SubnetGatewayIpv4Address string   `json:"SubnetGatewayIpv4Address"`
}

func (ecs *ECSMetadata) GetPrivateIP() net.IP {
	for _, network := range ecs.Networks {
		if network.NetworkMode != "awsvpc" {
			log.WithField("network", network).Warnln("Found network with mode", network.NetworkMode)
			continue
		}

		if len(network.IPv4Addresses) != 1 {
			log.WithField("network", network).Warnf("Found network with %d IPv4 addresses", len(network.IPv4Addresses))
			continue
		}

		ip := net.ParseIP(network.IPv4Addresses[0])
		if ip == nil {
			log.WithField("ip", network.IPv4Addresses[0]).Warnln("Error parsing IP")
		}

		return ip
	}

	return nil
}
