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

	downstreamConnections *connections
	callbacks             ConnectionCallbacks

	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc

	connectionsWg sync.WaitGroup
}

var (
	handlePath = "/"
)

func newServer(cfg *OpAMPServer, logger *zap.Logger, callbacks ConnectionCallbacks) *server {
	ctx, cancel := context.WithCancel(context.Background())
	return &server{
		cfg:                   cfg,
		logger:                logger,
		wsUpgrader:            websocket.Upgrader{},
		downstreamConnections: newConnections(),
		callbacks:             callbacks,
		shutdownCtx:           ctx,
		shutdownCancel:        cancel,
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
			// TODO: add TLS support
			return s.httpServer.Serve(l)
		},
	)

	return err
}

// Stop stops the HTTP Server.
func (s *server) Stop() error {
	// cancel the shutdown context to stop the existing connections
	s.shutdownCancel()

	// shutdown the http server
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
			if s.httpServerServeWg != nil {
				s.httpServerServeWg.Wait()
			}
		}
	}

	// wait in a separate goroutine to prevent blocking the return and bypassing the timeout
	done := make(chan struct{})
	go func() {
		s.connectionsWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// --------------------------------------------------------------------------------------
// downstream connection management

func (s *server) addDownstreamConnection(agentID string, conn *connection) {
	// set the id to the agent ID so we can remove the connection by id later
	conn.id = agentID
	s.downstreamConnections.set(agentID, conn)
}

func (s *server) getDownstreamConnection(agentID string) (*connection, bool) {
	conn, ok := s.downstreamConnections.get(agentID)
	return conn, ok
}

func (s *server) removeDownstreamConnection(conn *connection) {
	s.downstreamConnections.remove(conn.id)
}

// --------------------------------------------------------------------------------------

// startHttpServer starts the HTTP Server in background.
func (s *server) startHttpServer(addr string, serveFunc func(l net.Listener) error) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.addr = ln.Addr()

	// Run the HTTP Server in background.
	go func() {
		err = serveFunc(ln)

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

	// the initial id is the remote address of the connection but once we have parsed the
	// agent ID, we will use that as the id
	id := fmt.Sprintf("downstream-%s", conn.RemoteAddr().String())
	c := newConnection(conn, id, s.logger.Named("downstream-connection"))

	// start the connection in a goroutine to prevent blocking the handler
	s.connectionsWg.Add(1)
	go func() {
		defer s.connectionsWg.Done()
		c.start(s.shutdownCtx, s.callbacks)
	}()
}

// acceptOpAMPConnection returns true if the connection should be accepted
func (s *server) acceptOpAMPConnection(req *http.Request) bool {
	s.logger.Info("connection request", zap.String("user-agent", req.UserAgent()))
	// TODO: add authentication
	return true
}
