package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/probe-lab/parsec/pkg/db"
	"github.com/probe-lab/parsec/pkg/models"
)

type IServer interface {
	Provide(ctx context.Context, cid cid.Cid) (*ProvideResponse, error)
	Retrieve(ctx context.Context, cid cid.Cid) (*RetrievalResponse, error)
	Readiness() bool
	Shutdown(ctx context.Context) error
}

const headerSchedulerID = "x-scheduler-id"

type Config struct {
	Host         string
	Port         int
	StartupDelay time.Duration
}

func (s *Config) ListenAddr() string {
	return net.JoinHostPort(s.Host, fmt.Sprintf("%d", s.Port))
}

type Server struct {
	conf       *Config
	dbc        db.Client
	server     *http.Server
	serverImpl IServer
	done       chan struct{}
	dbNode     *models.Node
}

func NewServer(dbc db.Client, dbNode *models.Node, serverImpl IServer, conf *Config) (*Server, error) {
	s := &Server{
		dbc:        dbc,
		conf:       conf,
		dbNode:     dbNode,
		serverImpl: serverImpl,
		done:       make(chan struct{}),
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
		if s.server == nil {
			return nil
		}
		log.Infoln("Stopping server...")
		return s.server.Shutdown(ctx)
	})
	errg.Go(func() error {
		log.Infoln("Stopping p2p host...")
		return s.serverImpl.Shutdown(ctx)
	})

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

func (s *Server) ListenAndServe(ctx context.Context) error {
	tcpListener, err := net.Listen("tcp", s.conf.ListenAddr())
	if err != nil {
		return fmt.Errorf("listen tcp: %w", err)
	}

	go func() {
		// Start by waiting until the node is ready.
		log.WithField("sleep", s.conf.StartupDelay).Infoln("Waiting for node to be ready...")
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.conf.StartupDelay):
		}

		if err := s.dbc.UpdateHeartbeat(ctx, s.dbNode); err != nil {
			log.WithError(err).Warnln("Couldn't update heartbeat")
		}

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

	mux := http.NewServeMux()
	mux.Handle("POST /provide", http.HandlerFunc(s.provide))
	mux.Handle("POST /retrieve/{cid}", http.HandlerFunc(s.retrieve))
	mux.Handle("GET /readiness", http.HandlerFunc(s.readiness))

	s.server = &http.Server{
		Handler:     s.metricsHandler(s.logHandler(mux)),
		BaseContext: func(listener net.Listener) context.Context { return ctx },
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
		parts := strings.Split(r.URL.Path, "/")

		path := "-"
		if len(parts) > 1 {
			path = parts[1]
		}

		totalRequests.WithLabelValues(r.Method, path, r.Header.Get(headerSchedulerID)).Inc()

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func fmtErr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
