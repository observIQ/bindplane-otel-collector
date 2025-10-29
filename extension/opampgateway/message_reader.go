package opampgateway

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/gorilla/websocket"
)

type messageReader struct {
	conn      *websocket.Conn
	callbacks readerCallbacks
}

type readerCallbacks struct {
	OnMessage func(ctx context.Context, messageNumber int, messageType int, messageBytes []byte) error
	OnError   func(ctx context.Context, err error)
}

func newMessageReader(conn *websocket.Conn, callbacks readerCallbacks) *messageReader {
	return &messageReader{conn: conn, callbacks: callbacks}
}

// loop will read messages from the connection and call the OnMessage callback for each
// message. It will call OnError if an error occurs and then return. If the connection is
// closed, it will not call OnError, but will stop reading and return. It will stop
// reading when the context is done.
func (r *messageReader) loop(ctx context.Context, messageNumber int) {
	// loop until the connection is closed
	for {
		// try to read the message
		messageType, messageBytes, err := r.conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				// context is done, so we return cleanly
				return
			}
			if errors.Is(err, net.ErrClosed) || websocket.IsUnexpectedCloseError(err) {
				// unexpected close is expected to happen when the connection is closed
				return
			}
			r.callbacks.OnError(ctx, fmt.Errorf("read message: %w", err))
			return
		}

		// handle the message using the callback
		if err := r.callbacks.OnMessage(ctx, messageNumber, messageType, messageBytes); err != nil {
			r.callbacks.OnError(ctx, fmt.Errorf("handle message: %w", err))
			return
		}
		messageNumber++
	}
}
