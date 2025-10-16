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
}

func newOpAMPGateway(logger *zap.Logger, cfg *Config) *OpAMPGateway {
	o := &OpAMPGateway{
		logger:                logger,
		cfg:                   cfg,
		pool:                  newConnectionPool(),
		upstreamConnections:   newConnections(),
		downstreamConnections: newConnections(),
	}
	o.server = newServer(cfg.OpAMPServer, logger.Named("opamp-server"), o)
	return o
}

func (o *OpAMPGateway) Start(_ context.Context, host component.Host) error {
	o.server.Start()
	return nil
}

func (o *OpAMPGateway) Shutdown(ctx context.Context) error {
	o.server.Stop()
	return nil
}

// EnsureDownstreamConnection will ensure that a downstream connection exists for the given agent ID.
func (o *OpAMPGateway) EnsureDownstreamConnection(agentID string, conn *websocket.Conn) {
	_, ok := o.downstreamConnections.get(agentID)
	if !ok {
		o.logger.Debug("Creating downstream connection for agent ID", zap.String("agent_id", agentID))
		o.downstreamConnections.set(agentID, newConnection(conn))
	}
	if ok {
		o.logger.Debug("Downstream connection already exists for agent ID", zap.String("agent_id", agentID))
	}
}

// ForwardMessageUpstream will forward the given message to the upstream connection for the given agent ID.
func (o *OpAMPGateway) ForwardMessageUpstream(agentID string, msg []byte) error {
	conn, err := o.getUpstreamConnection(agentID)
	if err != nil {
		return fmt.Errorf("get upstream connection: %w", err)
	}
	return writeWSMessage(conn, msg)
}

func (o *OpAMPGateway) getUpstreamConnection(agentID string) (*websocket.Conn, error) {
	connID, ok := o.upstreamConnections.get(agentID)
	if !ok {
		// need to assign a new connection - get a random connection ID
		for id := range o.upstreamConnectionsIDs {
			connID = id
			break
		}
		o.upstreamConnections[agentID] = connID
	}
	conn, ok := o.upstreamConnectionsIDs[connID]
	if !ok {
		return nil, fmt.Errorf("upstream connection not found for connection ID: %s", connID)
	}
	return conn, nil
}

func (o *OpAMPGateway) AddUpstreamConnection(conn *websocket.Conn, id string) {
	o.pool.connections
	_, ok := o.upstreamConnections.get()
	if !ok {
		o.upstreamConnectionsIDs[id] = conn
	} else {
		o.logger.Error("upstream connection for this id already exists")
	}
}

func (o *OpAMPGateway) ForwardMessageDownstream(agentID string, msg []byte) error
