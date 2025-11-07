package opampgateway

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type downstreamConnection struct {
	id   string
	conn *websocket.Conn

	writeChan       chan []byte
	readerErrorChan chan error

	readerDone chan struct{}
	writerDone chan struct{}

	upstreamConnection *upstreamConnection

	logger *zap.Logger
}

func newDownstreamConnection(conn *websocket.Conn, upstreamConnection *upstreamConnection, id string, logger *zap.Logger) *downstreamConnection {
	return &downstreamConnection{
		conn:               conn,
		upstreamConnection: upstreamConnection,
		id:                 id,
		logger:             logger.Named("downstream-connection").With(zap.String("id", id)),
		writeChan:          make(chan []byte),

		// the error channel is buffered to prevent blocking the reader goroutine if it
		// encounters an error. it will return immediately after reporting the error and the
		// main connection goroutine will handle the error.
		readerErrorChan: make(chan error, 1),
		readerDone:      make(chan struct{}),
		writerDone:      make(chan struct{}),
	}
}

// start will start the reader and writer goroutines and wait for the context to be done
// or an error to be sent on the error channel. if an error is sent on the error channel,
// the connection will be stopped and the context will be cancelled.
func (c *downstreamConnection) start(ctx context.Context, callbacks ConnectionCallbacks[*downstreamConnection]) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// start the reader in a separate goroutine and cancel the context if it returns, likely
	// due to an error or the connection being closed
	go func() {
		defer cancel()
		c.startReader(ctx, callbacks)
	}()

	// block while writing messages to the connection. a connection close will unblock the writer.
	err := c.startWriter(ctx)
	if err != nil {
		c.logger.Error("error in connection writer", zap.Error(err))
		callbacks.OnError(ctx, c, err)
	}

	// if the writer returns, cancel the context to stop the reader
	cancel()

	// wait for the reader and writer to finish
	<-c.readerDone
	<-c.writerDone

	// check for errors from the reader
	select {
	case err := <-c.readerErrorChan:
		c.logger.Error("error in connection reader", zap.Error(err))
		callbacks.OnError(ctx, c, err)
	default:
	}

	// Call the on close handler
	err = callbacks.OnClose(ctx, c)
	if err != nil {
		c.logger.Error("error in on close handler", zap.Error(err))
	}
}

// send will send a message to the connection by putting it on the write channel. the
// writer goroutine will handle sending the message to the connection.
func (c *downstreamConnection) send(message []byte) {
	c.logger.Info("sending message", zap.String("message", string(message)))
	c.writeChan <- message
}

func (c *downstreamConnection) close() error {
	c.logger.Info("downstream connection closing")
	err := c.conn.Close()
	if err != nil {
		return fmt.Errorf("close connection: %w", err)
	}
	return nil
}

// --------------------------------------------------------------------------------------
// reader goroutine

func (c *downstreamConnection) startReader(ctx context.Context, callbacks ConnectionCallbacks[*downstreamConnection]) {
	defer close(c.readerDone)

	reader := newMessageReader(c.conn, readerCallbacks{
		OnMessage: func(ctx context.Context, messageNumber int, messageType int, messageBytes []byte) error {
			return callbacks.OnMessage(ctx, c, messageNumber, messageType, messageBytes)
		},
		OnError: func(ctx context.Context, err error) {
			callbacks.OnError(ctx, c, err)
		},
	})

	reader.loop(ctx, 0)
}

// --------------------------------------------------------------------------------------
// writer goroutine

func (c *downstreamConnection) startWriter(ctx context.Context) error {
	defer close(c.writerDone)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("writer context done")
			// closing the connection will cause ReadMessage to unblock and return an error
			err := c.conn.Close()
			if err != nil {
				// log the error but return nil to avoid propagating the error to the caller
				c.logger.Error("error closing connection", zap.Error(err))
			}
			return nil
		case message, ok := <-c.writeChan:
			if !ok {
				// the write channel is closed, so we return
				c.logger.Info("write channel closed")
				return nil
			}
			err := writeWSMessage(c.conn, message)
			if err != nil {
				return fmt.Errorf("write message: %w", err)
			}
		}
	}
}
