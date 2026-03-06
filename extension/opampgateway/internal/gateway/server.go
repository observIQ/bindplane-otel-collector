// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gateway

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/observiq/bindplane-otel-collector/extension/opampgateway/internal/metadata"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.uber.org/zap"
)

// authResponse holds the result of an authentication request
type authResponse struct {
	result OpampGatewayConnectResult
	err    error
}

type server struct {
	serverConfig confighttp.ServerConfig
	logger       *zap.Logger

	httpServer        *http.Server
	httpServerServeWg *sync.WaitGroup

	wsUpgrader websocket.Upgrader

	addr net.Addr

	// agentConnections represent connections per agent ID
	agentConnections *connections[*downstreamConnection]

	// downstreamConnections represent connections per downstream connection ID
	downstreamConnections *connections[*downstreamConnection]

	callbacks                  ConnectionCallbacks[*downstreamConnection]
	upstreamConnectionAssigner UpstreamConnectionAssigner

	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc

	connectionsWg sync.WaitGroup

	downstreamConnectionCount atomic.Uint32
	authTimeout               time.Duration

	telemetry *metadata.TelemetryBuilder

	// pendingAuthRequests tracks pending authentication requests awaiting responses
	pendingAuthRequests *authRequests
}

var (
	handlePath = "/"
)

