package parsec

import (
	"fmt"
	"net/http"

	"golang.org/x/sync/errgroup"

	"github.com/dennis-tra/parsec/pkg/dht"
	"github.com/julienschmidt/httprouter"
)

import (
	"context"
	"errors"
	"net"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	ctx    context.Context
	server *http.Server
	done   chan struct{}
	cancel context.CancelFunc
	addr   string
	host   *dht.Host
}

func NewServer(h string, p int) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	parsecHost, err := dht.New(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("new host: %w", err)
	}

	for _, bp := range kaddht.GetDefaultBootstrapPeerAddrInfos() {
		if err = parsecHost.Connect(ctx, bp); err != nil {
			log.WithError(err).Warnln("Could not connect to bootstrap peer")
		}
	}

	s := &Server{
		ctx:    ctx,
		cancel: cancel,
		addr:   fmt.Sprintf("%s:%d", h, p),
		host:   parsecHost,
		done:   make(chan struct{}),
	}

	return s, nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	errg := errgroup.Group{}

	errg.Go(func() error {
		return s.server.Shutdown(ctx)
	})
	errg.Go(func() error {
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

func (s *Server) ListenAndServe() error {
	tcpListener, err := net.Listen("tcp", s.ListenAddr())
	if err != nil {
		return fmt.Errorf("listen tcp: %w", err)
	}

	router := httprouter.New()
	router.GET("/info", s.info)
	router.POST("/provide", s.provide)
	router.POST("/retrieve/:cid", s.retrieve)

	s.server = &http.Server{
		Handler: s.logHandler(router),
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
			"url":    r.URL,
			"method": r.Method,
		}).Infoln("Received Request")

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
