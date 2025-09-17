package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/fullrt"
	pb "github.com/libp2p/go-libp2p-kad-dht/pb"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/multiformats/go-multiaddr"
	mh "github.com/multiformats/go-multihash"
	log "github.com/sirupsen/logrus"

	"github.com/probe-lab/parsec/pkg/util"
)

type DHTServerConfig struct {
	Badbits           string
	DeniedCIDs        string
	FirehoseRPCEvents bool
	ServerMode        bool
	FullRT            bool
	OptProv           bool
}

type DHTServer struct {
	Host *Host
	DHT  routing.Routing
	conf *DHTServerConfig

	mapMu         sync.RWMutex
	badbitsMap    map[string]struct{}
	deniedCIDsMap map[string]string
}

var _ IServer = (*DHTServer)(nil)

func InitDHTServer(ctx context.Context, h *Host, ds datastore.Batching, conf *DHTServerConfig) (*DHTServer, error) {
	badbitsMap, err := loadBadbits(conf.Badbits)
	if err != nil {
		return nil, fmt.Errorf("load badbits: %w", err)
	}

	deniedCIDsMap, err := loadDeniedCIDs(conf.DeniedCIDs)
	if err != nil {
		return nil, fmt.Errorf("load denied CIDs: %w", err)
	}

	d := &DHTServer{
		conf:          conf,
		Host:          h,
		badbitsMap:    badbitsMap,
		deniedCIDsMap: deniedCIDsMap,
	}

	mode := kaddht.ModeClient
	if conf.ServerMode {
		mode = kaddht.ModeServer
	}

	var dht routing.Routing
	if conf.FullRT {
		log.Infoln("Using full accelerated DHT strategy")
		opts := []kaddht.Option{
			kaddht.BootstrapPeers(kaddht.GetDefaultBootstrapPeerAddrInfos()...),
			kaddht.Mode(mode),
			kaddht.Datastore(ds),
		}
		if conf.FirehoseRPCEvents {
			opts = append(opts, kaddht.OnRequestHook(d.handlerWrapper))
		}

		dht, err = fullrt.NewFullRT(h, ipfsProtocolPrefix, fullrt.DHTOption(opts...))
	} else {
		log.Infoln("Using default DHT strategy")
		opts := []kaddht.Option{
			kaddht.Mode(mode),
			kaddht.Datastore(ds),
			kaddht.OnRequestHook(d.handlerWrapper),
		}
		if conf.OptProv {
			opts = append(opts, kaddht.EnableOptimisticProvide())
		}
		if conf.FirehoseRPCEvents {
			opts = append(opts, kaddht.OnRequestHook(d.handlerWrapper))
		}
		dht, err = kaddht.New(ctx, h, opts...)
	}
	if err != nil {
		return nil, fmt.Errorf("new router: %w", err)
	}
	d.DHT = dht

	log.Infoln("Bootstrapping DHT server")
	if err = dht.Bootstrap(ctx); err != nil {
		return nil, fmt.Errorf("bootstrap: %w", err)
	}

	return d, nil
}

func (d *DHTServer) Shutdown(ctx context.Context) error {
	return d.Host.Host.Close()
}

func (d *DHTServer) Provide(ctx context.Context, cid cid.Cid) (*ProvideResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	start := time.Now()
	err := d.DHT.Provide(timeoutCtx, cid, true)
	duration := time.Since(start)

	latencies.WithLabelValues("provide_duration", "dht", strconv.FormatBool(err == nil), ctx.Value(headerSchedulerID).(string)).Observe(duration.Seconds())
	log.WithField("cid", cid.String()).Infoln("Done providing content...")

	resp := &ProvideResponse{
		CID: cid.String(),
		Measurements: []*Measurement{
			{
				Step:     "provide",
				Duration: duration,
				Error:    fmtErr(err),
			},
		},
		RoutingTableSize: RoutingTableSize(d.DHT),
	}

	return resp, nil
}

func (d *DHTServer) Retrieve(ctx context.Context, cid cid.Cid) (*RetrievalResponse, error) {
	start := time.Now()
	provider := <-d.DHT.FindProvidersAsync(ctx, cid, 1)
	duration := time.Since(start)

	logEntry := log.WithField("dur", duration.Seconds())

	var err error
	if errors.Is(provider.ID.Validate(), peer.ErrEmptyPeerID) {
		err = fmt.Errorf("not found")
		logEntry.Infoln("Didn't find provider")
	} else {
		d.Host.Network().ClosePeer(provider.ID)
		d.Host.Peerstore().RemovePeer(provider.ID)
		d.Host.Peerstore().ClearAddrs(provider.ID)
		logEntry.WithField("provider", util.FmtPeerID(provider.ID)).Infoln("Found provider")
	}

	resp := &RetrievalResponse{
		CID: cid.String(),
		Measurements: []*Measurement{{
			Duration: duration,
			Error:    fmtErr(err),
		}},
		RoutingTableSize: RoutingTableSize(d.DHT),
	}

	latencies.WithLabelValues("retrieval_ttfpr", "dht", strconv.FormatBool(err == nil), ctx.Value(headerSchedulerID).(string)).Observe(duration.Seconds())
	return resp, nil
}

func (d *DHTServer) Readiness() bool {
	return true
}

func (d *DHTServer) handlerWrapper(ctx context.Context, s network.Stream, req *pb.Message) {
	switch req.GetType() {
	case pb.Message_ADD_PROVIDER, pb.Message_GET_PROVIDERS:
		go func() {
			_, mh, err := mh.MHFromBytes(req.GetKey())
			if err != nil {
				log.WithError(err).Warnln("failed to parse multihash from key")
				return
			}

			rec := &RPCRequest{
				MessageType: req.GetType().String(),
				Multihash:   mh.String(),
			}

			d.mapMu.RLock()
			for _, codec := range codecs {
				v1 := cid.NewCidV1(uint64(codec), mh)

				if source, found := d.deniedCIDsMap[v1.String()]; found {
					rec.Match = v1.String()
					rec.Source = source
					log.WithField("source", source).Infof("Found CID in %s!", source)
					break
				}

				preimage := v1.String() + "/"
				hsh := sha256.Sum256([]byte(preimage))
				matchStr := hex.EncodeToString(hsh[:])

				if _, found := d.badbitsMap[matchStr]; found {
					rec.Match = preimage
					rec.Source = "badbits"
					log.WithField("preimage", preimage).Infoln("Preimage found!", s.Conn().RemotePeer())
					break
				}
			}
			d.mapMu.RUnlock()

			addrInfo := peer.AddrInfo{
				ID:    s.Conn().RemotePeer(),
				Addrs: []multiaddr.Multiaddr{s.Conn().RemoteMultiaddr()},
			}

			if err := d.Host.fhClient.Submit("dht_rpc", d.Host.ID(), addrInfo, d.Host.AgentVersion(s.Conn().RemotePeer()), rec); err != nil {
				log.WithError(err).Warnln("Couldn't submit add_provider event")
			}
		}()
	default:
	}
}
