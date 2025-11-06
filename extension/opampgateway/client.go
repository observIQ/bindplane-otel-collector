package opampgateway

import (
	"context"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type ClientConnectionManagement interface {
	AddUpstreamConnection(ctx context.Context, conn *websocket.Conn, id string, callbacks ConnectionCallbacks[*upstreamConnection])

	ForwardMessageDownstream(ctx context.Context, agentID string, msg []byte) error
}

type client struct {
	logger *zap.Logger
	dialer websocket.Dialer

	pool *connectionPool
	// upstreamConnections is a set of connections to the upstream OpAMP server.
	upstreamConnections *connections[*upstreamConnection]

	agentClientConnections *agentClientConnections

	callbacks ConnectionCallbacks[*upstreamConnection]

	secretKey        string
	upstreamEndpoint string
	connectionCount  int

	clientConnectionsWg     *sync.WaitGroup
	clientConnectionsCancel context.CancelFunc
}

func newClient(cfg *Config, logger *zap.Logger, callbacks ConnectionCallbacks[*upstreamConnection]) *client {
	pool := newConnectionPool(cfg.UpstreamConnections, logger)
	connections := newConnections[*upstreamConnection]()
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

		clientConnection := newUpstreamConnection(c.dialer, upstreamConnectionSettings{
			endpoint:  c.upstreamEndpoint,
			secretKey: c.secretKey,
		}, id, c.logger.Named("upstream-connection"))

		c.pool.add(clientConnection)
		c.upstreamConnections.set(id, clientConnection)

		c.clientConnectionsWg.Add(1)
		go clientConnection.start(ctx, ConnectionCallbacks[*upstreamConnection]{
			OnMessage: c.callbacks.OnMessage,
			OnError:   c.callbacks.OnError,
			OnClose: func(ctx context.Context, connection *upstreamConnection) error {
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

func (c *client) assignedUpstreamConnection(agentID string) (*upstreamConnection, error) {
	conn, exists := c.agentClientConnections.assignedAgentConnection(agentID)
	if !exists {
		c.logger.Info("no upstream connection available", zap.String("agent_id", agentID), zap.Int("connection_count", c.pool.size()))
		return nil, fmt.Errorf("no upstream connection available for agent %s: %w", agentID, ErrNoUpstreamConnectionsAvailable)
	}
	return conn, nil
}

func (c *client) unassignUpstreamConnection(agentID string) {
	c.agentClientConnections.unassignAgentConnection(agentID)
}
