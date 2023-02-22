package parsec

import (
	"fmt"
	"net/http"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/multiformats/go-multiaddr"

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

	log.Infoln("Bootstrapping DHT...")

	bps := []peer.AddrInfo{}
	for _, s := range []string{
		//"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		//"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		//"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		//"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ", // mars.i.ipfs.io
	} {
		ma, err := multiaddr.NewMultiaddr(s)
		if err != nil {
			panic(err)
		}
		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			panic(err)
		}
		bps = append(bps, *pi)
	}

	for _, bp := range bps { // kaddht.GetDefaultBootstrapPeerAddrInfos() {
		log.WithField("peerID", util.FmtPeerID(bp.ID)).Infoln("Connecting to bootstrap peer...")
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
			"url":    r.URL.String(),
			"method": r.Method,
		}).Infoln("Received Request")

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
