package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/probe-lab/go-commons/maddr"
	"github.com/urfave/cli/v2"

	"github.com/probe-lab/go-commons/p2p"
	"github.com/probe-lab/parsec/pkg/server"
)

var serverDHTConfig = struct {
	Project    string
	Network    string
	FullRT     bool
	OptProv    bool
	ServerMode bool
	Badbits    string
	DeniedCIDs string
	Alpha      int
}{
	Project:    "ipfs",
	Network:    "amino",
	FullRT:     false,
	OptProv:    false,
	ServerMode: false,
	Alpha:      10,
	Badbits:    "",
	DeniedCIDs: "",
}

// ServerDHTCommand contains the crawl sub-command configuration.
var ServerDHTCommand = &cli.Command{
	Name:   "dht",
	Action: ServerDHTAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "project",
			EnvVars:     []string{"PARSEC_SERVER_DHT_PROJECT"},
			Value:       serverDHTConfig.Project,
			Destination: &serverDHTConfig.Project,
		},
		&cli.StringFlag{
			Name:        "network",
			EnvVars:     []string{"PARSEC_SERVER_DHT_NETWORK"},
			Value:       serverDHTConfig.Network,
			Destination: &serverDHTConfig.Network,
		},
		&cli.BoolFlag{
			Name:        "fullrt",
			Usage:       "Whether to enable the full routing table setting on the DHT",
			EnvVars:     []string{"PARSEC_SERVER_DHT_FULLRT"},
			DefaultText: strconv.FormatBool(serverDHTConfig.FullRT),
			Value:       serverDHTConfig.FullRT,
			Destination: &serverDHTConfig.FullRT,
		},
		&cli.BoolFlag{
			Name:        "optprov",
			Usage:       "Whether to enable optimistic provide",
			EnvVars:     []string{"PARSEC_SERVER_DHT_OPTPROV"},
			DefaultText: strconv.FormatBool(serverDHTConfig.OptProv),
			Value:       serverDHTConfig.OptProv,
			Destination: &serverDHTConfig.OptProv,
		},
		&cli.BoolFlag{
			Name:        "server-mode",
			Usage:       "Whether to enable DHT server mode",
			EnvVars:     []string{"PARSEC_SERVER_DHT_SERVER_MODE"},
			DefaultText: strconv.FormatBool(serverDHTConfig.ServerMode),
			Value:       serverDHTConfig.ServerMode,
			Destination: &serverDHTConfig.ServerMode,
		},
		&cli.IntFlag{
			Name:        "alpha",
			Usage:       "Kademlia's concurrency parameter",
			EnvVars:     []string{"PARSEC_SERVER_DHT_ALPHA"},
			Value:       serverDHTConfig.Alpha,
			Destination: &serverDHTConfig.Alpha,
		},
		&cli.StringFlag{
			Name:        "badbits",
			EnvVars:     []string{"PARSEC_SERVER_DHT_BADBITS"},
			DefaultText: serverDHTConfig.Badbits,
			Value:       serverDHTConfig.Badbits,
			Destination: &serverDHTConfig.Badbits,
		},
		&cli.StringFlag{
			Name:        "denied-cids",
			EnvVars:     []string{"PARSEC_SERVER_DHT_DENIED_CIDS"},
			DefaultText: serverDHTConfig.DeniedCIDs,
			Value:       serverDHTConfig.DeniedCIDs,
			Destination: &serverDHTConfig.DeniedCIDs,
		},
	},
}

// ServerDHTAction is the function that is called when running `nebula crawl`.
func ServerDHTAction(c *cli.Context) error {
	bootstrappers, prot, err := bootstrapConfig(serverDHTConfig.Project, serverDHTConfig.Network)
	if err != nil {
		return err
	}

	config := &server.DHTServerConfig{
		Badbits:           serverDHTConfig.Badbits,
		DeniedCIDs:        serverDHTConfig.DeniedCIDs,
		ServerMode:        serverDHTConfig.ServerMode,
		FullRT:            serverDHTConfig.FullRT,
		OptProv:           serverDHTConfig.OptProv,
		Alpha:             serverDHTConfig.Alpha,
		Bootstrappers:     bootstrappers,
		Protocol:          prot,
		FirehoseRPCEvents: serverConfig.FirehoseRPCEvents,
	}
	return serverAction(c, func(ctx context.Context, h *server.Host, ds datastore.Batching) (server.IServer, error) {
		return server.InitDHTServer(ctx, h, ds, config)
	})
}

func bootstrapConfig(project string, network string) ([]peer.AddrInfo, protocol.ID, error) {
	bcfg, err := p2p.GetBootstrapConfig(p2p.Project(project), p2p.Network(network))
	if err != nil {
		return nil, "", err
	}

	bootstrappers, err := maddr.FromStrings(bcfg.Bootstrappers)
	if err != nil {
		return nil, "", err
	}

	bootstrapPeers := make([]peer.AddrInfo, 0, len(bootstrappers))
	for _, addr := range bootstrappers {
		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return nil, "", fmt.Errorf("invalid bootstrap peer address: %s: %w", addr, err)
		}
		bootstrapPeers = append(bootstrapPeers, *peerInfo)
	}
	if len(bcfg.ProtocolIDs) == 0 {
		return nil, "", fmt.Errorf("no protocol ids found in bootstrap config")
	}

	return bootstrapPeers, protocol.ID(bcfg.ProtocolIDs[0]), nil
}
