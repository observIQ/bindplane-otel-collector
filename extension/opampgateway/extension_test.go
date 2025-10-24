package opampgateway

import (
	"context"
	"fmt"
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
		h.gateway.client.assignedConnectionsMtx.RLock()
		defer h.gateway.client.assignedConnectionsMtx.RUnlock()
		_, ok := h.gateway.client.assignedConnections[agent.ID()]
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
		h.gateway.client.assignedConnectionsMtx.RLock()
		defer h.gateway.client.assignedConnectionsMtx.RUnlock()
		_, ok := h.gateway.client.assignedConnections[agent.ID()]
		return !ok
	}, 5*time.Second, 50*time.Millisecond, "upstream assignment still registered")
}

// --------------------------------------------------------------------------------------
// test harness

type gatewayTestHarness struct {
	t        *testing.T
	ctx      context.Context
	cancel   context.CancelFunc
	gateway  *OpAMPGateway
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

	cfg := &Config{
		UpstreamOpAMPAddress: upstream.URL(),
		SecretKey:            "test-secret",
		UpstreamConnections:  upstreamConnections,
		OpAMPServer: &OpAMPServer{
			Endpoint: "127.0.0.1:0",
		},
	}

	logger := zaptest.NewLogger(t)

	gateway := newOpAMPGateway(logger, cfg, telemetry)

	require.NoError(t, gateway.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, gateway.Shutdown(context.Background()))
	})

	h := &gatewayTestHarness{
		t:        t,
		ctx:      ctx,
		cancel:   cancel,
		gateway:  gateway,
		upstream: upstream,
		agentURL: fmt.Sprintf("ws://%s%s", gateway.server.addr.String(), handlePath),
	}

	for i := 0; i < upstreamConnections; i++ {
		upstream.WaitForConnection(t, 5*time.Second)
	}

	t.Cleanup(cancel)
	return h
}

func (h *gatewayTestHarness) NewAgent(t *testing.T) *testAgent {
	t.Helper()
	id := uuid.New()
	raw := append([]byte(nil), id[:]...)
	agent := newTestAgent(t, h.agentURL, raw)
	h.agents.Store(agent.ID(), agent)
	return agent
}

type upstreamMessage struct {
	AgentID string
	Message *protobufs.AgentToServer
}

type testOpAMPServer struct {
	t        *testing.T
	server   *httptest.Server
	upgrader websocket.Upgrader

	mu          sync.Mutex
	connections []*websocket.Conn

	connectionCount atomic.Int32
	recvCh          chan upstreamMessage
	connCh          chan *websocket.Conn
	errCh           chan error
}

func newTestOpAMPServer(t *testing.T) *testOpAMPServer {
	t.Helper()

	s := &testOpAMPServer{
		t:      t,
		recvCh: make(chan upstreamMessage, 32),
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
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.errCh <- fmt.Errorf("upgrade: %w", err)
		return
	}

	s.mu.Lock()
	s.connections = append(s.connections, conn)
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
		}()
		s.readLoop(conn)
	}()
}

func (s *testOpAMPServer) readLoop(conn *websocket.Conn) {
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			s.errCh <- err
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

		agentID, err := parseAgentID(msg.GetInstanceUid())
		if err != nil {
			s.errCh <- err
			return
		}

		select {
		case s.recvCh <- upstreamMessage{
			AgentID: agentID,
			Message: &msg,
		}:
		default:
			s.errCh <- fmt.Errorf("recvCh buffer full")
			return
		}
	}
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

func (s *testOpAMPServer) WaitForAgentMessage(t *testing.T, agentID string, timeout time.Duration) *protobufs.AgentToServer {
	t.Helper()

	deadline := time.After(timeout)
	for {
		select {
		case msg := <-s.recvCh:
			if msg.AgentID == agentID {
				return proto.Clone(msg.Message).(*protobufs.AgentToServer)
			}
		case err := <-s.errCh:
			require.NoError(t, err)
		case <-deadline:
			t.Fatalf("timed out waiting for message for agent %s", agentID)
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

	return writeWSMessage(context.Background(), s.connections[0], payload)
}

func (s *testOpAMPServer) Close() {
	s.server.Close()

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, conn := range s.connections {
		_ = conn.Close()
	}
}

type testAgent struct {
	t      *testing.T
	conn   *websocket.Conn
	rawID  []byte
	id     string
	recvCh chan *protobufs.ServerToAgent
}

func newTestAgent(t *testing.T, url string, rawID []byte) *testAgent {
	t.Helper()

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)

	id, err := parseAgentID(rawID)
	require.NoError(t, err)

	agent := &testAgent{
		t:      t,
		conn:   conn,
		rawID:  append([]byte(nil), rawID...),
		id:     id,
		recvCh: make(chan *protobufs.ServerToAgent, 8),
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

func (a *testAgent) Send(msg *protobufs.AgentToServer) {
	if len(msg.GetInstanceUid()) == 0 {
		msg.InstanceUid = append([]byte(nil), a.rawID...)
	}

	payload, err := proto.Marshal(msg)
	require.NoError(a.t, err)

	require.NoError(a.t, a.conn.WriteMessage(websocket.BinaryMessage, payload))
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
