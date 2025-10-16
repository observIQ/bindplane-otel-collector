package opampgateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gorilla/websocket"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
)

type ClientConnectionManagement interface {
	AddUpstreamConnection(conn *websocket.Conn, id string)

	ForwardMessageDownstream(ctx context.Context, agentID string, msg []byte) error
}

type client struct {
	logger *zap.Logger
	dialer websocket.Dialer

	ccm ClientConnectionManagement

	secretKey        string
	upstreamEndpoint string
	connectionCount  int

	clientConnectionsWg     *sync.WaitGroup
	clientConnectionsCancel context.CancelFunc
}

func newClient(cfg *Config, logger *zap.Logger, ccm ClientConnectionManagement) *client {
	return &client{
		logger:           logger,
		dialer:           *websocket.DefaultDialer,
		ccm:              ccm,
		secretKey:        cfg.SecretKey,
		upstreamEndpoint: cfg.UpstreamOpAMPAddress,
		connectionCount:  cfg.UpstreamConnections,
	}
}

func (c *client) Start() error {
	for i := 0; i < c.connectionCount; i++ {
		conn, err := c.ensureConnected(context.Background())
		if err != nil {
			return fmt.Errorf("ensure connected: %w", err)
		}

		c.ccm.AddUpstreamConnection(conn, fmt.Sprintf("%s_%d", conn.RemoteAddr().String(), i))

		connCtx, connCancel := context.WithCancel(context.Background())
		c.clientConnectionsCancel = connCancel

		c.clientConnectionsWg = &sync.WaitGroup{}
		c.clientConnectionsWg.Add(1)
		go c.handleWSConnection(connCtx, conn)
	}

	return nil
}

func (c *client) Stop() error {
	c.clientConnectionsCancel()
	c.clientConnectionsWg.Wait()

	return nil
}

func (c *client) handleWSConnection(ctx context.Context, conn *websocket.Conn) {
	defer c.clientConnectionsWg.Done()

	type receivedMessage struct {
		message []byte
		err     error
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			result := make(chan receivedMessage, 1)

			go func() {
				_, bytes, err := conn.ReadMessage()
				result <- receivedMessage{bytes, err}
			}()

			select {
			case <-ctx.Done():
				return
			case res := <-result:
				messageCtx := context.Background()

				if res.err == nil {
					c.logger.Error("Failed to read message from websocket", zap.Error(res.err))
					continue
				}
				message := protobufs.ServerToAgent{}
				if err := decodeWSMessage(res.message, &message); err != nil {
					c.logger.Error("failed to decode ws message", zap.Error(err))
					continue
				}

				agentID, err := parseAgentID(message.GetInstanceUid())
				if err != nil {
					c.logger.Error("failed to parse agent id", zap.Error(err))
					continue
				}

				err = c.ccm.ForwardMessageDownstream(messageCtx, agentID, res.message)
				if err != nil {
					c.logger.Error("failed to forward message downstream")
				}
			}
		}
	}
}

// Continuously try until connected. Will return nil when successfully
// connected. Will return error if it is cancelled via context.
func (c *client) ensureConnected(ctx context.Context) (*websocket.Conn, error) {
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
				conn, err := c.tryConnectOnce(ctx)
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

func (c *client) tryConnectOnce(ctx context.Context) (*websocket.Conn, error) {
	var resp *http.Response

	conn, resp, err := c.dialer.DialContext(ctx, c.upstreamEndpoint, c.header())
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("server responded with status: %s", resp.Status)
		}
		return nil, err
	}
	c.logger.Info("Successfully connected to upstream OpAMP server", zap.String("upstream_endpoint", conn.RemoteAddr().String()))

	// Successfully connected.
	return conn, nil
}

func (c *client) header() http.Header {
	return http.Header{
		"Authorization": []string{fmt.Sprintf("Secret-Key %s", c.secretKey)},
	}
}
