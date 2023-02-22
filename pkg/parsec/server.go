package parsec

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ipfs/go-cid"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/routing"

	"github.com/julienschmidt/httprouter"
)

import (
	"context"
	"errors"
	"net"

	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	ctx    context.Context
	server *http.Server
	done   chan struct{}
	cancel context.CancelFunc
	dht    *kaddht.IpfsDHT
	addr   string
	host   host.Host
}

func NewServer(h string, p int) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	var dht *kaddht.IpfsDHT
	libp2pHost, err := libp2p.New(
		libp2p.Routing(func(libp2pHost host.Host) (routing.PeerRouting, error) {
			var err error
			dht, err = kaddht.New(ctx, libp2pHost)
			return dht, err
		}))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("new libp2p host: %w", err)
	}

	for _, bp := range kaddht.GetDefaultBootstrapPeerAddrInfos() {
		if err = libp2pHost.Connect(ctx, bp); err != nil {
			log.WithError(err).Warnln("Could not connect to bootstrap peer")
		}
	}

	s := &Server{
		ctx:    ctx,
		cancel: cancel,
		addr:   fmt.Sprintf("%s:%d", h, p),
		host:   libp2pHost,
		dht:    dht,
		done:   make(chan struct{}),
	}

	return s, nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	err := s.server.Shutdown(ctx)
	s.cancel()
	select {
	case <-s.done:
		return err
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

func (s *Server) info(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
}

type ProvideRequest struct {
	CID []byte
}

func (s *Server) provide(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var pr ProvideRequest
	data, err := io.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = json.Unmarshal(data, &pr); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	_, c, err := cid.CidFromBytes(pr.CID)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.dht.Provide(r.Context(), c, true)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type RetrieveRequest struct {
	Count int
}

type RetrieveResponse struct{}

func (s *Server) retrieve(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var rr RetrieveRequest
	data, err := io.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = json.Unmarshal(data, &rr); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	c, err := cid.Decode(params.ByName("cid"))
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	<-s.dht.FindProvidersAsync(r.Context(), c, rr.Count)
}
