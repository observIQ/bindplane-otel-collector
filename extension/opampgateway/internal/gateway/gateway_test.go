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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/observiq/bindplane-otel-collector/extension/opampgateway/internal/metadata"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/proto"
)

func TestGatewayMultipleAgentsRoundTrip(t *testing.T) {
	t.Parallel()

	h := newGatewayTestHarness(t, 1)

	agent1 := h.NewAgent(t)
	agent2 := h.NewAgent(t)

	agent1.Send(&protobufs.AgentToServer{SequenceNum: 1})
	agent2.Send(&protobufs.AgentToServer{SequenceNum: 2})

	received := map[string]*protobufs.AgentToServer{}
	for i := 0; i < 2; i++ {
		msg := h.upstream.WaitForAnyMessage(t, 5*time.Second)
		received[msg.AgentID] = msg.Message
	}

	require.Equal(t, uint64(1), received[agent1.ID()].GetSequenceNum())
	require.Equal(t, uint64(2), received[agent2.ID()].GetSequenceNum())

	require.NoError(t, h.upstream.Send(&protobufs.ServerToAgent{
		InstanceUid:  agent1.RawID(),
		Capabilities: 11,
	}))
	require.NoError(t, h.upstream.Send(&protobufs.ServerToAgent{
		InstanceUid:  agent2.RawID(),
		Capabilities: 22,
	}))

	resp1 := agent1.WaitForMessage(t, 5*time.Second)
	require.Equal(t, uint64(11), resp1.GetCapabilities())
	resp2 := agent2.WaitForMessage(t, 5*time.Second)
	require.Equal(t, uint64(22), resp2.GetCapabilities())
}

func TestGatewayHandlesAgentClose(t *testing.T) {
	t.Parallel()

	h := newGatewayTestHarness(t, 1)
	agent := h.NewAgent(t)

	require.NoError(t, agent.Close())

	require.Eventually(t, func() bool {
		_, ok := h.gateway.server.getDownstreamConnection(agent.ID())
		return !ok
	}, 5*time.Second, 50*time.Millisecond, "downstream connection still registered")

	require.Eventually(t, func() bool {
		_, ok := h.gateway.client.upstreamConnections.get(agent.ID())
		return !ok
	}, 5*time.Second, 50*time.Millisecond, "upstream assignment still registered")
}

func TestGatewayHandlesAgentCloseAfterSend(t *testing.T) {
	t.Parallel()

	h := newGatewayTestHarness(t, 1)
	agent := h.NewAgent(t)

	agent.Send(&protobufs.AgentToServer{SequenceNum: 1})
	h.upstream.WaitForAgentMessage(t, agent.ID(), 5*time.Second)

	require.NoError(t, agent.Close())

	require.Eventually(t, func() bool {
		_, ok := h.gateway.server.getDownstreamConnection(agent.ID())
		return !ok
	}, 5*time.Second, 50*time.Millisecond, "downstream connection still registered")

	require.Eventually(t, func() bool {
		_, ok := h.gateway.client.upstreamConnections.get(agent.ID())
		return !ok
	}, 5*time.Second, 50*time.Millisecond, "upstream assignment still registered")

	require.Eventually(t, func() bool {
		_, ok := h.gateway.server.getAgentConnection(agent.ID())
		return !ok
	}, 5*time.Second, 50*time.Millisecond, "agent connection still registered")
}

func TestGatewayUpstreamConnectionAffinity(t *testing.T) {
	t.Parallel()

	// 10 upstream connections
	h := newGatewayTestHarness(t, 10)
	agent1 := h.NewAgent(t)

	// send an initial message to determine the connection id
	agent1.Send()
	msg := h.upstream.WaitForAgentMessage(t, agent1.ID(), 5*time.Second)
	connectionID := msg.ConnectionID

	// we only expect the agent to use a single upstream connection
	for range 10 {
		agent1.Send()
		msg = h.upstream.WaitForAgentMessage(t, agent1.ID(), 5*time.Second)
		require.Equal(t, connectionID, msg.ConnectionID)
	}

	// // close the connection in use by the agent
	// h.CloseUpstreamConnection(t, connectionID)

	// // send a message to the agent
	// agent1.Send()
	// msg = h.upstream.WaitForAgentMessage(t, agent1.ID(), 5*time.Second)
	// newConnectionID := msg.ConnectionID
	// require.NotEqual(t, connectionID, newConnectionID)

	// // the agent should use the new connection
	// for range 10 {
	// 	agent1.Send()
	// 	msg = h.upstream.WaitForAgentMessage(t, agent1.ID(), 5*time.Second)
	// 	require.Equal(t, newConnectionID, msg.ConnectionID)
	// }
}

