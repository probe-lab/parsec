package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipni/go-libipni/apierror"
	"github.com/ipni/go-libipni/dagsync/ipnisync"
	"github.com/ipni/go-libipni/find/client"
	"github.com/ipni/go-libipni/metadata"
	provider "github.com/ipni/index-provider"
	"github.com/ipni/index-provider/engine"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
	log "github.com/sirupsen/logrus"
)

type IPNIServerConfig struct {
	IndexerHost string

	PublisherHost string
	PublisherPort int

	IndexerInterval            time.Duration
	IndexerIndexTimeout        time.Duration
	IndexerAvailabilityTimeout time.Duration
}

func (c *IPNIServerConfig) publisherHostPort() string {
	return net.JoinHostPort(c.PublisherHost, strconv.Itoa(c.PublisherPort))
}

type IPNIServer struct {
	host host.Host
	conf *IPNIServerConfig

	client *client.Client
	engine *engine.Engine
	srv    *http.Server

	subsMu sync.RWMutex
	subs   map[string]chan ipniRequest

	multihashesLk sync.RWMutex
	multihashes   map[string]multiHashEntry
}

var _ IServer = (*IPNIServer)(nil)

func InitIPNIServer(ctx context.Context, h *Host, ds datastore.Batching, conf *IPNIServerConfig) (*IPNIServer, error) {
	log.Infoln("Init indexer", conf.IndexerHost)

	publisherAnnounceHostPort := fmt.Sprintf("/ip4/%s/tcp/%d/http", h.publicIP.Load().(string), conf.PublisherPort)
	directAnnounce := fmt.Sprintf("https://%s/ingest/announce", conf.IndexerHost)
	engine.WithRetrievalAddrs()
	opts := []engine.Option{
		engine.WithHost(h),
		engine.WithDatastore(ds),
		engine.WithHttpPublisherListenAddr(conf.publisherHostPort()),
		engine.WithPublisherKind(engine.HttpPublisher),
		engine.WithHttpPublisherAnnounceAddr(publisherAnnounceHostPort),
		engine.WithHttpPublisherWithoutServer(),
		engine.WithDirectAnnounce(directAnnounce),
	}

	if err := logging.SetLogLevel("provider/engine", logging.LevelFatal.String()); err != nil {
		log.WithError(err).Warnln("Failed to set provider/engine log level")
	}

	log.WithFields(log.Fields{
		"listenAddr":   conf.publisherHostPort(),
		"announceAddr": publisherAnnounceHostPort,
		"indexer":      directAnnounce,
	}).Infoln("Starting new indexer engine")
	eng, err := engine.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("new provider engine: %w", err)
	}

	c, err := client.New("https://" + conf.IndexerHost)
	if err != nil {
		return nil, fmt.Errorf("new indexer client for %s: %w", conf.IndexerHost, err)
	}

	if err = eng.Start(ctx); err != nil {
		return nil, fmt.Errorf("start provider engine: %w", err)
	}

	publisherHTTPFunc, err := eng.GetPublisherHttpFunc()
	if err != nil {
		return nil, fmt.Errorf("get publisher http func: %w", err)
	}

	i := &IPNIServer{
		host:        h,
		conf:        conf,
		client:      c,
		engine:      eng,
		subs:        map[string]chan ipniRequest{},
		multihashes: map[string]multiHashEntry{},
	}

	i.srv = &http.Server{
		Addr:    conf.publisherHostPort(),
		Handler: i.httpHandler(publisherHTTPFunc),
	}

	go func() {
		log.WithField("addr", conf.publisherHostPort()).Infoln("Starting publisher http host")
		if err := i.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal("serve publisher http host")
		}
	}()

	eng.RegisterMultihashLister(i.multiHashLister)

	go i.gcMultihashEntries(ctx)

	return i, nil
}

func (i *IPNIServer) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := i.srv.Shutdown(ctx); err != nil {
		log.WithError(err).WithField("indexer", i.conf.IndexerHost).Warnln("Failed to shut down publisher HTTP server")
	}

	if err := i.engine.Shutdown(); err != nil {
		log.WithError(err).WithField("indexer", i.conf.IndexerHost).Warnln("Failed to shut down indexer engine")
	}

	return i.host.Close()
}

