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
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/observiq/bindplane-otel-collector/extension/opampgateway/internal/metadata"
	"go.uber.org/zap"
)

type downstreamConnection struct {
	id   string
	conn *websocket.Conn

	writeChan       chan *message
	readerErrorChan chan error

	readerDone chan struct{}
	writerDone chan struct{}

	upstreamConnection *upstreamConnection

	telemetry *metadata.TelemetryBuilder
	logger    *zap.Logger

	cancel context.CancelFunc
}

func newDownstreamConnection(conn *websocket.Conn, telemetry *metadata.TelemetryBuilder, upstreamConnection *upstreamConnection, id string, logger *zap.Logger) *downstreamConnection {
	return &downstreamConnection{
		conn:               conn,
		upstreamConnection: upstreamConnection,
		id:                 id,
		telemetry:          telemetry,
		logger:             logger.Named("downstream-connection").With(zap.String("id", id)),
		writeChan:          make(chan *message),

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
	ctx, c.cancel = context.WithCancel(ctx)
	defer c.cancel()

	// start the reader in a separate goroutine and cancel the context if it returns, likely
	// due to an error or the connection being closed
	go func() {
		defer c.cancel()
		c.startReader(ctx, callbacks)
	}()

	// block while writing messages to the connection. a connection close will unblock the writer.
	err := c.startWriter(ctx)
	if err != nil {
		c.logger.Error("error in connection writer", zap.Error(err))
		callbacks.OnError(ctx, c, err)
	}

	// if the writer returns, cancel the context to stop the reader
	c.cancel()

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
func (c *downstreamConnection) send(message *message) {
	c.logger.Debug("sending message", zap.String("message", string(message.data)))
	c.writeChan <- message
}

func (c *downstreamConnection) close() error {
	c.logger.Info("downstream connection closing")
	c.cancel()
	return nil
}

// --------------------------------------------------------------------------------------
// reader goroutine

func (c *downstreamConnection) startReader(ctx context.Context, callbacks ConnectionCallbacks[*downstreamConnection]) {
	defer close(c.readerDone)

	reader := newMessageReader(c.conn, c.id, readerCallbacks{
		OnMessage: func(ctx context.Context, messageType int, message *message) error {
			return callbacks.OnMessage(ctx, c, messageType, message)
		},
		OnError: func(ctx context.Context, err error) {
			callbacks.OnError(ctx, c, err)
		},
	}, c.logger)

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
			// Send a WebSocket close frame to notify the peer before closing the TCP connection.
			closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			_ = c.conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(5*time.Second))
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
			err := writeWSMessage(c.conn, message.data)
			if err != nil {
				return fmt.Errorf("write message: %w", err)
			}
			c.telemetry.OpampgatewayMessagesLatency.Record(ctx, message.elapsedTime().Milliseconds(), directionDownstream)
		}
	}
}