func TestGatewayRestartAfterShutdown(t *testing.T) {
	t.Parallel()

	upstream := newTestOpAMPServer(t)

	testTel := componenttest.NewTelemetry()
	t.Cleanup(func() {
		require.NoError(t, testTel.Shutdown(context.Background()))
	})
	telemetry, err := metadata.NewTelemetryBuilder(testTel.NewTelemetrySettings())
	require.NoError(t, err)

	settings := Settings{
		UpstreamOpAMPAddress: upstream.URL(),
		SecretKey:            "test-secret",
		UpstreamConnections:  1,
		ServerEndpoint:       "127.0.0.1:0",
	}

	logger := zaptest.NewLogger(t)
	gw := New(logger, settings, telemetry)

	// Ensure the gateway is always shut down, even if the test fails partway through.
	t.Cleanup(func() { _ = gw.Shutdown(context.Background()) })

	// Helper that starts the gateway, waits for upstream readiness, and returns
	// the agent-facing websocket URL.
	startAndWait := func() string {
		t.Helper()
		ctx := context.Background()
		require.NoError(t, gw.Start(ctx))

		upstream.WaitForConnection(t, 5*time.Second)
		require.Eventually(t, func() bool {
			conn, ok := gw.client.upstreamConnections.get("upstream-0")
			return ok && conn.isConnected()
		}, 5*time.Second, 10*time.Millisecond)

		return fmt.Sprintf("ws://%s%s", gw.server.addr.String(), handlePath)
	}

	// --- first lifecycle ---
	agentURL := startAndWait()

	id1 := uuid.New()
	agent1 := newTestAgent(t, agentURL, id1[:])

	agent1.Send(&protobufs.AgentToServer{SequenceNum: 10})
	msg1 := upstream.WaitForAgentMessage(t, agent1.ID(), 5*time.Second)
	require.Equal(t, uint64(10), msg1.Message.GetSequenceNum())

	_ = agent1.Close()
	require.NoError(t, gw.Shutdown(context.Background()))

	// --- second lifecycle (restart) ---
	agentURL = startAndWait()

	id2 := uuid.New()
	agent2 := newTestAgent(t, agentURL, id2[:])

	agent2.Send(&protobufs.AgentToServer{SequenceNum: 20})
	msg2 := upstream.WaitForAgentMessage(t, agent2.ID(), 5*time.Second)
	require.Equal(t, uint64(20), msg2.Message.GetSequenceNum())

	// Also verify round-trip: upstream can send back to the new agent
	require.NoError(t, upstream.Send(&protobufs.ServerToAgent{
		InstanceUid:  agent2.RawID(),
		Capabilities: 42,
	}))
	resp := agent2.WaitForMessage(t, 5*time.Second)
	require.Equal(t, uint64(42), resp.GetCapabilities())

	_ = agent2.Close()
	require.NoError(t, gw.Shutdown(context.Background()))
}

func TestGatewayShutdownWithoutStart(t *testing.T) {
	t.Parallel()

	testTel := componenttest.NewTelemetry()
	t.Cleanup(func() {
		require.NoError(t, testTel.Shutdown(context.Background()))
	})
	telemetry, err := metadata.NewTelemetryBuilder(testTel.NewTelemetrySettings())
	require.NoError(t, err)

	upstream := newTestOpAMPServer(t)
	settings := Settings{
		UpstreamOpAMPAddress: upstream.URL(),
		SecretKey:            "test-secret",
		UpstreamConnections:  1,
		ServerEndpoint:       "127.0.0.1:0",
	}

	logger := zaptest.NewLogger(t)
	gw := New(logger, settings, telemetry)

	// Shutdown without Start should not panic or error.
	require.NoError(t, gw.Shutdown(context.Background()))
}

func TestGatewayDoubleShutdown(t *testing.T) {
	t.Parallel()

	h := newGatewayTestHarness(t, 1)

	// First shutdown is explicit, second comes from t.Cleanup in the harness.
	require.NoError(t, h.gateway.Shutdown(context.Background()))
	require.NoError(t, h.gateway.Shutdown(context.Background()))
}

// --------------------------------------------------------------------------------------
// test harness

