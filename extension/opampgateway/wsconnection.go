package opampgateway

import (
	"context"
	"net"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

// borrowed from opamp-go library

// wsConnection represents a persistent OpAMP connection over a WebSocket.
type wsConnection struct {
	// The websocket library does not allow multiple concurrent write operations,
	// so ensure that we only have a single operation in progress at a time.
	// For more: https://pkg.go.dev/github.com/gorilla/websocket#hdr-Concurrency
	connMutex sync.Mutex
	wsConn    *websocket.Conn
	closed    atomic.Bool
}

func newWSConnection(wsConn *websocket.Conn) *wsConnection {
	return &wsConnection{wsConn: wsConn}
}

func (c *wsConnection) Connection() net.Conn {
	return c.wsConn.UnderlyingConn()
}

func (c *wsConnection) Send(ctx context.Context, message []byte) error {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	return writeWSMessage(ctx, c.wsConn, message)
}

func (c *wsConnection) Disconnect() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	return c.wsConn.Close()
}
