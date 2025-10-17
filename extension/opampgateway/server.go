package opampgateway

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
)

type ServerConnectionManagement interface {
	// EnsureConnection will ensure that a connection exists for the given agent ID.
	// If a connection does not exist, it will be created using the provided websocket connection.
	EnsureDownstreamConnection(agentID string, conn *websocket.Conn)

	// ForwardMessageUpstream will forward the given message to the upstream connection for the given agent ID.
	ForwardMessageUpstream(ctx context.Context, agentID string, msg []byte) error
}

type server struct {
	cfg    *OpAMPServer
	logger *zap.Logger

	httpServer        *http.Server
	httpServerServeWg *sync.WaitGroup

	wsUpgrader websocket.Upgrader

	addr net.Addr

	connectionManagement ServerConnectionManagement
}

var (
	handlePath = "/"
)

func newServer(cfg *OpAMPServer, logger *zap.Logger, connectionManagement ServerConnectionManagement) *server {
	return &server{
		cfg:                  cfg,
		logger:               logger,
		wsUpgrader:           websocket.Upgrader{},
		connectionManagement: connectionManagement,
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

	go s.handleWSConnection(conn)
}

// acceptOpAMPConnection returns true if the connection should be accepted
func (s *server) acceptOpAMPConnection(req *http.Request) bool {
	s.logger.Info("connection request", zap.String("user-agent", req.UserAgent()))
	return true
}

func (s *server) handleWSConnection(conn *websocket.Conn) {
	connectionCreated := false
	for {
		msgContext := context.Background()
		request := protobufs.AgentToServer{}

		// Block on reading a message from the WebSocket connection.
		mt, msgBytes, err := conn.ReadMessage()
		isBreak, err := func() (bool, error) {
			if err != nil {
				if !websocket.IsUnexpectedCloseError(err) {
					s.logger.Error("Cannot read a message from WebSocket", zap.Error(err))
					return true, err
				}
				// This is a normal closing of the WebSocket connection.
				s.logger.Debug("Agent disconnected", zap.Error(err))
				return true, err
			}
			if mt != websocket.BinaryMessage {
				err = fmt.Errorf("unexpected message type: %v, must be binary message", mt)
				s.logger.Error("Cannot process a message from WebSocket", zap.Error(err))
				return false, err
			}

			// Decode WebSocket message as a Protobuf message.
			err = decodeWSMessage(msgBytes, &request)
			if err != nil {
				s.logger.Error("Cannot decode message from WebSocket", zap.Error(err))
				return false, err
			}
			return false, nil
		}()
		if err != nil {
			s.logger.Error("Error processing message from WebSocket", zap.Error(err))
			if isBreak {
				break
			}
			continue
		}
		s.logger.Info("Received message from WebSocket",
			zap.String("message", string(msgBytes)),
		)

		// Parse out AgentID
		agentID, err := parseAgentID(request.GetInstanceUid())
		if err != nil {
			s.logger.Error("Error parsing agent ID", zap.Error(err))
			continue
		}

		// Ensure a downstream connection exists for the agent ID.
		if !connectionCreated {
			s.logger.Info("Creating downstream connection for agent ID", zap.String("agent_id", agentID))
			s.connectionManagement.EnsureDownstreamConnection(agentID, conn)
			connectionCreated = true
		}

		// Forward the message to the upstream connection for the agent ID.
		err = s.connectionManagement.ForwardMessageUpstream(msgContext, agentID, msgBytes)
		if err != nil {
			s.logger.Error("Error forwarding message to upstream", zap.Error(err))
			continue
		}
	}
}