type gatewayTestHarness struct {
	t        *testing.T
	ctx      context.Context
	cancel   context.CancelFunc
	gateway  *Gateway
	upstream *testOpAMPServer
	agentURL string
	agents   sync.Map
}

func newGatewayTestHarness(t *testing.T, upstreamConnections int) *gatewayTestHarness {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	upstream := newTestOpAMPServer(t)

	testTel := componenttest.NewTelemetry()
	t.Cleanup(func() {
		require.NoError(t, testTel.Shutdown(context.Background()))
	})

	telemetry, err := metadata.NewTelemetryBuilder(testTel.NewTelemetrySettings())
	require.NoError(t, err)

	settings := Settings{
		UpstreamOpAMPAddress: upstream.URL(),
		SecretKey:            "test-secret",
		UpstreamConnections:  upstreamConnections,
		ServerEndpoint:       "127.0.0.1:0",
	}

	logger := zaptest.NewLogger(t)

	gw := New(logger, settings, telemetry)

	require.NoError(t, gw.Start(ctx))
	t.Cleanup(func() {
		require.NoError(t, gw.Shutdown(context.Background()))
	})

	h := &gatewayTestHarness{
		t:        t,
		ctx:      ctx,
		cancel:   cancel,
		gateway:  gw,
		upstream: upstream,
		agentURL: fmt.Sprintf("ws://%s%s", gw.server.addr.String(), handlePath),
	}

	for i := 0; i < upstreamConnections; i++ {
		upstream.WaitForConnection(t, 5*time.Second)
	}

	// Wait for all upstream connections to be marked as connected in the pool.
	// WaitForConnection only confirms the websocket was established at the server side;
	// the gateway's writerLoop may not have called setConnected(true) yet.
	require.Eventually(t, func() bool {
		for i := 0; i < upstreamConnections; i++ {
			id := fmt.Sprintf("upstream-%d", i)
			conn, ok := gw.client.upstreamConnections.get(id)
			if !ok || !conn.isConnected() {
				return false
			}
		}
		return true
	}, 5*time.Second, 10*time.Millisecond, "upstream connections not all connected")

	t.Cleanup(cancel)
	return h
}

// NewAgent creates a new test agent with a generated id and returns it.
func (h *gatewayTestHarness) NewAgent(t *testing.T) *testAgent {
	t.Helper()
	id := uuid.New()
	raw := append([]byte(nil), id[:]...)
	agent := newTestAgent(t, h.agentURL, raw)
	h.agents.Store(agent.ID(), agent)
	return agent
}

// CloseUpstreamConnection closes the upstream connection with the given id. It will panic
// if the connection is not found.
//
// This can be used to simulate a connection being closed by the upstream server.
func (h *gatewayTestHarness) CloseUpstreamConnection(t *testing.T, id string) {
	t.Helper()

	for _, c := range h.upstream.connections {
		if c.id == id {
			_ = c.conn.Close()
			return
		}
	}

	t.Fatalf("upstream connection %s not found", id)
}

type upstreamMessage struct {
	// the id of the connection that sent the message
	ConnectionID string
	AgentID      string
	Message      *protobufs.AgentToServer
}

type testUpstreamConnection struct {
	id   string
	conn *websocket.Conn
}

type testOpAMPServer struct {
	t        *testing.T
	server   *httptest.Server
	upgrader websocket.Upgrader

	mu          sync.Mutex
	connections []*testUpstreamConnection

	connectionCount atomic.Int32
	recvCh          chan upstreamMessage
	connCh          chan *websocket.Conn
	errCh           chan error

	// skipAuth when true causes the server to ignore auth requests (never respond)
	skipAuth bool
}

func newTestOpAMPServer(t *testing.T) *testOpAMPServer {
	t.Helper()

	s := &testOpAMPServer{
		t:      t,
		recvCh: make(chan upstreamMessage, 128),
		connCh: make(chan *websocket.Conn, 4),
		errCh:  make(chan error, 4),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s.server = httptest.NewUnstartedServer(http.HandlerFunc(s.handle))
	s.server.Listener = listener
	s.server.Start()
	t.Cleanup(s.Close)

	return s
}

func (s *testOpAMPServer) URL() string {
	return "ws" + s.server.URL[len("http"):]
}

func (s *testOpAMPServer) handle(w http.ResponseWriter, r *http.Request) {
	// extract the connection id from the request
	id := r.Header.Get("X-Opamp-Gateway-Connection-Id")

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.errCh <- fmt.Errorf("upgrade: %w", err)
		return
	}

	s.mu.Lock()
	s.connections = append(s.connections, &testUpstreamConnection{id: id, conn: conn})
	s.mu.Unlock()
	s.connectionCount.Add(1)

	select {
	case s.connCh <- conn:
	default:
	}

	go func() {
		defer func() {
			_ = conn.Close()
			s.connectionCount.Add(-1)
			s.removeConnection(id)
		}()
		s.readLoop(conn, id)
	}()
}

