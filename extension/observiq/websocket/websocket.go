package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/observiq/observiq-collector/extension/observiq/message"
	"golang.org/x/sync/errgroup"
)

// Conn is a websocket connection.
type Conn = websocket.Conn

// ErrConnectionClose is the error returned during a normal connection closure.
var ErrConnectionClosed = errors.New("connection closed")

// Open will open a new websocket connection.
func Open(ctx context.Context, url string, headers http.Header) (*Conn, error) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}
	return conn, err
}

// Close will attempt to gracefully close a websocket connection.
func Close(conn *Conn) {
	deadline := time.Now().Add(time.Second)
	message := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")

	err := conn.WriteControl(websocket.CloseMessage, message, deadline)
	if err != nil {
		conn.Close()
	}
}

// PumpInbound will pump inbound traffic from the connection to the pipeline.
func PumpInbound(ctx context.Context, conn *Conn, pipeline *message.Pipeline) error {
	defer Close(conn)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			message, err := read(conn)
			if err != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return fmt.Errorf("unknown read error: %w", err)
			}

			if err != nil {
				return ErrConnectionClosed
			}

			pipeline.Inbound() <- message
		}
	}
}

// PumpOutbound will pump outbound traffic from the pipeline to the connection.
func PumpOutbound(ctx context.Context, conn *Conn, pipeline *message.Pipeline) error {
	defer Close(conn)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case message := <-pipeline.Outbound():
			if err := write(message, conn); err != nil {
				return fmt.Errorf("unknown write error: %w", err)
			}
		}
	}
}

// Pump will handle pumping a connection's traffic.
func Pump(ctx context.Context, conn *Conn, pipeline *message.Pipeline) error {
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error { return PumpInbound(groupCtx, conn, pipeline) })
	group.Go(func() error { return PumpOutbound(groupCtx, conn, pipeline) })
	return group.Wait()
}

// PumpWithTimeout will pump a connection's traffic until a timeout occurs.
func PumpWithTimeout(ctx context.Context, conn *websocket.Conn, pipeline *message.Pipeline, timeout time.Duration) error {
	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return Pump(timedCtx, conn, pipeline)
}

// read will read the next message from the supplied connection.
func read(conn *Conn) (*message.Message, error) {
	_, reader, err := conn.NextReader()
	if err != nil {
		return nil, err
	}

	message := &message.Message{}
	err = json.NewDecoder(reader).Decode(message)
	return message, err
}

// write will write a message to the supplied connection.
func write(m *message.Message, conn *Conn) error {
	bytes, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, bytes)
}
