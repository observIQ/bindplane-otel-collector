package websocket

import (
	"context"

	"github.com/gorilla/websocket"
)

// Websocket is a client that communicates to a server through a websocket connection.
type Websocket interface {
	// Open will open the connection to the server.
	Open(ctx context.Context) (*websocket.Conn, error)

	// Close will attempt to gracefully close the supplied connection
	Close(conn *websocket.Conn) error

	// Send returns the client's channel for sending messages.
	Inbound() chan []byte

	// Receive returns the client's channel for receiving messages.
	Outbound() chan []byte

	// HandleInbound will handle reading inbound traffic from the supplied connection.
	HandleInbound(ctx context.Context, conn *websocket.Conn) error

	// HandleOutbound will handle writing outbound traffic to the supplied connection.
	HandleOutbound(ctx context.Context, conn *websocket.Conn) error
}