func (s *testOpAMPServer) readLoop(conn *websocket.Conn, id string) {
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || websocket.IsUnexpectedCloseError(err) {
				// unexpected close is expected to happen when the connection is closed
				return
			}
			s.errCh <- fmt.Errorf("read message: %w", err)
			return
		}
		if messageType != websocket.BinaryMessage {
			s.errCh <- fmt.Errorf("unexpected message type: %d", messageType)
			return
		}

		var msg protobufs.AgentToServer
		if err := decodeWSMessage(data, &msg); err != nil {
			s.errCh <- err
			return
		}

		// Handle authentication requests
		if cm := msg.GetCustomMessage(); cm != nil && cm.Capability == OpampGatewayCapability && cm.Type == OpampGatewayConnectType {
			if !s.skipAuth {
				if err := s.respondToConnect(conn, cm.Data); err != nil {
					s.errCh <- err
					return
				}
			}
			continue
		}

		agentID, err := parseAgentID(msg.GetInstanceUid())
		if err != nil {
			s.errCh <- err
			return
		}

		select {
		case s.recvCh <- upstreamMessage{
			ConnectionID: id,
			AgentID:      agentID,
			Message:      &msg,
		}:
		default:
			s.errCh <- fmt.Errorf("recvCh buffer full")
			return
		}
	}
}

// respondToConnect auto-accepts an OpampGatewayConnect authentication request by
// sending back an OpampGatewayConnectResult with Accept: true.
func (s *testOpAMPServer) respondToConnect(conn *websocket.Conn, data []byte) error {
	var connectMsg OpampGatewayConnect
	if err := json.Unmarshal(data, &connectMsg); err != nil {
		return fmt.Errorf("unmarshal connect request: %w", err)
	}

	result := OpampGatewayConnectResult{
		RequestUID:     connectMsg.RequestUID,
		Accept:         true,
		HTTPStatusCode: http.StatusOK,
	}
	resultData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal connect result: %w", err)
	}

	resp := &protobufs.ServerToAgent{
		CustomMessage: &protobufs.CustomMessage{
			Capability: OpampGatewayCapability,
			Type:       OpampGatewayConnectResultType,
			Data:       resultData,
		},
	}
	payload, err := proto.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal connect response: %w", err)
	}

	return writeWSMessage(conn, payload)
}

func (s *testOpAMPServer) WaitForConnection(t *testing.T, timeout time.Duration) *websocket.Conn {
	t.Helper()

	select {
	case conn := <-s.connCh:
		return conn
	case err := <-s.errCh:
		require.NoError(t, err)
		return nil
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for upstream connection")
		return nil
	}
}

func (s *testOpAMPServer) WaitForAnyMessage(t *testing.T, timeout time.Duration) upstreamMessage {
	t.Helper()

	select {
	case msg := <-s.recvCh:
		return msg
	case err := <-s.errCh:
		require.NoError(t, err)
		return upstreamMessage{}
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for upstream message")
		return upstreamMessage{}
	}
}

func (s *testOpAMPServer) WaitForAgentMessage(t *testing.T, agentID string, timeout time.Duration) upstreamMessage {
	t.Helper()

	deadline := time.After(timeout)
	for {
		select {
		case msg := <-s.recvCh:
			if msg.AgentID == agentID {
				return msg
			}
		case err := <-s.errCh:
			require.NoError(t, err)
		case <-deadline:
			t.Fatalf("timed out waiting for message for agent %s", agentID)
		}
	}
}

func (s *testOpAMPServer) removeConnection(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.connections {
		if c.id == id {
			s.connections = append(s.connections[:i], s.connections[i+1:]...)
			return
		}
	}
}

