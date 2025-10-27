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
	"go.uber.org/zap"
)

type ClientConnectionManagement interface {
	AddUpstreamConnection(ctx context.Context, conn *websocket.Conn, id string, callbacks ConnectionCallbacks)

	ForwardMessageDownstream(ctx context.Context, agentID string, msg []byte) error
}

type client struct {
	logger *zap.Logger
	dialer websocket.Dialer

	pool *connectionPool
	// upstreamConnections is a set of connections to the upstream OpAMP server.
	upstreamConnections *connections

	agentClientConnections *agentClientConnections

	callbacks ConnectionCallbacks

	secretKey        string
	upstreamEndpoint string
	connectionCount  int

	clientConnectionsWg     *sync.WaitGroup
	clientConnectionsCancel context.CancelFunc
}

func newClient(cfg *Config, logger *zap.Logger, callbacks ConnectionCallbacks) *client {
	pool := newConnectionPool(cfg.UpstreamConnections, logger)
	connections := newConnections()
	agentClientConnections := newAgentClientConnections(connections, pool)
	return &client{
		logger:                 logger,
		dialer:                 *websocket.DefaultDialer,
		pool:                   pool,
		upstreamConnections:    connections,
		agentClientConnections: agentClientConnections,
		callbacks:              callbacks,
		secretKey:              cfg.SecretKey,
		upstreamEndpoint:       cfg.UpstreamOpAMPAddress,
		connectionCount:        cfg.UpstreamConnections,
		clientConnectionsWg:    &sync.WaitGroup{},
	}
}

func (c *client) Start(ctx context.Context) {
	ctx, c.clientConnectionsCancel = context.WithCancel(ctx)
	go c.startClientConnections(ctx)
}

func (c *client) startClientConnections(ctx context.Context) {
	for i := 0; i < c.connectionCount; i++ {
		// generate a unique id for the connection
		id := fmt.Sprintf("upstream-%d", i)

		// ensure connected will do infinite retries until the context is done or an error is
		// returned. this could mean that it takes a while to open the connections to the
		// upstream server.
		conn, err := c.ensureConnected(ctx, id)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("ensure connected", zap.Error(err))
			return
		}

		clientConnection := newConnection(conn, id, c.logger.Named("upstream-connection"))
		c.pool.add(clientConnection)
		c.upstreamConnections.set(id, clientConnection)

		c.clientConnectionsWg.Add(1)
		go clientConnection.start(ctx, ConnectionCallbacks{
			OnMessage: c.callbacks.OnMessage,
			OnError:   c.callbacks.OnError,
			OnClose: func(ctx context.Context, connection *connection) error {
				defer c.clientConnectionsWg.Done()
				c.upstreamConnections.remove(connection.id)
				// TODO: replace with a new connection
				c.pool.remove(connection)
				c.logger.Info("upstream connection closed", zap.String("id", connection.id), zap.Int("connection_count", c.pool.size()))
				return c.callbacks.OnClose(ctx, connection)
			},
		})
	}
}

func (c *client) Stop() {
	if c.clientConnectionsCancel != nil {
		c.clientConnectionsCancel()
	}
	c.clientConnectionsWg.Wait()
	c.logger.Info("client stopped")
}

// --------------------------------------------------------------------------------------
// upstream connection management

func (c *client) assignedUpstreamConnection(agentID string) (*connection, error) {
	conn, exists := c.agentClientConnections.assignedAgentConnection(agentID)
	if !exists {
		return nil, fmt.Errorf("no upstream connection available for agent %s", agentID)
	}
	return conn, nil
}

func (c *client) unassignUpstreamConnection(agentID string) {
	c.agentClientConnections.unassignAgentConnection(agentID)
}

// --------------------------------------------------------------------------------------

// Continuously try until connected. Will return nil when successfully
// connected. Will return error if it is cancelled via context.
func (c *client) ensureConnected(ctx context.Context, id string) (*websocket.Conn, error) {
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

func (c *client) tryConnectOnce(ctx context.Context, id string) (*websocket.Conn, error) {
	var resp *http.Response

	conn, resp, err := c.dialer.DialContext(ctx, c.upstreamEndpoint, c.header(id))
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

func (c *client) header(id string) http.Header {
	return http.Header{
		"Authorization":                 []string{fmt.Sprintf("Secret-Key %s", c.secretKey)},
		"X-Opamp-Gateway-Connection-Id": []string{id},
	}
}
