package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/ipfs/go-cid"
	log "github.com/sirupsen/logrus"

	"github.com/probe-lab/parsec/pkg/util"
)

type ProvideRequest struct {
	Content []byte
}

func (s *Server) provide(rw http.ResponseWriter, req *http.Request) {
	var pr ProvideRequest
	data, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := req.Body.Close(); err != nil {
			log.WithError(err).Warnln("Failed closing request body")
		}
	}()

	if err = json.Unmarshal(data, &pr); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	content, err := util.ContentFrom(pr.Content)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	log.WithField("cid", content.CID.String()).Infoln("Start providing content...")

	ctx := context.WithValue(req.Context(), headerSchedulerID, req.Header.Get(headerSchedulerID))
	resp, err := s.serverImpl.Provide(ctx, content.CID)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err = json.Marshal(resp)
	if err != nil {
		rw.Write([]byte(err.Error()))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = rw.Write(data); err != nil {
		rw.Write([]byte(err.Error()))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type RetrieveRequest struct{}

func (s *Server) retrieve(rw http.ResponseWriter, req *http.Request) {
	ctx := context.WithValue(req.Context(), headerSchedulerID, req.Header.Get(headerSchedulerID))
	var rr RetrieveRequest
	data, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := req.Body.Close(); err != nil {
			log.WithError(err).Warnln("Failed closing request body")
		}
	}()

	if err = json.Unmarshal(data, &rr); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	c, err := cid.Decode(req.PathValue("cid"))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := s.serverImpl.Retrieve(ctx, c)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	//resp := RetrievalResponse{
	//	CID:              c.String(),
	//	RoutingTableSize: RoutingTableSize(s.host.DHT),
	//}
	//logEntry := log.WithField("cid", c.String()).WithField("rtSize", resp.RoutingTableSize)
	//
	//logEntry.Infoln("Start finding providers")
	//
	//// here's where the magic happens
	//switch rr.Routing {
	//case config.RoutingIPNI:
	//	start := time.Now()
	//	pr, err := s.host.IndexerLookup(ctx, c)
	//	resp.Duration = time.Since(start)
	//
	//	logEntry = logEntry.WithField("dur", resp.Duration.Seconds())
	//	if err != nil {
	//		logEntry.Warnln("Failed looking up provider")
	//		resp.Error = err.Error()
	//	} else {
	//		if len(pr.MultihashResults) == 0 {
	//			resp.Error = "not found"
	//		}
	//	}
	//	latencies.WithLabelValues("retrieval_ttfpr", string(config.RoutingIPNI), strconv.FormatBool(resp.Error == ""), r.Header.Get(headerSchedulerID)).Observe(resp.Duration.Seconds())
	//case config.RoutingDHT:
	//	start := time.Now()
	//	provider := <-s.host.DHT.FindProvidersAsync(ctx, c, 1)
	//	resp.Duration = time.Since(start)
	//
	//	logEntry = logEntry.WithField("dur", resp.Duration.Seconds())
	//
	//	if errors.Is(provider.ID.Validate(), peer.ErrEmptyPeerID) {
	//		resp.Error = "not found"
	//		logEntry.Infoln("Didn't find provider")
	//	} else {
	//		s.host.Network().ClosePeer(provider.ID)
	//		s.host.Peerstore().RemovePeer(provider.ID)
	//		s.host.Peerstore().ClearAddrs(provider.ID)
	//		logEntry.WithField("provider", util.FmtPeerID(provider.ID)).Infoln("Found provider")
	//	}
	//	latencies.WithLabelValues("retrieval_ttfpr", string(config.RoutingDHT), strconv.FormatBool(resp.Error == ""), r.Header.Get(headerSchedulerID)).Observe(resp.Duration.Seconds())
	//default:
	//	panic(fmt.Sprintf("unknown routing: %s", rr.Routing))
	//}

	data, err = json.Marshal(resp)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err = rw.Write(data); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) readiness(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
}
