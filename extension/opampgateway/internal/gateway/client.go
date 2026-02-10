package gateway

import (
	"context"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/observiq/bindplane-otel-collector/extension/opampgateway/internal/metadata"
	"go.uber.org/zap"
)

// UpstreamConnectionAssigner assigns and unassigns upstream connections for downstream connections.
type UpstreamConnectionAssigner interface {
	AssignUpstreamConnection(downstreamConnectionID string) (*upstreamConnection, error)
	UnassignUpstreamConnection(downstreamConnectionID string)
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

	telemetry *metadata.TelemetryBuilder
}

func newClient(settings Settings, telemetry *metadata.TelemetryBuilder, callbacks ConnectionCallbacks[*upstreamConnection], logger *zap.Logger) *client {
	logger = logger.Named("client")
	pool := newConnectionPool(settings.UpstreamConnections, logger)
	connections := newConnections[*upstreamConnection]()
	connectionAssignments := newConnectionAssignments(connections, pool)
	return &client{
		logger:                logger,
		dialer:                *websocket.DefaultDialer,
		pool:                  pool,
		upstreamConnections:   connections,
		connectionAssignments: connectionAssignments,
		callbacks:             callbacks,
		secretKey:             settings.SecretKey,
		upstreamEndpoint:      settings.UpstreamOpAMPAddress,
		connectionCount:       settings.UpstreamConnections,
		clientConnectionsWg:   &sync.WaitGroup{},
		telemetry:             telemetry,
	}
}

// Start begins connecting to the upstream OpAMP server. It resets internal
// state so the client can be restarted after a previous Stop (e.g. during
// collector hot-reload).
func (c *client) Start(ctx context.Context) {
	// Reset state so the client can be restarted after a previous Stop.
	c.clientConnectionsWg = &sync.WaitGroup{}
	c.pool = newConnectionPool(c.connectionCount, c.logger)
	c.upstreamConnections = newConnections[*upstreamConnection]()
	c.connectionAssignments = newConnectionAssignments(c.upstreamConnections, c.pool)

	ctx, c.clientConnectionsCancel = context.WithCancel(ctx)
	go c.startClientConnections(ctx)
}

func (c *client) startClientConnections(ctx context.Context) {
	for i := 0; i < c.connectionCount; i++ {
		// generate a unique id for the connection
		id := fmt.Sprintf("upstream-%d", i)

		clientConnection := newUpstreamConnection(c.dialer, c.telemetry, upstreamConnectionSettings{
			endpoint:  c.upstreamEndpoint,
			secretKey: c.secretKey,
		}, id, c.logger)

		c.pool.add(clientConnection)
		c.upstreamConnections.set(id, clientConnection)

		c.clientConnectionsWg.Add(1)
		go func() {
			defer c.clientConnectionsWg.Done()

			c.telemetry.OpampgatewayConnections.Add(context.Background(), 1, directionUpstream)
			defer c.telemetry.OpampgatewayConnections.Add(context.Background(), -1, directionUpstream)

			// cleanup function to remove the connection from the pool and connections map
			defer func() {
				c.upstreamConnections.remove(clientConnection.id)
				c.pool.remove(clientConnection)
				c.logger.Info("upstream connection shutdown", zap.String("id", clientConnection.id), zap.Int("downstream_count", clientConnection.downstreamCount()))
			}()

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