func (s *testOpAMPServer) Send(resp *protobufs.ServerToAgent) error {
	payload, err := proto.Marshal(resp)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.connections) == 0 {
		return fmt.Errorf("no upstream connections")
	}

	// Use the most recent connection (last in slice) so that Send works after
	// a gateway restart where old connections have been removed.
	return writeWSMessage(s.connections[len(s.connections)-1].conn, payload)
}

// Close closes the test OpAMP server. It will close all the connections and the server.
func (s *testOpAMPServer) Close() {
	s.server.Close()

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, conn := range s.connections {
		_ = conn.conn.Close()
	}
}

type testAgent struct {
	t           *testing.T
	conn        *websocket.Conn
	rawID       []byte
	id          string
	sequenceNum uint64
	recvCh      chan *protobufs.ServerToAgent
}

func newTestAgent(t *testing.T, url string, rawID []byte) *testAgent {
	t.Helper()

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)

	id, err := parseAgentID(rawID)
	require.NoError(t, err)

	agent := &testAgent{
		t:           t,
		conn:        conn,
		rawID:       append([]byte(nil), rawID...),
		id:          id,
		sequenceNum: 1,
		recvCh:      make(chan *protobufs.ServerToAgent, 8),
	}

	go agent.readLoop()

	t.Cleanup(func() {
		_ = agent.conn.Close()
	})

	return agent
}

func (a *testAgent) ID() string {
	return a.id
}

func (a *testAgent) RawID() []byte {
	return append([]byte(nil), a.rawID...)
}

// Send sends the given messages to the agent. If no messages are provided, a default
// message is sent.
func (a *testAgent) Send(msgs ...*protobufs.AgentToServer) {
	if len(msgs) == 0 {
		// send a default message
		msgs = []*protobufs.AgentToServer{{}}
	}

	for _, msg := range msgs {
		if len(msg.GetInstanceUid()) == 0 {
			msg.InstanceUid = append([]byte(nil), a.rawID...)
		}

		// assign the next sequence number for this agent
		if msg.SequenceNum == 0 {
			msg.SequenceNum = a.sequenceNum
			a.sequenceNum++
		}

		payload, err := proto.Marshal(msg)
		require.NoError(a.t, err)

		require.NoError(a.t, a.conn.WriteMessage(websocket.BinaryMessage, payload))
	}
}

func (a *testAgent) WaitForMessage(t *testing.T, timeout time.Duration) *protobufs.ServerToAgent {
	t.Helper()

	select {
	case msg, ok := <-a.recvCh:
		if !ok {
			t.Fatalf("agent %s connection closed before receiving message", a.id)
		}
		return proto.Clone(msg).(*protobufs.ServerToAgent)
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for message for agent %s", a.id)
		return nil
	}
}

func (a *testAgent) Close() error {
	return a.conn.Close()
}

func (a *testAgent) readLoop() {
	defer close(a.recvCh)

	for {
		messageType, data, err := a.conn.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.BinaryMessage {
			a.t.Errorf("agent %s received unexpected message type %d", a.id, messageType)
			return
		}

		var msg protobufs.ServerToAgent
		if err := decodeWSMessage(data, &msg); err != nil {
			a.t.Errorf("agent %s failed to decode message: %v", a.id, err)
			return
		}

		select {
		case a.recvCh <- &msg:
		default:
			a.t.Errorf("agent %s receive buffer full", a.id)
			return
		}
	}
}

// newTestAgentWithDialer creates a test agent using a custom websocket dialer (e.g. for TLS).
func newTestAgentWithDialer(t *testing.T, url string, rawID []byte, dialer *websocket.Dialer) *testAgent {
	t.Helper()

	conn, _, err := dialer.Dial(url, nil)
	require.NoError(t, err)

	id, err := parseAgentID(rawID)
	require.NoError(t, err)

	agent := &testAgent{
		t:           t,
		conn:        conn,
		rawID:       append([]byte(nil), rawID...),
		id:          id,
		sequenceNum: 1,
		recvCh:      make(chan *protobufs.ServerToAgent, 8),
	}

	go agent.readLoop()

	t.Cleanup(func() {
		_ = agent.conn.Close()
	})

	return agent
}

// generateTestTLSConfig creates a self-signed certificate and returns the server TLS config
// and a CA cert pool that trusts it.
func generateTestTLSConfig(t *testing.T) (*tls.Config, *x509.CertPool) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	serverCert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	certPool := x509.NewCertPool()
	parsedCert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)
	certPool.AddCert(parsedCert)

	serverTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}

	return serverTLSConfig, certPool
}

