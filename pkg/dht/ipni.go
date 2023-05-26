package dht

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/ipni/go-libipni/find/model"

	"github.com/cenkalti/backoff/v4"
	datatransfer "github.com/filecoin-project/go-data-transfer/v2"
	dtimpl "github.com/filecoin-project/go-data-transfer/v2/impl"
	dtnetwork "github.com/filecoin-project/go-data-transfer/v2/network"
	gstransport "github.com/filecoin-project/go-data-transfer/v2/transport/graphsync"
	"github.com/ipfs/go-cid"
	leveldb "github.com/ipfs/go-ds-leveldb"
	gsimpl "github.com/ipfs/go-graphsync/impl"
	gsnet "github.com/ipfs/go-graphsync/network"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipni/go-libipni/apierror"
	client "github.com/ipni/go-libipni/find/client/http"
	"github.com/ipni/go-libipni/metadata"
	provider "github.com/ipni/index-provider"
	"github.com/ipni/index-provider/engine"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
	log "github.com/sirupsen/logrus"
)

type Indexer struct {
	hostname  string
	client    *client.Client
	engine    *engine.Engine
	dtManager datatransfer.Manager
}

func (h *Host) initIndexer(ctx context.Context, ds *leveldb.Datastore, indexerHost string) (*Indexer, error) {
	log.Infoln("Init indexer", indexerHost)
	gsnet := gsnet.NewFromLibp2pHost(h.Host)
	dtNet := dtnetwork.NewFromLibp2pHost(h.Host)
	gs := gsimpl.New(context.Background(), gsnet, cidlink.DefaultLinkSystem())
	tp := gstransport.NewTransport(h.Host.ID(), gs)
	dt, err := dtimpl.NewDataTransfer(ds, dtNet, tp)
	if err != nil {
		return nil, fmt.Errorf("new data transfer: %w", err)
	}
	err = dt.Start(context.Background())
	if err != nil {
		return nil, fmt.Errorf("start data transfer: %w", err)
	}

	eng, err := h.newIndexerEngine(ctx, dt, indexerHost)
	if err != nil {
		return nil, fmt.Errorf("new indexer engine for %s: %w", indexerHost, err)
	}

	c, err := client.New("https://" + indexerHost)
	if err != nil {
		return nil, fmt.Errorf("new indexer client for %s: %w", indexerHost, err)
	}

	return &Indexer{
		hostname:  indexerHost,
		client:    c,
		engine:    eng,
		dtManager: dt,
	}, nil
}

func (h *Host) newIndexerEngine(ctx context.Context, dtManager datatransfer.Manager, url string) (*engine.Engine, error) {
	log.WithField("indexer", url).Infoln("Starting new indexer engine")

	opts := []engine.Option{
		engine.WithHost(h.BasicHost),
		engine.WithDataTransfer(dtManager),
		engine.WithPublisherKind(engine.DataTransferPublisher),
		engine.WithDirectAnnounce(fmt.Sprintf("https://%s/ingest/announce", url)),
	}

	eng, err := engine.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("new provider engine")
	}

	eng.RegisterMultihashLister(h.multiHashLister)

	if err = eng.Start(ctx); err != nil {
		return nil, fmt.Errorf("start provider engine: %w", err)
	}

	return eng, nil
}

func (h *Host) multiHashLister(ctx context.Context, p peer.ID, contextID []byte) (provider.MultihashIterator, error) {
	h.multihashesLk.RLock()
	defer h.multihashesLk.RUnlock()

	// Looking up the multihash for the given contextID
	m, found := h.multihashes[string(contextID)]
	if !found {
		return nil, fmt.Errorf("unknown context ID")
	}

	// Generate probe multihash by hashing the multihash again
	probe, err := genProbeMultihash(m)
	if err != nil {
		return nil, fmt.Errorf("gen probe digest")
	}

	return provider.SliceMultihashIterator([]mh.Multihash{m, probe}), nil
}

func (h *Host) IndexerLookup(ctx context.Context, c cid.Cid) (*model.FindResponse, error) {
	if h.indexer == nil {
		return nil, fmt.Errorf("no indexer configured")
	}

	return h.indexer.client.Find(ctx, c.Hash())
}

