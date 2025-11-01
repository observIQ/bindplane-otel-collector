package opampgateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type upstreamConnection struct {
	id       string
	settings upstreamConnectionSettings
	dialer   websocket.Dialer

	writeChan       chan []byte
	readerErrorChan chan error

	writerDone chan struct{}

	count  atomic.Int32
	logger *zap.Logger
}

type upstreamConnectionSettings struct {
	endpoint  string
	secretKey string
}

func newUpstreamConnection(dialer websocket.Dialer, settings upstreamConnectionSettings, id string, logger *zap.Logger) *upstreamConnection {
	return &upstreamConnection{
		dialer:    dialer,
		settings:  settings,
		id:        id,
		logger:    logger,
		writeChan: make(chan []byte),

		// the error channel is buffered to prevent blocking the reader goroutine if it
		// encounters an error. it will return immediately after reporting the error and the
		// main connection goroutine will handle the error.
		readerErrorChan: make(chan error, 1),
		writerDone:      make(chan struct{}),
	}
}

func (c *upstreamConnection) agentCount() int {
	return int(c.count.Load())
}

func (c *upstreamConnection) incrementAgentCount() {
	c.count.Add(1)
}

func (c *upstreamConnection) decrementAgentCount() {
	c.count.Add(-1)
}

// start will start the reader and writer goroutines and wait for the context to be done
// or an error to be sent on the error channel. if an error is sent on the error channel,
// the connection will be stopped and the context will be cancelled.
func (c *upstreamConnection) start(ctx context.Context, callbacks ConnectionCallbacks[*upstreamConnection]) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// block while writing messages to the connection. a connection close will unblock the writer.
	err := c.startWriter(ctx, callbacks)
	if err != nil {
		c.logger.Error("error in connection writer", zap.Error(err), zap.String("id", c.id))
		callbacks.OnError(ctx, c, err)
	}

	// if the writer returns, cancel the context to stop the reader
	cancel()

	// wait for the reader and writer to finish
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
func (c *upstreamConnection) send(message []byte) {
	c.logger.Info("sending message", zap.String("id", c.id), zap.String("message", string(message)))
	c.writeChan <- message
}

// --------------------------------------------------------------------------------------
// reader goroutine

func (c *upstreamConnection) startReader(ctx context.Context, conn *websocket.Conn, callbacks ConnectionCallbacks[*upstreamConnection]) {
	reader := newMessageReader(conn, readerCallbacks{
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

func (c *upstreamConnection) startWriter(ctx context.Context, callbacks ConnectionCallbacks[*upstreamConnection]) error {
	defer close(c.writerDone)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// nextMessage is the message that was not written to the connection because the
	// connection was closed. it will be written to the connection when it is reconnected.
	// it is used to avoid losing messages when the connection is closed.
	var nextMessage []byte

	for {
		// ensure connected will do infinite retries until the context is done or an error is
		// returned. this could mean that it takes a while to open the connections to the
		// upstream server.
		conn, err := c.ensureConnected(ctx, c.id)
		if err != nil {
			if ctx.Err() != nil {
				// context is done, so we return
				return nil
			}
			// shutdown, not really an error
			c.logger.Error("ensure connected", zap.Error(err))
			return nil
		}

		readerDone := make(chan struct{})

		// start the reader in a separate goroutine and cancel the context if it returns, likely
		// due to an error or the connection being closed
		go func() {
			defer cancel()
			defer close(readerDone)
			c.startReader(ctx, conn, callbacks)
		}()

		nextMessage, err = c.writerLoop(ctx, conn, nextMessage)
		if err != nil {
			c.logger.Error("writer loop", zap.Error(err))
		}

		// wait for the reader to finish
		<-readerDone
	}
}

// writerLoop will loop until the context is done or the write channel is closed. it takes
// the next message to write and returns the next message to write and an error if one
// occurs.
func (c *upstreamConnection) writerLoop(ctx context.Context, conn *websocket.Conn, nextMessage []byte) ([]byte, error) {
	// attempt to write the next message if one is pending
	if nextMessage != nil {
		err := writeWSMessage(conn, nextMessage)
		if err != nil {
			return nextMessage, fmt.Errorf("write message: %w", err)
		}
		// clear the next message after successfully writing it
		nextMessage = nil
	}

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("writer context done", zap.String("id", c.id))
			// closing the connection will cause ReadMessage to unblock and return an error
			err := conn.Close()
			if err != nil {
				// log the error but return nil to avoid propagating the error to the caller
				c.logger.Error("error closing connection", zap.Error(err))
			}
			return nextMessage, nil

		case message, ok := <-c.writeChan:
			if !ok {
				// the write channel is closed, so we return
				c.logger.Info("write channel closed", zap.String("id", c.id))
				return message, nil
			}
			err := writeWSMessage(conn, message)
			if err != nil {
				return message, fmt.Errorf("write message: %w", err)
			}
		}
	}
}

// --------------------------------------------------------------------------------------

// Continuously try until connected. Will return nil when successfully
// connected. Will return error if it is cancelled via context.
func (c *upstreamConnection) ensureConnected(ctx context.Context, id string) (*websocket.Conn, error) {
	infiniteBackoff := backoff.NewExponentialBackOff()

	// Make ticker run forever.
	infiniteBackoff.MaxElapsedTime = 0

	interval := time.Duration(0)

	for {
		timer := time.NewTimer(interval)
		interval = infiniteBackoff.NextBackOff()

		select {
		case <-timer.C:
			{
				conn, err := c.tryConnectOnce(ctx, id)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						c.logger.Debug("Client is stopped, will not try anymore.")
						return nil, err
					} else {
						c.logger.Error("Connection failed", zap.Error(err))
					}
					// Retry again a bit later.
					continue
				}
				// Connected successfully.
				return conn, nil
			}

		case <-ctx.Done():
			c.logger.Debug("Client is stopped, will not try anymore.")
			timer.Stop()
			return nil, ctx.Err()
		}
	}
}

func (c *upstreamConnection) tryConnectOnce(ctx context.Context, id string) (*websocket.Conn, error) {
	var resp *http.Response

	conn, resp, err := c.dialer.DialContext(ctx, c.settings.endpoint, c.header(id))
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("server responded with status: %s", resp.Status)
		}
		return nil, err
	}
	c.logger.Info("Successfully connected to upstream OpAMP server", zap.String("id", id), zap.String("upstream_endpoint", conn.RemoteAddr().String()))

	// Successfully connected.
	return conn, nil
}

func (c *upstreamConnection) header(id string) http.Header {
	return http.Header{
		"Authorization":                 []string{fmt.Sprintf("Secret-Key %s", c.settings.secretKey)},
		"X-Opamp-Gateway-Connection-Id": []string{id},
	}
}