func newServer(serverConfig confighttp.ServerConfig, authTimeout time.Duration, telemetry *metadata.TelemetryBuilder, upstreamConnectionAssigner UpstreamConnectionAssigner, callbacks ConnectionCallbacks[*downstreamConnection], logger *zap.Logger) *server {
	if authTimeout == 0 {
		authTimeout = 30 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &server{
		serverConfig:               serverConfig,
		authTimeout:                authTimeout,
		logger:                     logger.Named("server"),
		wsUpgrader:                 websocket.Upgrader{},
		agentConnections:           newConnections[*downstreamConnection](),
		downstreamConnections:      newConnections[*downstreamConnection](),
		upstreamConnectionAssigner: upstreamConnectionAssigner,
		callbacks:                  callbacks,
		shutdownCtx:                ctx,
		shutdownCancel:             cancel,
		telemetry:                  telemetry,
		pendingAuthRequests:        newAuthRequests(),
	}
}

// Start starts the HTTP Server. It resets internal state so the server can be
// restarted after a previous Stop (e.g. during collector hot-reload).
func (s *server) Start(ctx context.Context, host component.Host, telemetrySettings component.TelemetrySettings) error {
	// Reset state so the server can be restarted after a previous Stop.
	s.shutdownCtx, s.shutdownCancel = context.WithCancel(context.Background())
	s.agentConnections = newConnections[*downstreamConnection]()
	s.downstreamConnections = newConnections[*downstreamConnection]()
	s.pendingAuthRequests = newAuthRequests()

	mux := http.NewServeMux()
	mux.HandleFunc(handlePath, s.handleRequest)

	hs, err := s.serverConfig.ToServer(ctx, host.GetExtensions(), telemetrySettings, mux)
	if err != nil {
		return fmt.Errorf("create http server: %w", err)
	}
	s.httpServer = hs

	ln, err := s.serverConfig.ToListener(ctx)
	if err != nil {
		return fmt.Errorf("create listener: %w", err)
	}
	s.addr = ln.Addr()
	s.logger.Info("server listening", zap.String("endpoint", s.addr.String()))

	s.httpServerServeWg = &sync.WaitGroup{}
	s.httpServerServeWg.Add(1)

	// Run the HTTP Server in background.
	go func() {
		defer s.httpServerServeWg.Done()
		if serveErr := s.httpServer.Serve(ln); serveErr != nil && serveErr != http.ErrServerClosed {
			s.logger.Error("Error running HTTP Server", zap.Error(serveErr))
		}
	}()

	return nil
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
// agent connection management

func (s *server) getAgentConnection(agentID string) (*downstreamConnection, bool) {
	conn, ok := s.agentConnections.get(agentID)
	return conn, ok
}

func (s *server) setAgentConnection(agentID string, conn *downstreamConnection) {
	s.agentConnections.set(agentID, conn)
}

// --------------------------------------------------------------------------------------
// downstream connection management

func (s *server) addDownstreamConnection(connectionID string, conn *downstreamConnection) {
	s.downstreamConnections.set(connectionID, conn)
}

func (s *server) getDownstreamConnection(connectionID string) (*downstreamConnection, bool) {
	conn, ok := s.downstreamConnections.get(connectionID)
	return conn, ok
}

func (s *server) removeDownstreamConnection(conn *downstreamConnection) {
	s.downstreamConnections.remove(conn.id)
	s.agentConnections.removeByValue(conn)
}

func (s *server) closeDownstreamConnections(downstreamConnectionIDs []string) {
	for _, downstreamConnectionID := range downstreamConnectionIDs {
		if conn, ok := s.downstreamConnections.get(downstreamConnectionID); ok {
			s.logger.Info("closing downstream connection", zap.String("downstream_connection_id", downstreamConnectionID))
			err := conn.close()
			if err != nil {
				s.logger.Error("failed to close downstream connection", zap.Error(err), zap.String("downstream_connection_id", downstreamConnectionID))
			}
			s.logger.Info("closed downstream connection", zap.String("downstream_connection_id", downstreamConnectionID))
		}
	}
}

// --------------------------------------------------------------------------------------

// handleRequest handles accepting OpAMP connections and upgrading to a websocket connection
func (s *server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// the initial id is the remote address of the connection but once we have parsed the
	// agent ID, we will use that as the id
	id := fmt.Sprintf("downstream-%d-%s", s.downstreamConnectionCount.Add(1), r.RemoteAddr)
	upstreamConnection, err := s.upstreamConnectionAssigner.AssignUpstreamConnection(id)
	if err != nil {
		if errors.Is(err, ErrNoUpstreamConnectionsAvailable) {
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		s.logger.Error("assign upstream connection", zap.Error(err))
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.logger.Info("assigned upstream connection", zap.String("downstream_connection_id", id), zap.String("upstream_connection_id", upstreamConnection.id))

	// Authenticate the connection via the upstream OpAMP server
	ctx, cancel := context.WithTimeout(r.Context(), s.authTimeout)
	defer cancel()

	accepted, result := s.acceptOpAMPConnection(ctx, r, upstreamConnection, id)
	if !accepted {
		s.upstreamConnectionAssigner.UnassignUpstreamConnection(id)
		// Set response headers from the result
		for key, value := range result.ResponseHeaders {
			w.Header().Set(key, value)
		}
		// Use the HTTP status code from the result, default to 401 if not set
		statusCode := result.HTTPStatusCode
		if statusCode == 0 {
			statusCode = http.StatusUnauthorized
		}
		w.WriteHeader(statusCode)
		return
	}

	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.upstreamConnectionAssigner.UnassignUpstreamConnection(id)
		s.logger.Error("accept OpAMP connection", zap.Error(err))
		return
	}
	s.logger.Debug("accepted OpAMP connection", zap.String("remote_addr", conn.RemoteAddr().String()))

	s.telemetry.OpampgatewayConnections.Add(context.Background(), 1, directionDownstream)

	// create the downstream connection
	c := newDownstreamConnection(conn, s.telemetry, upstreamConnection, id, s.logger)
	s.addDownstreamConnection(id, c)

	// start the connection in a goroutine to prevent blocking the handler
	s.connectionsWg.Add(1)
	go func() {
		defer s.connectionsWg.Done()
		defer s.telemetry.OpampgatewayConnections.Add(context.Background(), -1, directionDownstream)

		c.start(s.shutdownCtx, s.callbacks)
	}()
}

// acceptOpAMPConnection authenticates the connection by sending an OpampGatewayConnect
// message to the upstream server and waiting for an OpampGatewayConnectResult response.
// Returns true if the connection is accepted, along with the result containing status details.
func (s *server) acceptOpAMPConnection(ctx context.Context, req *http.Request, upstreamConn *upstreamConnection, connectionID string) (bool, OpampGatewayConnectResult) {
	s.logger.Info("connection request", zap.String("user-agent", req.UserAgent()), zap.String("remote_addr", req.RemoteAddr), zap.String("downstream_connection_id", connectionID), zap.String("upstream_connection_id", upstreamConn.id))

	// Create a unique ID for this authentication request
	requestUID := uuid.New().String()

	// Create a channel to receive the authentication response
	responseChan := make(chan authResponse, 1)

	// Register the pending auth request
	s.pendingAuthRequests.addRequest(requestUID, responseChan)

	// Ensure cleanup
	defer s.pendingAuthRequests.removeRequest(requestUID)

	// Create the OpampGatewayConnect message with RequestUid for correlation
	connectMsg := OpampGatewayConnect{
		RequestUID:    requestUID,
		RemoteAddress: req.RemoteAddr,
		Headers:       req.Header,
	}

	// Marshal the connect message
	connectData, err := jsoniter.Marshal(connectMsg)
	if err != nil {
		s.logger.Error("failed to marshal OpampGatewayConnect", zap.Error(err))
		return false, OpampGatewayConnectResult{
			Accept:         false,
			HTTPStatusCode: http.StatusInternalServerError,
		}
	}

	// Create the AgentToServer message with the custom message
	agentToServer := &protobufs.AgentToServer{
		CustomMessage: &protobufs.CustomMessage{
			Capability: OpampGatewayCapability,
			Type:       OpampGatewayConnectType,
			Data:       connectData,
		},
	}

	// Encode and send the message
	msgData, err := encodeWSMessage(agentToServer)
	if err != nil {
		s.logger.Error("failed to encode auth message", zap.Error(err))
		return false, OpampGatewayConnectResult{
			Accept:         false,
			HTTPStatusCode: http.StatusInternalServerError,
		}
	}

	// Send the authentication request to upstream
	msg := newMessage(0, msgData)
	if err := upstreamConn.send(msg); err != nil {
		s.logger.Error("failed to send OpampGatewayConnect", zap.Error(err))
		return false, OpampGatewayConnectResult{
			Accept:         false,
			HTTPStatusCode: http.StatusInternalServerError,
		}
	}
	s.logger.Debug("sent OpampGatewayConnect", zap.String("request_uid", requestUID))

	// Wait for the response
	select {
	case <-ctx.Done():
		s.logger.Warn("authentication timed out", zap.String("request_uid", requestUID))
		return false, OpampGatewayConnectResult{
			Accept:         false,
			HTTPStatusCode: http.StatusGatewayTimeout,
			ResponseHeaders: map[string]string{
				"Retry-After": "30",
			},
		}
	case resp := <-responseChan:
		if resp.err != nil {
			s.logger.Error("authentication error", zap.Error(resp.err), zap.String("request_uid", requestUID))
			return false, OpampGatewayConnectResult{
				Accept:         false,
				HTTPStatusCode: http.StatusInternalServerError,
			}
		}
		s.logger.Info("authentication result",
			zap.Bool("accepted", resp.result.Accept),
			zap.Int("status_code", resp.result.HTTPStatusCode),
			zap.String("request_uid", requestUID))
		return resp.result.Accept, resp.result
	}
}

// handleAuthResponse processes an authentication response from upstream and routes it
// to the waiting goroutine. Returns true if the message was an auth response.
func (s *server) handleAuthResponse(customMsg *protobufs.CustomMessage) bool {
	if customMsg == nil {
		return false
	}
	if customMsg.Capability != OpampGatewayCapability || customMsg.Type != OpampGatewayConnectResultType {
		return false
	}

	// Parse the result to get the RequestUid
	var result OpampGatewayConnectResult
	if err := jsoniter.Unmarshal(customMsg.Data, &result); err != nil {
		s.logger.Error("failed to unmarshal OpampGatewayConnectResult", zap.Error(err))
		return false
	}

	if result.RequestUID == "" {
		s.logger.Debug("OpampGatewayConnectResult missing RequestUid")
		return false
	}

	// Look up the pending auth request using RequestUid
	responseChan, ok := s.pendingAuthRequests.getRequest(result.RequestUID)
	if !ok {
		s.logger.Debug("no pending auth request for RequestUid", zap.String("request_uid", result.RequestUID))
		return false
	}

	// Send the response
	responseChan <- authResponse{result: result}
	return true
}

type authRequests struct {
	mtx      sync.Mutex
	requests map[string]chan<- authResponse
}

func newAuthRequests() *authRequests {
	return &authRequests{
		requests: make(map[string]chan<- authResponse),
	}
}

func (a *authRequests) addRequest(requestUID string, responseChan chan<- authResponse) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.requests[requestUID] = responseChan
}

func (a *authRequests) getRequest(requestUID string) (chan<- authResponse, bool) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	responseChan, ok := a.requests[requestUID]
	return responseChan, ok
}

func (a *authRequests) removeRequest(requestUID string) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	delete(a.requests, requestUID)
}
