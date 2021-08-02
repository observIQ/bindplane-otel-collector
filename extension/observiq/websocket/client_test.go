package websocket

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewWebsocketFromConfig(t *testing.T) {
	config := ClientConfig{
		URL:          "test_url",
		Headers:      http.Header{},
		InboundSize:  1,
		OutboundSize: 2,
	}

	client := NewClient(config)
	require.Equal(t, config.URL, client.url)
	require.Equal(t, config.Headers, client.headers)
	require.Equal(t, config.InboundSize, cap(client.inbound))
	require.Equal(t, config.OutboundSize, cap(client.outbound))
}

func TestWebsocketOpenSuccess(t *testing.T) {
	server := NewMockServer(t)
	server.On("ValidateRequest", mock.Anything).Return(nil)
	server.On("OpenedConnection", mock.Anything).Return()

	config := ClientConfig{
		URL: fmt.Sprintf("ws://localhost:%d", server.port),
	}
	client := NewClient(config)

	server.Start()
	defer server.Stop()

	conn, err := client.Open(context.Background())
	require.NoError(t, err)
	require.NotNil(t, conn)
	conn.Close()
}

func TestWebsocketOpenFailure(t *testing.T) {
	server := NewMockServer(t)
	server.On("ValidateRequest", mock.Anything).Return(nil)
	server.On("OpenedConnection", mock.Anything).Return()

	config := ClientConfig{
		URL: fmt.Sprintf("ws://localhost:%d", server.port),
	}
	client := NewClient(config)

	conn, err := client.Open(context.Background())
	require.Error(t, err)
	require.Nil(t, conn)
}

func TestWebsocketInbound(t *testing.T) {
	client := Client{
		inbound: make(chan []byte, 20),
	}
	require.Equal(t, client.inbound, client.Inbound())
}

func TestWebsocketOutbound(t *testing.T) {
	client := Client{
		outbound: make(chan []byte, 20),
	}
	require.Equal(t, client.outbound, client.Outbound())
}

