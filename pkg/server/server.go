package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dennis-tra/parsec/pkg/models"

	"github.com/dennis-tra/parsec/pkg/db"

	"github.com/dennis-tra/parsec/pkg/config"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"

	"github.com/dennis-tra/parsec/pkg/util"

	"github.com/dennis-tra/parsec/pkg/dht"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/sync/errgroup"
)

import (
	"context"
	"errors"
	"net"

	log "github.com/sirupsen/logrus"
)

type Server struct {
	server *http.Server
	done   chan struct{}
	cancel context.CancelFunc
	addr   string
	host   *dht.Host
	dbc    db.Client
	dbNode *models.Node
}

func NewServer(ctx context.Context, dbc db.Client, conf config.ServerConfig) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)

	parsecHost, err := dht.New(ctx, conf.PeerPort, conf.FullRT, conf.DHTServer)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("new host: %w", err)
	}

	log.Infoln("Bootstrapping DHT...")
	for _, bp := range kaddht.GetDefaultBootstrapPeerAddrInfos() {
		log.WithField("peerID", util.FmtPeerID(bp.ID)).Infoln("Connecting to bootstrap peer...")
		if err = parsecHost.Connect(ctx, bp); err != nil {
			log.WithError(err).Warnln("Could not connect to bootstrap peer")
		}
	}

	dbNode, err := dbc.InsertNode(ctx, parsecHost.ID(), conf)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("insert node: %w", err)
	}

	s := &Server{
		cancel: cancel,
		dbc:    dbc,
		addr:   fmt.Sprintf("%s:%d", conf.ServerHost, conf.ServerPort),
		host:   parsecHost,
		dbNode: dbNode,
		done:   make(chan struct{}),
	}

	return s, nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	defer func() {
		log.Infoln("Updating server offline timestamp...")
		if err := s.dbc.UpdateOfflineSince(ctx, s.dbNode); err != nil {
			log.WithError(err).WithField("nodeID", s.dbNode.ID).Warnln("Couldn't update offline since field of node")
		}
	}()

	errg := errgroup.Group{}

	errg.Go(func() error {
		log.Infoln("Stopping server...")
		return s.server.Shutdown(ctx)
	})
	errg.Go(func() error {
		log.Infoln("Stopping p2p host...")
		return s.host.Close()
	})
	s.cancel()

	if err := errg.Wait(); err != nil {
		return fmt.Errorf("shutting down: %w", err)
	}

	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) ListenAddr() string {
	return s.addr
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	tcpListener, err := net.Listen("tcp", s.ListenAddr())
	if err != nil {
		return fmt.Errorf("listen tcp: %w", err)
	}

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
			case <-s.done:
				return
			case <-ctx.Done():
				return
			}

			if err := s.dbc.UpdateHeartbeat(ctx, s.dbNode); err != nil {
				log.WithError(err).Warnln("Couldn't update heartbeat")
			}
		}
	}()

	router := httprouter.New()
	router.GET("/info", s.info)
	router.POST("/provide", s.provide)
	router.POST("/retrieve/:cid", s.retrieve)
	router.GET("/readiness", s.readiness)

	s.server = &http.Server{
		Handler: s.metricsHandler(s.logHandler(router)),
	}

	defer func() {
		close(s.done)
	}()

	err = s.server.Serve(tcpListener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func (s *Server) logHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"url":    r.URL.String(),
			"method": r.Method,
		}).Infoln("Received Request")

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func (s *Server) metricsHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		totalRequests.WithLabelValues(r.Method, r.URL.Path).Inc()
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