func (h *Host) Announce(ctx context.Context, c cid.Cid) (time.Duration, error) {
	if h.indexer == nil {
		return 0, fmt.Errorf("no indexer configured")
	}

	logEntry := log.WithField("indexer", h.indexer.hostname)
	logEntry.Infoln("Announcing CID...")
	logEntry.Infoln("  CID:  ", c.String())

	// Generating multihash for probing after we have pushed out the advertisement.
	// Usually we would be able to check the indexing status by polling the
	// /providers endpoint. However, the response is cached for 1h. So, what we
	// are doing instead is deriving a "probe" multihash. This means we are
	// publishing the given CID + the probe multihash. Then we poll the API for
	// the probe multihash. If that hash is indexed, we can be sure that the
	// given CID was also indexed
	probe, err := genProbeMultihash(c.Hash())
	if err != nil {
		return 0, fmt.Errorf("gen probe digest: %w", err)
	}
	logEntry.Infoln("  Probe:", probe.B58String())

	// The contextID just needs to be unique per multihash
	contextID := []byte(fmt.Sprintf("parsec-%d", rand.Uint64()))
	logEntry.Infoln("  CtxID:", base64.StdEncoding.EncodeToString(contextID))

	// Now, we are storing the multihash in our in memory "database". If this is
	// a "delete" advertisement, we are removing it.
	h.multihashesLk.Lock()
	if _, found := h.multihashes[string(contextID)]; found {
		h.multihashesLk.Unlock()
		return 0, provider.ErrAlreadyAdvertised
	} else {
		h.multihashes[string(contextID)] = c.Hash()
	}
	h.multihashesLk.Unlock()

	// Notify engine of new data
	logEntry = logEntry.WithField("probe", fmtMultihash(probe)).WithField("ctxID", fmtContextID(contextID))
	logEntry.Infoln("Notify Engine")

	start := time.Now()
	adCid, err := h.indexer.engine.NotifyPut(ctx, nil, contextID, metadata.Default.New(metadata.Bitswap{}))
	if err != nil {
		return 0, fmt.Errorf("notify engine: %w", err)
	}

	logEntry.Infoln("Put CID!")
	logEntry.Infoln("  Advertisement:  ", adCid.String())

	done := make(chan struct{})
	// wait for indexer to reach out to us
	unsubFn := h.indexer.dtManager.SubscribeToEvents(func(evt datatransfer.Event, state datatransfer.ChannelState) {
		if !state.BaseCID().Equals(adCid) {
			return
		}
		logEntry.WithField("peerID", state.OtherPeer().String()[:16]).WithField("code", evt.Code.String()).Debugln("Received event")
		if evt.Code == datatransfer.CleanupComplete {
			done <- struct{}{}
		}
	})

	logEntry.Infoln("Waiting for indexer to reach out")
	select {
	case <-ctx.Done():
	case <-done:
		logEntry.Infoln("Indexer to reached out to us!")
	}
	duration := time.Since(start)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		drainChannel(done)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		unsubFn()
		wg.Done()
	}()

	wg.Wait()
	drainChannel(done)
	close(done)

	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	bo := backoff.NewExponentialBackOff()
	bo.RandomizationFactor = 0
	bo.Multiplier = 1.3
	bo.MaxInterval = 10 * time.Second
	bo.MaxElapsedTime = 5 * time.Minute

	select {
	case <-ctx.Done():
		return duration, nil
	case <-time.After(10 * time.Second):
	}

	for {
		select {
		case <-ctx.Done():
			// at this point, it's likely that the data was already indexed
			// so even if the context was cancelled, don't return the error
			logEntry.Infoln("Context cancelled")
			return duration, nil
		case <-time.After(bo.NextBackOff()):
		}

		logEntry.Infoln("Getting probe Multihash")

		resp, err := h.indexer.client.Find(ctx, probe)
		if aErr, ok := err.(*apierror.Error); ok && aErr.Status() == http.StatusNotFound {
			logEntry.Infoln("Probe Multihash not found")
			continue
		} else if err != nil {
			return 0, fmt.Errorf("get probe multihash %s: %w", h.indexer.hostname, err)
		} else {
			logEntry.Infoln("Checking presence of provider ID")
			for _, mhRes := range resp.MultihashResults {
				for _, pRes := range mhRes.ProviderResults {
					if pRes.Provider.ID == h.ID() {
						logEntry.Infoln("Provider Found!")
						return duration, nil
					}
				}
			}

			logEntry.Infoln("Provider not in response")
			continue
		}
	}
}

func drainChannel[T any](c chan T) {
	for {
		select {
		case <-c:
			continue
		default:
			return
		}
	}
}
