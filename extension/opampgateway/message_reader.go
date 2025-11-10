package opampgateway

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type messageReader struct {
	conn      *websocket.Conn
	callbacks readerCallbacks
	id        string
	logger    *zap.Logger
}

type readerCallbacks struct {
	OnMessage func(ctx context.Context, messageType int, message *message) error
	OnError   func(ctx context.Context, err error)
}

func newMessageReader(conn *websocket.Conn, id string, callbacks readerCallbacks, logger *zap.Logger) *messageReader {
	return &messageReader{conn: conn, id: id, callbacks: callbacks, logger: logger.Named("message-reader").With(zap.String("id", id))}
}

// loop will read messages from the connection and call the OnMessage callback for each
// message. It will call OnError if an error occurs and then return. If the connection is
// closed, it will not call OnError, but will stop reading and return. It will stop
// reading when the context is done.
func (r *messageReader) loop(ctx context.Context, messageNumber int) {
	// loop until the connection is closed
	for {
		// try to read the message. ReadMessage will block until a message is received or the
		// connection is closed.
		messageType, messageBytes, err := r.conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				// context is done, so we return cleanly
				r.logger.Info("context done")
				return
			}
			if errors.Is(err, net.ErrClosed) || websocket.IsUnexpectedCloseError(err) {
				// unexpected close is expected to happen when the connection is closed
				r.logger.Info("closed")
				return
			}
			r.logger.Error("read message", zap.Error(err))
			r.callbacks.OnError(ctx, fmt.Errorf("read message: %w", err))
			return
		}

		// handle the message using the callback
		message := newMessage(messageNumber, messageBytes)
		if err := r.callbacks.OnMessage(ctx, messageType, message); err != nil {
			r.callbacks.OnError(ctx, fmt.Errorf("handle message: %w", err))
			return
		}
		messageNumber++
	}
}
