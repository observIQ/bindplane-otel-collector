package websocket

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var ErrConnectionClosed = errors.New("connection closed")

// Client is a websocket client.
type Client struct {
	url      string
	headers  http.Header
	inbound  chan []byte
	outbound chan []byte
}

// ClientConfig is the config of a websocket client.
type ClientConfig struct {
	URL          string
	Headers      http.Header
	InboundSize  int
	OutboundSize int
}

// NewClient returns a new websocket client.
func NewClient(config ClientConfig) *Client {
	return &Client{
		url:      config.URL,
		headers:  config.Headers,
		inbound:  make(chan []byte, config.InboundSize),
		outbound: make(chan []byte, config.OutboundSize),
	}
}

// Open will open a new websocket connection.
func (c *Client) Open(ctx context.Context) (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.url, c.headers)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}
	return conn, err
}

// Close will attempt to gracefully close the supplied connection
func (c *Client) Close(conn *websocket.Conn) error {
	deadline := time.Now().Add(time.Second)
	message := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")

	err := conn.WriteControl(websocket.CloseMessage, message, deadline)
	if err != nil {
		err = conn.Close()
	}

	return err
}

// Send returns the client's outbound channel.
func (c *Client) Outbound() chan []byte {
	return c.outbound
}

// Receive returns the client's inbound channel.
func (c *Client) Inbound() chan []byte {
	return c.inbound
}

// HandleInbound will handle reading inbound traffic from the supplied connection.
func (c *Client) HandleInbound(ctx context.Context, conn *websocket.Conn) error {
	defer c.Close(conn)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			message, err := c.read(conn)
			if err != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return fmt.Errorf("unknown read error: %w", err)
			}

			if err != nil {
				return ErrConnectionClosed
			}

			c.inbound <- message
		}
	}
}

// HandleOutbound will handle writing outbound traffic to the supplied connection.
func (c *Client) HandleOutbound(ctx context.Context, conn *websocket.Conn) error {
	defer c.Close(conn)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case message := <-c.outbound:
			if err := c.write(message, conn); err != nil {
				return fmt.Errorf("unknown write error: %w", err)
			}
		}
	}
}

// read will read the next message from the supplied connection.
func (c *Client) read(conn *websocket.Conn) ([]byte, error) {
	_, reader, err := conn.NextReader()
	if err != nil {
		return nil, err
	}

	buffer := new(bytes.Buffer)
	buffer.ReadFrom(reader)
	return buffer.Bytes(), nil
}

// write will write a message to the supplied connection.
func (c *Client) write(message []byte, conn *websocket.Conn) error {
	return conn.WriteMessage(websocket.TextMessage, message)
}
