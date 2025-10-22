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

	// assignedConnections is a map of agent ID to the websocket connection id that
	// AgentToServer messages should be forwarded on.
	assignedConnections    map[string]string
	assignedConnectionsMtx sync.RWMutex

	callbacks ConnectionCallbacks

	secretKey        string
	upstreamEndpoint string
	connectionCount  int

	clientConnectionsWg     *sync.WaitGroup
	clientConnectionsCancel context.CancelFunc
}

func newClient(cfg *Config, logger *zap.Logger, callbacks ConnectionCallbacks) *client {
	return &client{
		logger:              logger,
		dialer:              *websocket.DefaultDialer,
		pool:                newConnectionPool(cfg.UpstreamConnections, logger),
		upstreamConnections: newConnections(),
		assignedConnections: make(map[string]string),
		callbacks:           callbacks,
		secretKey:           cfg.SecretKey,
		upstreamEndpoint:    cfg.UpstreamOpAMPAddress,
		connectionCount:     cfg.UpstreamConnections,
	}
}

func (c *client) Start(ctx context.Context) {
	ctx, c.clientConnectionsCancel = context.WithCancel(ctx)

	c.clientConnectionsWg = &sync.WaitGroup{}
	for i := 0; i < c.connectionCount; i++ {
		// ensure connected will do infinite retries until the context is done or an error is returned.
		conn, err := c.ensureConnected(ctx)
		if err != nil {
			c.logger.Error("ensure connected", zap.Error(err))
			return
		}

		clientConnection := newConnection(conn, fmt.Sprintf("%s_%d", conn.RemoteAddr().String(), i), c.logger.Named("upstream-connection"))
		c.pool.add(clientConnection)

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
	c.logger.Info("stopping client")
	c.clientConnectionsCancel()
	c.clientConnectionsWg.Wait()
	c.logger.Info("client stopped")
}

// --------------------------------------------------------------------------------------
// upstream connection management

func (c *client) assignedUpstreamConnection(agentID string) (*connection, error) {
	// check for an existing assignment
	c.assignedConnectionsMtx.RLock()
	connectionID, ok := c.assignedConnections[agentID]
	c.assignedConnectionsMtx.RUnlock()

	if ok {
		// make sure the connection still exists
		if conn, ok := c.upstreamConnections.get(connectionID); ok {
			return conn, nil
		}
	}

	// if no existing assignment or missing connection, assign a new connection
	c.assignedConnectionsMtx.Lock()
	defer c.assignedConnectionsMtx.Unlock()

	// get a connection from the pool
	conn := c.pool.next()

	// assign the connection to the agent ID
	c.assignedConnections[agentID] = conn.id

	return conn, nil
}

func (c *client) unassignUpstreamConnection(agentID string) {
	c.assignedConnectionsMtx.Lock()
	defer c.assignedConnectionsMtx.Unlock()
	delete(c.assignedConnections, agentID)
}

// --------------------------------------------------------------------------------------

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
