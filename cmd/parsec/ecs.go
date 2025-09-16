package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

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

// processMetadata returns the available CPU and memory of the container.
// If the container is running on ECS, it will try to get the metadata from the
// environment variable ECS_CONTAINER_METADATA_URI_V4. If the environment variable
// is not set, it will try to get the metadata from the environment variable
// ECS_CONTAINER_METADATA. If both are not set, it will try to get the metadata
// from the hostname. If the hostname is not set, it will return an error.
// If the metadata is successfully retrieved, it will return the available CPU
// and memory of the container. If the metadata is not successfully retrieved,
// it will return an error.
func processMetadata() (int64, int64, net.IP, error) {
	var data []byte
	if rootConfig.ECSContainerMetadataURIV4 != "" && rootConfig.ECSContainerMetadata == "" {

		resp, err := http.Get(rootConfig.ECSContainerMetadataURIV4)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("get metadata from %s: %w", rootConfig.ECSContainerMetadataURIV4, err)
		}

		if resp.StatusCode != http.StatusOK {
			return 0, 0, nil, fmt.Errorf("resp status code %d", resp.StatusCode)
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("reading ecs metadata request body: %w", err)
		}
	} else if rootConfig.ECSContainerMetadata != "" {
		data = []byte(rootConfig.ECSContainerMetadata)
	} else {

		hostname, err := os.Hostname()
		if err != nil {
			return 0, 0, nil, fmt.Errorf("get hostname: %w", err)
		}
		addrs, err := net.LookupHost(hostname)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("lookup host %s: %w", hostname, err)
		}

		for _, addr := range addrs {
			log.Infoln("Detected private addr:", addr)
		}

		if len(addrs) == 0 {
			return 0, 0, nil, fmt.Errorf("no private ip found")
		}

		ip := net.ParseIP(addrs[0])
		if ip == nil {
			return 0, 0, nil, fmt.Errorf("parse ip: %s", addrs[0])
		}
		return 1, 1, ip, nil
	}

	log.Debugln("ECS Metadata:")
	log.Debugln(string(data))

	var md ECSMetadata
	if err := json.Unmarshal(data, &md); err != nil {
		return 0, 0, nil, fmt.Errorf("unmarshal ecs metadata: %w", err)
	}

	return md.Limits.CPU, md.Limits.Memory, md.GetPrivateIP(), nil
}
