package opampgateway

import (
	"context"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type UpstreamConnectionAssigner interface {
	AssignUpstreamConnection(downstreamConnectionID string) (*upstreamConnection, error)
}

type client struct {
	logger *zap.Logger
	dialer websocket.Dialer

	pool *connectionPool
	// upstreamConnections is a set of connections to the upstream OpAMP server.
	upstreamConnections *connections[*upstreamConnection]

	connectionAssignments *connectionAssignments

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
	connectionAssignments := newConnectionAssignments(connections, pool)
	return &client{
		logger:                logger.Named("client"),
		dialer:                *websocket.DefaultDialer,
		pool:                  pool,
		upstreamConnections:   connections,
		connectionAssignments: connectionAssignments,
		callbacks:             callbacks,
		secretKey:             cfg.SecretKey,
		upstreamEndpoint:      cfg.UpstreamOpAMPAddress,
		connectionCount:       cfg.UpstreamConnections,
		clientConnectionsWg:   &sync.WaitGroup{},
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
		}, id, c.logger)

		c.pool.add(clientConnection)
		c.upstreamConnections.set(id, clientConnection)

		go func() {
			// cleanup function to remove the connection from the pool and connections map
			defer func() {
				defer c.clientConnectionsWg.Done()
				c.upstreamConnections.remove(clientConnection.id)
				c.pool.remove(clientConnection)
				c.logger.Info("upstream connection shutdown", zap.String("id", clientConnection.id), zap.Int("downstream_count", clientConnection.downstreamCount()))
			}()
			c.clientConnectionsWg.Add(1)

			// start the connection
			clientConnection.start(ctx, ConnectionCallbacks[*upstreamConnection]{
				OnMessage: c.callbacks.OnMessage,
				OnError:   c.callbacks.OnError,
				OnClose: func(ctx context.Context, connection *upstreamConnection) error {
					c.logger.Info("upstream connection closed", zap.String("id", connection.id), zap.Int("downstream_count", clientConnection.downstreamCount()))
					return c.callbacks.OnClose(ctx, connection)
				},
			})
		}()
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

func (c *client) assignedUpstreamConnection(downstreamConnectionID string) (*upstreamConnection, error) {
	conn, exists := c.connectionAssignments.assignedUpstreamConnection(downstreamConnectionID)
	if !exists {
		c.logger.Info("no upstream connection available", zap.String("downstream_connection_id", downstreamConnectionID), zap.Int("connection_count", c.pool.size()))
		return nil, fmt.Errorf("no upstream connection available for downstream connection %s: %w", downstreamConnectionID, ErrNoUpstreamConnectionsAvailable)
	}
	c.logger.Info("assigned upstream connection", zap.String("downstream_connection_id", downstreamConnectionID), zap.String("upstream_connection_id", conn.id))
	return conn, nil
}

func (c *client) unassignUpstreamConnection(downstreamConnectionID string) {
	c.connectionAssignments.unassignDownstreamConnection(downstreamConnectionID)
}

// --------------------------------------------------------------------------------------
// UpstreamConnectionAssigner

func (c *client) AssignUpstreamConnection(downstreamConnectionID string) (*upstreamConnection, error) {
	return c.assignedUpstreamConnection(downstreamConnectionID)
}

func (c *client) UnassignUpstreamConnection(downstreamConnectionID string) {
	c.unassignUpstreamConnection(downstreamConnectionID)
}