// --------------------------------------------------------------------------------------
// Additional test cases

func TestGatewayAuthTimeout(t *testing.T) {
	t.Parallel()

	// Create an upstream server that never responds to auth requests.
	upstream := newTestOpAMPServer(t)
	upstream.skipAuth = true

	testTel := componenttest.NewTelemetry()
	t.Cleanup(func() {
		require.NoError(t, testTel.Shutdown(context.Background()))
	})
	telemetry, err := metadata.NewTelemetryBuilder(testTel.NewTelemetrySettings())
	require.NoError(t, err)

	settings := Settings{
		UpstreamOpAMPAddress: upstream.URL(),
		SecretKey:            "test-secret",
		UpstreamConnections:  1,
		ServerEndpoint:       "127.0.0.1:0",
		AuthTimeout:          500 * time.Millisecond,
	}

	logger := zaptest.NewLogger(t)
	gw := New(logger, settings, telemetry)
	require.NoError(t, gw.Start(context.Background()))
	t.Cleanup(func() { _ = gw.Shutdown(context.Background()) })

	upstream.WaitForConnection(t, 5*time.Second)
	require.Eventually(t, func() bool {
		conn, ok := gw.client.upstreamConnections.get("upstream-0")
		return ok && conn.isConnected()
	}, 5*time.Second, 10*time.Millisecond)

	// Make a plain HTTP request to the gateway server. Since auth never responds,
	// the gateway should return 504 Gateway Timeout.
	serverAddr := gw.server.addr.String()
	resp, err := http.Get(fmt.Sprintf("http://%s%s", serverAddr, handlePath))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
	require.Equal(t, "30", resp.Header.Get("Retry-After"))
}

func TestGatewayUpstreamReconnection(t *testing.T) {
	t.Parallel()

	upstream := newTestOpAMPServer(t)

	testTel := componenttest.NewTelemetry()
	t.Cleanup(func() {
		require.NoError(t, testTel.Shutdown(context.Background()))
	})
	telemetry, err := metadata.NewTelemetryBuilder(testTel.NewTelemetrySettings())
	require.NoError(t, err)

	settings := Settings{
		UpstreamOpAMPAddress: upstream.URL(),
		SecretKey:            "test-secret",
		UpstreamConnections:  1,
		ServerEndpoint:       "127.0.0.1:0",
	}

	logger := zaptest.NewLogger(t)
	gw := New(logger, settings, telemetry)
	require.NoError(t, gw.Start(context.Background()))
	t.Cleanup(func() { _ = gw.Shutdown(context.Background()) })

	upstream.WaitForConnection(t, 5*time.Second)
	require.Eventually(t, func() bool {
		conn, ok := gw.client.upstreamConnections.get("upstream-0")
		return ok && conn.isConnected()
	}, 5*time.Second, 10*time.Millisecond)

	agentURL := fmt.Sprintf("ws://%s%s", gw.server.addr.String(), handlePath)

	// Connect an agent and verify initial round-trip.
	id1 := uuid.New()
	agent1 := newTestAgent(t, agentURL, id1[:])
	agent1.Send(&protobufs.AgentToServer{SequenceNum: 1})
	msg1 := upstream.WaitForAgentMessage(t, agent1.ID(), 5*time.Second)
	require.Equal(t, uint64(1), msg1.Message.GetSequenceNum())

	// Close all upstream connections from the server side to simulate a network failure.
	upstream.mu.Lock()
	for _, c := range upstream.connections {
		_ = c.conn.Close()
	}
	upstream.mu.Unlock()

	// Wait for the gateway to reconnect to upstream.
	upstream.WaitForConnection(t, 10*time.Second)
	require.Eventually(t, func() bool {
		conn, ok := gw.client.upstreamConnections.get("upstream-0")
		return ok && conn.isConnected()
	}, 10*time.Second, 50*time.Millisecond)

	// The original downstream agent was closed by HandleUpstreamClose.
	// Connect a new agent and verify full round-trip on the recovered connection.
	id2 := uuid.New()
	agent2 := newTestAgent(t, agentURL, id2[:])
	agent2.Send(&protobufs.AgentToServer{SequenceNum: 42})
	msg2 := upstream.WaitForAgentMessage(t, agent2.ID(), 5*time.Second)
	require.Equal(t, uint64(42), msg2.Message.GetSequenceNum())

	// Verify upstream can respond to the new agent.
	require.NoError(t, upstream.Send(&protobufs.ServerToAgent{
		InstanceUid:  agent2.RawID(),
		Capabilities: 99,
	}))
	resp := agent2.WaitForMessage(t, 5*time.Second)
	require.Equal(t, uint64(99), resp.GetCapabilities())
}