func (i *IPNIServer) Provide(ctx context.Context, c cid.Cid) (pr *ProvideResponse, err error) {
	logEntry := log.WithField("indexer", i.conf.IndexerHost)
	logEntry.Infoln("Announcing CID...")
	logEntry.Infoln("  CID:  ", c.String())

	pr = &ProvideResponse{
		CID:              c.String(),
		Measurements:     []*Measurement{},
		RoutingTableSize: 0,
	}

	defer func() {
		if len(pr.Measurements) == 0 {
			return
		}
		m := pr.Measurements[len(pr.Measurements)-1]

		// if it's a "too many requests" error, we don't want to record the latency
		if strings.Contains(m.Error, "Too Many Requests") {
			return
		}

		latencies.WithLabelValues("provide_duration", strconv.FormatBool(m.Error == ""), ctx.Value(headerSchedulerID).(string)).Observe(m.Duration.Seconds())
	}()

	// Generating multihash for probing after we have pushed out the advertisement.
	// Usually we would be able to check the indexing status by polling the
	// /providers endpoint. However, the response is cached for 1h. So, what we
	// are doing instead is deriving a "probe" multihash. This means we are
	// publishing the given CID + the probe multihash. Then we poll the API for
	// the probe multihash. If that hash is indexed, we can be sure that the
	// given CID was also indexed

	probeCount := int(i.conf.IndexerAvailabilityTimeout/i.conf.IndexerInterval) + 1
	probes, err := genProbes(c.Hash(), probeCount)
	if err != nil {
		return nil, fmt.Errorf("gen probes digest: %w", err)
	}
	logEntry.Infoln("  Probes:", probeCount)

	// The contextID just needs to be unique per multihash
	contextID := []byte(fmt.Sprintf("parsec-%d", rand.Uint64()))
	logEntry.Infoln("  CtxID:", base64.StdEncoding.EncodeToString(contextID))
	logEntry = logEntry.WithField("ctxID", fmtContextID(contextID))

	// Now, we are storing the multihash in our in-memory "database". If this is
	// a "delete" advertisement, we are removing it.
	i.multihashesLk.Lock()
	if _, found := i.multihashes[string(contextID)]; found {
		i.multihashesLk.Unlock()
		return nil, provider.ErrAlreadyAdvertised
	} else {
		i.multihashes[string(contextID)] = multiHashEntry{
			ts:  time.Now(),
			mhs: append([]mh.Multihash{c.Hash()}, probes...),
		}
	}
	i.multihashesLk.Unlock()

	// Notify engine of new data
	prov := &peer.AddrInfo{
		ID:    i.host.ID(),
		Addrs: i.host.Addrs(),
	}

	ipniReqChan := make(chan ipniRequest)
	subID := uuid.New().String()
	logEntry.WithField("subID", subID).Debugln("Subscribing to IPNI requests")
	i.subsMu.Lock()
	i.subs[subID] = ipniReqChan
	i.subsMu.Unlock()

	unsubscribe := func() {
		// drain chan
		go func() {
			for range ipniReqChan {
				logEntry.WithField("subID", subID).Debugln("Drained IPNI request message")
			}
		}()

		i.subsMu.Lock()
		defer i.subsMu.Unlock()

		delete(i.subs, subID)
		close(ipniReqChan)
		logEntry.WithField("subID", subID).Debugln("Unsubscribed from IPNI requests")
	}

	logEntry.Infoln("Notify Engine")
	start := time.Now()
	adCid, err := i.engine.NotifyPut(ctx, prov, contextID, metadata.Default.New(metadata.Bitswap{}))
	if err != nil {
		unsubscribe()
		pr.Measurements = append(pr.Measurements, &Measurement{
			Step:     "announce",
			Duration: time.Since(start),
			Error:    "notify engine: " + err.Error(),
		})
		log.WithError(err).Warnln("Failed to notify engine")
		return pr, nil
	}

	pr.Measurements = append(pr.Measurements, &Measurement{
		Step:     "announce",
		Duration: time.Since(start),
	})

	ad, err := i.engine.GetAdv(ctx, adCid)
	if err != nil {
		unsubscribe()
		return nil, fmt.Errorf("get advertisement: %w", err)
	}

	logEntry.WithField("adCID", adCid.String()).WithField("entriesCID", ad.Entries.String()).Infoln("Waiting for indexer to reach out")
	adIndexed := false
loop:
	for {
		select {
		case <-ctx.Done():
			logEntry.Infoln("Context canceled. Unsubscribing from IPNI requests.")
			unsubscribe()
			step := "index_ad"
			if adIndexed {
				step = "index_entries"
			}
			pr.Measurements = append(pr.Measurements, &Measurement{
				Step:     step,
				Duration: time.Since(start),
				Error:    fmtErr(ctx.Err()),
			})
			return pr, nil
		case <-time.After(i.conf.IndexerIndexTimeout):
			logEntry.Infoln("Indexer did not reach out/complete the sync with us.")
			unsubscribe()

			step := "index_ad"
			if adIndexed {
				step = "index_entries"
			}

			pr.Measurements = append(pr.Measurements, &Measurement{
				Step:     step,
				Duration: time.Since(start),
				Error:    fmt.Sprintf("timeout waiting %s for the indexer to reach out", i.conf.IndexerIndexTimeout),
			})
			return pr, nil
		case ipniReq := <-ipniReqChan:
			if ipniReq.cid == adCid.String() {
				adIndexed = true
				adIndexDuration := ipniReq.requestTime.Sub(start)
				pr.Measurements = append(pr.Measurements, &Measurement{
					Step:     "index_ad",
					Duration: adIndexDuration,
				})
				logEntry.WithField("duration", adIndexDuration.String()).Infoln("Indexer reached out to us to fetch advertisement!")
			} else if ipniReq.cid == ad.Entries.String() {
				entriesIndexDuration := ipniReq.requestTime.Sub(start)
				pr.Measurements = append(pr.Measurements, &Measurement{
					Step:     "index_entries",
					Duration: entriesIndexDuration,
				})
				logEntry.WithField("duration", entriesIndexDuration.String()).Infoln("Indexer reached out to us to fetch entries!")
				unsubscribe()
				break loop
			}
		}
	}

	ctx, cancel := context.WithTimeout(ctx, i.conf.IndexerAvailabilityTimeout)
	defer cancel()

	t := time.NewTimer(0)

	probeIdx := 0
	for {
		logEntry.Infof("Pausing for %s\n", i.conf.IndexerInterval)
		select {
		case <-ctx.Done():
			pr.Measurements = append(pr.Measurements, &Measurement{
				Step:     "availability",
				Duration: time.Since(start),
				Error:    fmtErr(ctx.Err()),
			})
			log.Infoln("Context canceled while probing CID availability.")
			return pr, nil
		case <-t.C:
			t.Reset(i.conf.IndexerInterval)
		}

		logEntry.WithField("probeIdx", probeIdx).Infoln("Getting probe Multihash")

		ipniResp, err := i.client.Find(ctx, probes[probeIdx])
		// update probe index for new round
		probeIdx = (probeIdx + 1) % probeCount

		if aErr, ok := err.(*apierror.Error); ok && aErr.Status() == http.StatusNotFound {
			logEntry.Infoln("Probe Multihash not found")
			continue
		} else if err != nil {
			pr.Measurements = append(pr.Measurements, &Measurement{
				Step:     "availability",
				Duration: time.Since(start),
				Error:    fmt.Sprintf("get probe multihash %s: %s", i.conf.IndexerHost, err),
			})
			log.WithError(err).Warnln("Failed to get probe multihash")
			return pr, nil

		} else {
			for _, mhRes := range ipniResp.MultihashResults {
				for _, pRes := range mhRes.ProviderResults {
					if pRes.Provider.ID == i.host.ID() {
						pr.Measurements = append(pr.Measurements, &Measurement{
							Step:     "availability",
							Duration: time.Since(start),
						})
						logEntry.Infoln("Provider Found!")
						return pr, nil
					}
				}
			}
			logEntry.Infoln("Provider not in response")
			continue
		}
	}
}

