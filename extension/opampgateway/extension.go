package opampgateway

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type OpAMPGateway struct {
	logger *zap.Logger
	cfg    *Config

	pool *connectionPool

	// upstreamConnections is a map of agent ID to the websocket connection AgentToServer messages should be forwarded on.
	upstreamConnections *connections

	// downstreamConnections is a map of agent ID to the websocket connection ServerToAgent messages should be forwarded on.
	downstreamConnections *connections

	server *server

	client *client
}

func newOpAMPGateway(logger *zap.Logger, cfg *Config) *OpAMPGateway {
	o := &OpAMPGateway{
		logger:                logger,
		cfg:                   cfg,
		pool:                  newConnectionPool(logger),
		upstreamConnections:   newConnections(),
		downstreamConnections: newConnections(),
	}
	o.server = newServer(cfg.OpAMPServer, logger.Named("opamp-server"), o)
	o.client = newClient(cfg, logger.Named("opamp-client"), o)
	return o
}

func (o *OpAMPGateway) Start(_ context.Context, host component.Host) error {
	o.server.Start()
	o.client.Start(context.Background())
	return nil
}

func (o *OpAMPGateway) Shutdown(ctx context.Context) error {
	o.server.Stop()
	o.client.Stop()
	return nil
}

// EnsureDownstreamConnection will ensure that a downstream connection exists for the given agent ID.
func (o *OpAMPGateway) EnsureDownstreamConnection(agentID string, conn *websocket.Conn) {
	_, ok := o.downstreamConnections.get(agentID)
	if !ok {
		o.logger.Debug("Creating downstream connection for agent ID", zap.String("agent_id", agentID))
		o.downstreamConnections.set(agentID, newConnection(conn, o.logger.Named("downstream-connection")))
	}
	if ok {
		o.logger.Debug("Downstream connection already exists for agent ID", zap.String("agent_id", agentID))
	}
}

// ForwardMessageUpstream will forward the given message to the upstream connection for the given agent ID.
func (o *OpAMPGateway) ForwardMessageUpstream(ctx context.Context, agentID string, msg []byte) error {
	o.logger.Info("Forwarding message upstream", zap.String("agent_id", agentID), zap.String("message", string(msg)))
	conn, err := o.getUpstreamConnection(agentID)
	if err != nil {
		return fmt.Errorf("get upstream connection: %w", err)
	}
	return conn.Send(ctx, msg)
}

func (o *OpAMPGateway) getUpstreamConnection(agentID string) (*wsConnection, error) {
	c, ok := o.upstreamConnections.get(agentID)
	if !ok {
		c = o.pool.next()
		o.upstreamConnections.set(agentID, c)
	}
	return c.conn, nil
}

func (o *OpAMPGateway) AddUpstreamConnection(conn *websocket.Conn, id string) {
	c := newConnection(conn, o.logger.Named("upstream-connection"))
	c.id = id
	o.pool.add(c)
}

func (o *OpAMPGateway) ForwardMessageDownstream(ctx context.Context, agentID string, msg []byte) error {
	c, ok := o.downstreamConnections.get(agentID)
	if !ok {
		return fmt.Errorf("downstream connection not found for id '%s'", agentID)
	}
	return c.conn.Send(ctx, msg)
}