func TestGatewayTLSConnection(t *testing.T) {
	t.Parallel()

	serverTLSCfg, certPool := generateTestTLSConfig(t)

	upstream := newTestOpAMPServer(t)

	testTel := componenttest.NewTelemetry()
	t.Cleanup(func() {
		require.NoError(t, testTel.Shutdown(context.Background()))
	})
	telemetry, err := metadata.NewTelemetryBuilder(testTel.NewTelemetrySettings())
	require.NoError(t, err)

	settings := Settings{
		UpstreamOpAMPAddress: upstream.URL(),
		SecretKey:            "test-secret",
		UpstreamConnections:  1,
		ServerEndpoint:       "127.0.0.1:0",
		ServerTLS:            serverTLSCfg,
	}

	logger := zaptest.NewLogger(t)
	gw := New(logger, settings, telemetry)
	require.NoError(t, gw.Start(context.Background()))
	t.Cleanup(func() { _ = gw.Shutdown(context.Background()) })

	upstream.WaitForConnection(t, 5*time.Second)
	require.Eventually(t, func() bool {
		conn, ok := gw.client.upstreamConnections.get("upstream-0")
		return ok && conn.isConnected()
	}, 5*time.Second, 10*time.Millisecond)

	// Connect an agent over TLS using wss://.
	agentURL := fmt.Sprintf("wss://%s%s", gw.server.addr.String(), handlePath)
	dialer := &websocket.Dialer{
		TLSClientConfig: &tls.Config{
			RootCAs: certPool,
		},
	}

	id := uuid.New()
	agent := newTestAgentWithDialer(t, agentURL, id[:], dialer)

	// Verify full round-trip over TLS.
	agent.Send(&protobufs.AgentToServer{SequenceNum: 7})
	msg := upstream.WaitForAgentMessage(t, agent.ID(), 5*time.Second)
	require.Equal(t, uint64(7), msg.Message.GetSequenceNum())

	require.NoError(t, upstream.Send(&protobufs.ServerToAgent{
		InstanceUid:  agent.RawID(),
		Capabilities: 77,
	}))
	resp := agent.WaitForMessage(t, 5*time.Second)
	require.Equal(t, uint64(77), resp.GetCapabilities())

	// Verify that a plain ws:// connection is rejected.
	plainURL := fmt.Sprintf("ws://%s%s", gw.server.addr.String(), handlePath)
	_, _, err = websocket.DefaultDialer.Dial(plainURL, nil)
	require.Error(t, err, "plain ws:// connection should fail against TLS server")
}

func TestGatewayGracefulShutdownWithActiveConnections(t *testing.T) {
	t.Parallel()

	upstream := newTestOpAMPServer(t)

	testTel := componenttest.NewTelemetry()
	t.Cleanup(func() {
		require.NoError(t, testTel.Shutdown(context.Background()))
	})
	telemetry, err := metadata.NewTelemetryBuilder(testTel.NewTelemetrySettings())
	require.NoError(t, err)

	settings := Settings{
		UpstreamOpAMPAddress: upstream.URL(),
		SecretKey:            "test-secret",
		UpstreamConnections:  2,
		ServerEndpoint:       "127.0.0.1:0",
	}

	logger := zaptest.NewLogger(t)
	gw := New(logger, settings, telemetry)
	require.NoError(t, gw.Start(context.Background()))

	for i := 0; i < 2; i++ {
		upstream.WaitForConnection(t, 5*time.Second)
	}
	require.Eventually(t, func() bool {
		for i := 0; i < 2; i++ {
			id := fmt.Sprintf("upstream-%d", i)
			conn, ok := gw.client.upstreamConnections.get(id)
			if !ok || !conn.isConnected() {
				return false
			}
		}
		return true
	}, 5*time.Second, 10*time.Millisecond)

	agentURL := fmt.Sprintf("ws://%s%s", gw.server.addr.String(), handlePath)

	// Connect several agents and verify they're active.
	const numAgents = 5
	agents := make([]*testAgent, numAgents)
	for i := 0; i < numAgents; i++ {
		id := uuid.New()
		agents[i] = newTestAgent(t, agentURL, id[:])
		agents[i].Send()
	}

	// Drain all upstream messages to confirm agents are connected.
	for i := 0; i < numAgents; i++ {
		upstream.WaitForAnyMessage(t, 5*time.Second)
	}

	// Shutdown should complete within a reasonable time even with active connections.
	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- gw.Shutdown(context.Background())
	}()

	select {
	case err := <-shutdownDone:
		require.NoError(t, err)
	case <-time.After(15 * time.Second):
		t.Fatal("shutdown did not complete within 15 seconds with active connections")
	}
}

