package opampgateway

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type ConnectionCallbacks struct {
	// OnMessage is called when a message is received.
	OnMessage func(ctx context.Context, connection *connection, messageNumber int, messageType int, messageBytes []byte) error

	// OnError is called when an error occurs. The connection will be closed and the context
	// will be cancelled after this call.
	OnError func(ctx context.Context, connection *connection, err error)

	// OnClose is called when the connection is closed.
	OnClose func(ctx context.Context, connection *connection) error
}

type connection struct {
	id   string
	conn *websocket.Conn

	writeChan       chan []byte
	readerErrorChan chan error

	readerDone chan struct{}
	writerDone chan struct{}

	count  atomic.Int32
	logger *zap.Logger
}

func newConnection(conn *websocket.Conn, id string, logger *zap.Logger) *connection {
	return &connection{
		conn:      conn,
		id:        id,
		logger:    logger,
		writeChan: make(chan []byte),

		// the error channel is buffered to prevent blocking the reader goroutine if it
		// encounters an error. it will return immediately after reporting the error and the
		// main connection goroutine will handle the error.
		readerErrorChan: make(chan error, 1),
		readerDone:      make(chan struct{}),
		writerDone:      make(chan struct{}),
	}
}

func (c *connection) agentCount() int {
	return int(c.count.Load())
}

func (c *connection) incrementAgentCount() {
	c.count.Add(1)
}

func (c *connection) decrementAgentCount() {
	c.count.Add(-1)
}

// start will start the reader and writer goroutines and wait for the context to be done
// or an error to be sent on the error channel. if an error is sent on the error channel,
// the connection will be stopped and the context will be cancelled.
func (c *connection) start(ctx context.Context, callbacks ConnectionCallbacks) {
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
		c.logger.Error("error in connection writer", zap.Error(err), zap.String("id", c.id))
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
		c.logger.Error("error in connection reader", zap.Error(err), zap.String("id", c.id))
		callbacks.OnError(ctx, c, err)
	default:
	}

	// Call the on close handler
	err = callbacks.OnClose(ctx, c)
	if err != nil {
		c.logger.Error("error in on close handler", zap.Error(err), zap.String("id", c.id))
	}
}

// send will send a message to the connection by putting it on the write channel. the
// writer goroutine will handle sending the message to the connection.
func (c *connection) send(message []byte) {
	c.logger.Info("sending message", zap.String("id", c.id), zap.String("message", string(message)))
	c.writeChan <- message
}

// --------------------------------------------------------------------------------------
// reader goroutine

func (c *connection) startReader(ctx context.Context, callbacks ConnectionCallbacks) {
	defer close(c.readerDone)

	messageNumber := 0
	// loop until the connection is closed
	for {
		// try to read the message
		messageType, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				// context is done, so we return cleanly
				return
			}
			if websocket.IsUnexpectedCloseError(err) {
				// unexpected close is expected to happen when the connection is closed
				return
			}
			c.readerErrorChan <- fmt.Errorf("read message: %w", err)
			return
		}

		// handle the message using the callback
		if err := callbacks.OnMessage(ctx, c, messageNumber, messageType, messageBytes); err != nil {
			c.readerErrorChan <- fmt.Errorf("handle message: %w", err)
			return
		}
		messageNumber++
	}
}

// --------------------------------------------------------------------------------------
// writer goroutine

func (c *connection) startWriter(ctx context.Context) error {
	defer close(c.writerDone)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("writer context done", zap.String("id", c.id))
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
				c.logger.Info("write channel closed", zap.String("id", c.id))
				return nil
			}
			err := writeWSMessage(ctx, c.conn, message)
			if err != nil {
				return fmt.Errorf("write message: %w", err)
			}
		}
	}
}