func (i *IPNIServer) Retrieve(ctx context.Context, cid cid.Cid) (*RetrievalResponse, error) {
	start := time.Now()
	ipniResp, err := i.client.Find(ctx, cid.Hash())

	var duration time.Duration
	if err == nil && len(ipniResp.MultihashResults) == 0 {
		err = fmt.Errorf("not found")
	}

	retResp := &RetrievalResponse{
		CID: cid.String(),
		Measurements: []*Measurement{
			{
				Step:     "retrieval",
				Duration: time.Since(start),
				Error:    fmtErr(err),
			},
		},
		RoutingTableSize: 0,
	}

	latencies.WithLabelValues("retrieval_ttfpr", strconv.FormatBool(err == nil), ctx.Value(headerSchedulerID).(string)).Observe(duration.Seconds())
	return retResp, nil
}

func (i *IPNIServer) Readiness() bool {
	return true
}

type ipniRequest struct {
	cid         string
	requestTime time.Time
}

func (i *IPNIServer) httpHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		ts := time.Now()
		ask := path.Base(req.URL.Path)
		logEntry := log.WithField("ask", ask)

		reqType := req.Header.Get(ipnisync.CidSchemaHeader)
		if reqType != "" {
			logEntry = logEntry.WithField("reqType", reqType)
		}

		logEntry.Debugln("Publisher received request")

		handler(rw, req)

		go func() {
			i.subsMu.RLock()
			defer i.subsMu.RUnlock()
			for subID, sub := range i.subs {
				logEntry.WithField("subID", subID).Debugln("Sending request to subscriber")
				sub <- ipniRequest{
					cid:         ask,
					requestTime: ts,
				}
			}
		}()
	}
}

func (i *IPNIServer) multiHashLister(ctx context.Context, p peer.ID, contextID []byte) (provider.MultihashIterator, error) {
	i.multihashesLk.RLock()
	defer i.multihashesLk.RUnlock()

	// Looking up the multihash for the given contextID
	entry, found := i.multihashes[string(contextID)]
	if !found {
		return nil, fmt.Errorf("unknown context ID")
	}

	return provider.SliceMultihashIterator(entry.mhs), nil
}