func TestWebsocketInboundMessage(t *testing.T) {
	server := NewMockServer(t)
	server.On("ValidateRequest", mock.Anything).Return(nil)
	server.On("OpenedConnection", mock.Anything).Return()

	config := ClientConfig{
		URL:         fmt.Sprintf("ws://localhost:%d", server.port),
		InboundSize: 20,
	}
	client := NewClient(config)

	server.Start()
	defer server.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := client.Open(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()

	go client.HandleInbound(ctx, conn)
	server.SendMessage(websocket.TextMessage, []byte("test"))
	message := <-client.Inbound()
	require.Equal(t, []byte("test"), message)
}

func TestWebsocketInboundCtx(t *testing.T) {
	server := NewMockServer(t)
	server.On("ValidateRequest", mock.Anything).Return(nil)
	server.On("OpenedConnection", mock.Anything).Return()

	config := ClientConfig{
		URL:         fmt.Sprintf("ws://localhost:%d", server.port),
		InboundSize: 20,
	}
	client := NewClient(config)

	server.Start()
	defer server.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := client.Open(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()

	cancel()
	err = client.HandleInbound(ctx, conn)
	require.Error(t, err)
	require.Contains(t, context.Canceled.Error(), err.Error())
}

func TestWebsocketInboundError(t *testing.T) {
	server := NewMockServer(t)
	server.On("ValidateRequest", mock.Anything).Return(nil)
	server.On("OpenedConnection", mock.Anything).Return()

	config := ClientConfig{
		URL:         fmt.Sprintf("ws://localhost:%d", server.port),
		InboundSize: 20,
	}
	client := NewClient(config)

	server.Start()
	defer server.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := client.Open(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn)

	errChan := make(chan error)
	go func() {
		errChan <- client.HandleInbound(ctx, conn)
	}()
	client.Close(conn)

	err = <-errChan
	require.Error(t, err)
	require.Contains(t, ErrConnectionClosed.Error(), err.Error())
}

func TestWebsocketOutboundMessage(t *testing.T) {
	messageChan := make(chan []byte, 5)

	server := NewMockServer(t)
	server.On("ValidateRequest", mock.Anything).Return(nil)
	server.On("OpenedConnection", mock.Anything).Return()
	server.On("ReceivedMessage", mock.Anything).Run(func(args mock.Arguments) {
		message := args.Get(0).([]byte)
		messageChan <- message
	})

	config := ClientConfig{
		URL:          fmt.Sprintf("ws://localhost:%d", server.port),
		OutboundSize: 20,
	}
	client := NewClient(config)

	server.Start()
	defer server.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := client.Open(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()

	go client.HandleOutbound(ctx, conn)
	client.Outbound() <- []byte("test")
	message := <-messageChan
	require.Equal(t, []byte("test"), message)
}

func TestWebsocketOutboundCtx(t *testing.T) {
	server := NewMockServer(t)
	server.On("ValidateRequest", mock.Anything).Return(nil)
	server.On("OpenedConnection", mock.Anything).Return()

	config := ClientConfig{
		URL:          fmt.Sprintf("ws://localhost:%d", server.port),
		OutboundSize: 20,
	}
	client := NewClient(config)

	server.Start()
	defer server.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := client.Open(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()

	cancel()
	err = client.HandleOutbound(ctx, conn)
	require.Error(t, err)
	require.Contains(t, context.Canceled.Error(), err.Error())
}

func TestWebsocketOutboundError(t *testing.T) {
	server := NewMockServer(t)
	server.On("ValidateRequest", mock.Anything).Return(nil)
	server.On("OpenedConnection", mock.Anything).Return()

	config := ClientConfig{
		URL:          fmt.Sprintf("ws://localhost:%d", server.port),
		OutboundSize: 20,
	}
	client := NewClient(config)

	server.Start()
	defer server.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := client.Open(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn)
	conn.Close()

	errChan := make(chan error)
	go func() {
		errChan <- client.HandleOutbound(ctx, conn)
	}()
	client.Close(conn)
	client.Outbound() <- []byte("test")

	err = <-errChan
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown write error")
}

// MockServer is a server used for testing
type MockServer struct {
	Upgrader websocket.Upgrader
	mock.Mock
	address    string
	port       int
	httpServer *http.Server
	conn       *websocket.Conn
}

// ValidateRequest is a stub that is called when a request is received
func (s *MockServer) ValidateRequest(r *http.Request) error {
	args := s.Called(r)
	return args.Error(0)
}

// OpenedConnection is a stub that is called when a connection is opened
func (s *MockServer) OpenedConnection(conn *websocket.Conn) {
	s.Called(conn)
}

// ReceivedMessage is a stub that is called when a message is received
func (s *MockServer) ReceivedMessage(message []byte) {
	s.Called(message)
}

// SendMessage is used to send a message to a connection
func (s *MockServer) SendMessage(messageType int, message []byte) error {
	return s.conn.WriteMessage(messageType, message)
}

// HandleRequest converts an http request to a websocket connection and handles incoming traffic
func (s *MockServer) HandleRequest(w http.ResponseWriter, r *http.Request) {
	err := s.ValidateRequest(r)
	if err != nil {
		return
	}

	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.conn = conn
	s.OpenedConnection(conn)
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		s.ReceivedMessage(message)
	}

	s.conn = nil
}

// Start will initiate the server and begin listening for requests.
func (s *MockServer) Start() {
	router := mux.NewRouter()
	router.HandleFunc("/", s.HandleRequest)
	s.httpServer = &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("%s:%d", s.address, s.port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	ready := make(chan struct{})
	go func() {
		ready <- struct{}{}
		s.httpServer.ListenAndServe()
	}()
	<-ready
}

// Stop will close the server and stop listening for requests.
func (s *MockServer) Stop() {
	if s.httpServer != nil {
		s.httpServer.Close()
	}
}

// NewMockServer returns a new mock server.
func NewMockServer(t *testing.T) *MockServer {
	port, err := findOpenPort()
	require.NoError(t, err, "Could not find open port for test")
	fmt.Printf("Websocket Server using port %d\n", port)

	return &MockServer{
		Upgrader: websocket.Upgrader{},
		address:  "127.0.0.1",
		port:     port,
	}
}

// findOpenPort attempts to find an open port on the localhost.
func findOpenPort() (int, error) {
	for i := 0; i < 10; i++ {
		port := randomNumberInRange(49152, 61000)
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			l.Close()
			return port, nil
		}
	}

	return 0, errors.New("unable to find open port")
}

// randomNumberInRange returns a random number within the supplied range.
func randomNumberInRange(min int, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}
