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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// wsTestPair creates an httptest server with a websocket handler and returns
// the server-side and client-side websocket connections.
func wsTestPair(t *testing.T, handler func(conn *websocket.Conn)) *websocket.Conn {
	t.Helper()

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		handler(conn)
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	t.Cleanup(func() { clientConn.Close() })

	return clientConn
}

func TestMessageReaderLoopReceivesMessages(t *testing.T) {
	var mu sync.Mutex
	var received []*message
	var receivedTypes []int

	serverReady := make(chan struct{})

	clientConn := wsTestPair(t, func(conn *websocket.Conn) {
		defer conn.Close()
		<-serverReady
		conn.WriteMessage(websocket.BinaryMessage, []byte("hello"))
		conn.WriteMessage(websocket.BinaryMessage, []byte("world"))
		// close cleanly so the reader exits
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		// wait for the client to read the close
		time.Sleep(100 * time.Millisecond)
	})

	reader := newMessageReader(clientConn, "test", readerCallbacks{
		OnMessage: func(_ context.Context, messageType int, msg *message) error {
			mu.Lock()
			defer mu.Unlock()
			received = append(received, msg)
			receivedTypes = append(receivedTypes, messageType)
			return nil
		},
		OnError: func(_ context.Context, err error) {
			t.Errorf("unexpected OnError: %v", err)
		},
	}, zap.NewNop())

	close(serverReady)
	reader.loop(context.Background(), 0)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, received, 2)
	assert.Equal(t, []byte("hello"), received[0].data)
	assert.Equal(t, 0, received[0].number)
	assert.Equal(t, []byte("world"), received[1].data)
	assert.Equal(t, 1, received[1].number)
	assert.Equal(t, websocket.BinaryMessage, receivedTypes[0])
	assert.Equal(t, websocket.BinaryMessage, receivedTypes[1])
}

func TestMessageReaderLoopStartingMessageNumber(t *testing.T) {
	var mu sync.Mutex
	var received []*message

	serverReady := make(chan struct{})

	clientConn := wsTestPair(t, func(conn *websocket.Conn) {
		defer conn.Close()
		<-serverReady
		conn.WriteMessage(websocket.BinaryMessage, []byte("msg"))
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		time.Sleep(100 * time.Millisecond)
	})

	reader := newMessageReader(clientConn, "test", readerCallbacks{
		OnMessage: func(_ context.Context, _ int, msg *message) error {
			mu.Lock()
			defer mu.Unlock()
			received = append(received, msg)
			return nil
		},
		OnError: func(_ context.Context, err error) {
			t.Errorf("unexpected OnError: %v", err)
		},
	}, zap.NewNop())

	close(serverReady)
	reader.loop(context.Background(), 42)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, received, 1)
	assert.Equal(t, 42, received[0].number)
}

func TestMessageReaderLoopContextCancelled(t *testing.T) {
	onErrorCalled := false

	clientConn := wsTestPair(t, func(conn *websocket.Conn) {
		defer conn.Close()
		// don't send anything; just hold the connection open
		time.Sleep(2 * time.Second)
	})

	ctx, cancel := context.WithCancel(context.Background())

	reader := newMessageReader(clientConn, "test", readerCallbacks{
		OnMessage: func(_ context.Context, _ int, _ *message) error {
			t.Error("unexpected OnMessage")
			return nil
		},
		OnError: func(_ context.Context, _ error) {
			onErrorCalled = true
		},
	}, zap.NewNop())

	done := make(chan struct{})
	go func() {
		reader.loop(ctx, 0)
		close(done)
	}()

	// cancel the context and close the connection to unblock ReadMessage
	cancel()
	clientConn.Close()

	select {
	case <-done:
		// loop returned
	case <-time.After(5 * time.Second):
		t.Fatal("loop did not return after context cancellation")
	}
	assert.False(t, onErrorCalled, "OnError should not be called on context cancellation")
}

func TestMessageReaderLoopConnectionClosed(t *testing.T) {
	onErrorCalled := false
	serverReady := make(chan struct{})

	clientConn := wsTestPair(t, func(conn *websocket.Conn) {
		defer conn.Close()
		<-serverReady
		// close with normal closure
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
		time.Sleep(100 * time.Millisecond)
	})

	reader := newMessageReader(clientConn, "test", readerCallbacks{
		OnMessage: func(_ context.Context, _ int, _ *message) error {
			t.Error("unexpected OnMessage")
			return nil
		},
		OnError: func(_ context.Context, _ error) {
			onErrorCalled = true
		},
	}, zap.NewNop())

	close(serverReady)
	reader.loop(context.Background(), 0)
	assert.False(t, onErrorCalled, "OnError should not be called on clean close")
}

func TestMessageReaderLoopOnMessageError(t *testing.T) {
	var onErrorErr error
	serverReady := make(chan struct{})

	clientConn := wsTestPair(t, func(conn *websocket.Conn) {
		defer conn.Close()
		<-serverReady
		conn.WriteMessage(websocket.BinaryMessage, []byte("trigger"))
		time.Sleep(500 * time.Millisecond)
	})

	callbackErr := errors.New("callback failed")

	reader := newMessageReader(clientConn, "test", readerCallbacks{
		OnMessage: func(_ context.Context, _ int, _ *message) error {
			return callbackErr
		},
		OnError: func(_ context.Context, err error) {
			onErrorErr = err
		},
	}, zap.NewNop())

	close(serverReady)
	reader.loop(context.Background(), 0)

	require.Error(t, onErrorErr)
	assert.ErrorIs(t, onErrorErr, callbackErr)
	assert.Contains(t, onErrorErr.Error(), "handle message")
}

func TestMessageReaderLoopStopsAfterOnMessageError(t *testing.T) {
	var mu sync.Mutex
	messageCount := 0
	serverReady := make(chan struct{})

	clientConn := wsTestPair(t, func(conn *websocket.Conn) {
		defer conn.Close()
		<-serverReady
		// send two messages; reader should stop after the first
		conn.WriteMessage(websocket.BinaryMessage, []byte("first"))
		conn.WriteMessage(websocket.BinaryMessage, []byte("second"))
		time.Sleep(500 * time.Millisecond)
	})

	reader := newMessageReader(clientConn, "test", readerCallbacks{
		OnMessage: func(_ context.Context, _ int, _ *message) error {
			mu.Lock()
			defer mu.Unlock()
			messageCount++
			return errors.New("stop")
		},
		OnError: func(_ context.Context, _ error) {},
	}, zap.NewNop())

	close(serverReady)
	reader.loop(context.Background(), 0)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, messageCount, "should stop reading after first OnMessage error")
}
