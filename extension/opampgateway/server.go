package opampgateway

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type server struct {
	cfg    *OpAMPServer
	logger *zap.Logger

	httpServer        *http.Server
	httpServerServeWg *sync.WaitGroup

	wsUpgrader websocket.Upgrader

	addr net.Addr
}

var (
	handlePath = "/v1/opamp"
)

func newServer(cfg *OpAMPServer, logger *zap.Logger) *server {
	return &server{
		cfg:        cfg,
		logger:     logger,
		wsUpgrader: websocket.Upgrader{},
	}
}

// Start starts the HTTP Server.
func (s *server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(handlePath, s.handleRequest)

	hs := &http.Server{
		Handler:   mux,
		Addr:      s.cfg.Endpoint,
		TLSConfig: s.cfg.TLS,
	}
	s.httpServer = hs

	s.httpServerServeWg = &sync.WaitGroup{}
	s.httpServerServeWg.Add(1)

	// Start the HTTP Server in background.
	err := s.startHttpServer(
		s.httpServer.Addr,
		func(l net.Listener) error {
			defer s.httpServerServeWg.Done()
			return s.httpServer.ServeTLS(l, "", "")
		},
	)

	return err
}

// Stop stops the HTTP Server.
func (s *server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if s.httpServer != nil {
		defer func() { s.httpServer = nil }()
		// This stops accepting new connections. TODO: close existing
		// connections and wait them to be terminated.
		err := s.httpServer.Shutdown(ctx)
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			s.httpServerServeWg.Wait()
		}
	}
	return nil
}

// startHttpServer starts the HTTP Server in background.
func (s *server) startHttpServer(addr string, fn func(l net.Listener) error) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.addr = ln.Addr()

	// Run the HTTP Server in background.
	go func() {
		err = fn(ln)

		// ErrServerClosed is expected after successful Stop(), so we won't log that
		// particular error.
		if err != nil && err != http.ErrServerClosed {
			s.logger.Error("Error running HTTP Server", zap.Error(err))
		}
	}()

	return nil
}

// handleRequest handles accepting OpAMP connections and upgrading to a websocket connection
func (s *server) handleRequest(w http.ResponseWriter, r *http.Request) {
	if !s.acceptOpAMPConnection(r) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("failed to accept OpAMP connection", zap.Error(err))
		return
	}
	s.logger.Debug("accepted OpAMP connection", zap.String("remote_addr", conn.RemoteAddr().String()))

	// TODO: Handle the websocket connection
}

// acceptOpAMPConnection returns true if the connection should be accepted
func (s *server) acceptOpAMPConnection(req *http.Request) bool {
	return true
}