func TestGatewayConcurrentAgentStress(t *testing.T) {
	t.Parallel()

	const numAgents = 50
	const numUpstreamConns = 3

	upstream := newTestOpAMPServer(t)

	testTel := componenttest.NewTelemetry()
	t.Cleanup(func() {
		require.NoError(t, testTel.Shutdown(context.Background()))
	})
	telemetry, err := metadata.NewTelemetryBuilder(testTel.NewTelemetrySettings())
	require.NoError(t, err)

	settings := Settings{
		UpstreamOpAMPAddress: upstream.URL(),
		SecretKey:            "test-secret",
		UpstreamConnections:  numUpstreamConns,
		ServerEndpoint:       "127.0.0.1:0",
	}

	logger := zaptest.NewLogger(t)
	gw := New(logger, settings, telemetry)
	require.NoError(t, gw.Start(context.Background()))
	t.Cleanup(func() { _ = gw.Shutdown(context.Background()) })

	for i := 0; i < numUpstreamConns; i++ {
		upstream.WaitForConnection(t, 5*time.Second)
	}
	require.Eventually(t, func() bool {
		for i := 0; i < numUpstreamConns; i++ {
			id := fmt.Sprintf("upstream-%d", i)
			conn, ok := gw.client.upstreamConnections.get(id)
			if !ok || !conn.isConnected() {
				return false
			}
		}
		return true
	}, 5*time.Second, 10*time.Millisecond)

	agentURL := fmt.Sprintf("ws://%s%s", gw.server.addr.String(), handlePath)

	// Connect all agents sequentially (each connection involves an auth round-trip).
	agents := make([]*testAgent, numAgents)
	for i := 0; i < numAgents; i++ {
		id := uuid.New()
		agents[i] = newTestAgent(t, agentURL, id[:])
	}

	// Send one message from each agent.
	for _, agent := range agents {
		agent.Send()
	}

	// Collect all messages, draining concurrently to avoid buffer overflow.
	received := &sync.Map{}
	collectDone := make(chan struct{})
	go func() {
		defer close(collectDone)
		for i := 0; i < numAgents; i++ {
			select {
			case msg := <-upstream.recvCh:
				received.Store(msg.AgentID, msg.Message)
			case <-time.After(15 * time.Second):
				return
			}
		}
	}()

	select {
	case <-collectDone:
	case <-time.After(20 * time.Second):
		t.Fatal("timed out collecting messages from all agents")
	}

	// Verify every agent's message arrived.
	for _, agent := range agents {
		_, ok := received.Load(agent.ID())
		require.True(t, ok, "missing message from agent %s", agent.ID())
	}

	// Verify load was distributed across multiple upstream connections.
	connectionsSeen := map[string]bool{}
	received.Range(func(_, _ any) bool { return true })

	// Check that all upstream connections have at least some downstream agents.
	for i := 0; i < numUpstreamConns; i++ {
		id := fmt.Sprintf("upstream-%d", i)
		conn, ok := gw.client.upstreamConnections.get(id)
		if ok && conn.downstreamCount() > 0 {
			connectionsSeen[id] = true
		}
	}
	require.Greater(t, len(connectionsSeen), 1,
		"expected agents to be distributed across multiple upstream connections, but only used %d", len(connectionsSeen))

	// Clean up all agents.
	for _, agent := range agents {
		_ = agent.Close()
	}

	// Verify all downstream connections are eventually cleaned up.
	require.Eventually(t, func() bool {
		return gw.server.downstreamConnections.size() == 0
	}, 10*time.Second, 100*time.Millisecond, "downstream connections not fully cleaned up")

	// Verify all agent connection entries are cleaned up (no stale entries).
	require.Eventually(t, func() bool {
		return gw.server.agentConnections.size() == 0
	}, 10*time.Second, 100*time.Millisecond, "agent connections not fully cleaned up")
}
