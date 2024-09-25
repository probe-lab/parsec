package dht

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"time"

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
	"github.com/ipni/go-libipni/find/client"
	"github.com/ipni/go-libipni/find/model"
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

	eng, err := h.newIndexerEngine(ctx, indexerHost)
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

func (h *Host) newIndexerEngine(ctx context.Context, url string) (*engine.Engine, error) {
	log.WithField("indexer", url).Infoln("Starting new indexer engine")

	opts := []engine.Option{
		engine.WithHost(h.BasicHost),
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
	entry, found := h.multihashes[string(contextID)]
	if !found {
		return nil, fmt.Errorf("unknown context ID")
	}

	return provider.SliceMultihashIterator(entry.mhs), nil
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
	probeCount := 300 // 1 per second for the 5-minutes negative cache
	probes, err := genProbes(c.Hash(), probeCount)
	if err != nil {
		return 0, fmt.Errorf("gen probes digest: %w", err)
	}
	logEntry.Infoln("  Probes:", probeCount)

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
		h.multihashes[string(contextID)] = multiHashEntry{
			ts:  time.Now(),
			mhs: append([]mh.Multihash{c.Hash()}, probes...),
		}
	}
	h.multihashesLk.Unlock()

	// Notify engine of new data
	logEntry = logEntry.WithField("ctxID", fmtContextID(contextID))
	logEntry.Infoln("Notify Engine")

	prov := &peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}

	start := time.Now()
	adCid, err := h.indexer.engine.NotifyPut(ctx, prov, contextID, metadata.Default.New(metadata.Bitswap{}))
	if err != nil {
		return 0, fmt.Errorf("notify engine: %w", err)
	}

	logEntry.Infoln("Put CID!")
	logEntry.Infoln("  Advertisement:  ", adCid.String())

	done := make(chan struct{})
	// wait for indexer to reach out to us
	unsubscribe := h.indexer.dtManager.SubscribeToEvents(func(evt datatransfer.Event, state datatransfer.ChannelState) {
		if !state.BaseCID().Equals(adCid) {
			return
		}

		logEntry := log.WithFields(log.Fields{
			"otherID": state.OtherPeer().String(),
			"code":    evt.Code.String(),
			"status":  state.Status().String(),
		})
		logEntry.Debugln("Received event")

		if evt.Code == datatransfer.CleanupComplete {
			done <- struct{}{}
		}
	})

	logEntry.Infoln("Waiting for indexer to reach out")
	select {
	case <-ctx.Done():
		logEntry.Infoln("Indexer did not reach out/complete the sync with us.")
	case <-done:
		logEntry.Infoln("Indexer reached out to us!")
	}
	duration := time.Since(start)

	go func() {
		logEntry.Infoln("Draining events...")
		for range done {
			logEntry.Infoln("Drained cleanup complete event")
		}
		logEntry.Infoln("Draining events done!")
	}()
	logEntry.Infoln("Unsubscribing from GraphSync events")
	unsubscribe()
	close(done)
	logEntry.Infoln("Unsubscribing done!")

	if ctx.Err() != nil {
		return duration, ctx.Err()
	}

	probeIdx := 0
	for {
		logEntry.Infoln("Pausing for 1s")
		select {
		case <-ctx.Done():
			// at this point, it's likely that the data was already indexed
			// so even if the context was cancelled, don't return the error
			logEntry.Infoln("Context cancelled")
			return duration, nil
		case <-time.After(time.Second):
		}

		logEntry.WithField("probeIdx", probeIdx).Infoln("Getting probe Multihash")

		resp, err := h.indexer.client.Find(ctx, probes[probeIdx])
		// update probe index for new round
		probeIdx = (probeIdx + 1) % probeCount
		if aErr, ok := err.(*apierror.Error); ok && aErr.Status() == http.StatusNotFound {
			logEntry.Infoln("Probe Multihash not found")
			continue
		} else if err != nil {
			return 0, fmt.Errorf("get probe multihash %s: %w", h.indexer.hostname, err)
		} else {
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
