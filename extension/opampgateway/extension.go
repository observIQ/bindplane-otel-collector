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
	// upstreamConnections is a map of agent ID to the websocket ID of the connection AgentToServer messages should be forwarded on.
	upstreamConnections    map[string]string
	upstreamConnectionsIDs map[string]*websocket.Conn
	// downstreamConnections is a map of agent ID to the websocket connection ServerToAgent messages should be forwarded on.
	downstreamConnections map[string]*websocket.Conn
}

func newOpAMPGateway(logger *zap.Logger, cfg *Config) *OpAMPGateway {
	return &OpAMPGateway{
		logger: logger,
		cfg:    cfg,
	}
}

func (o *OpAMPGateway) Start(_ context.Context, host component.Host) error {
	return nil
}

func (o *OpAMPGateway) Shutdown(ctx context.Context) error {
	return nil
}

// EnsureDownstreamConnection will ensure that a downstream connection exists for the given agent ID.
func (o *OpAMPGateway) EnsureDownstreamConnection(agentID string, conn *websocket.Conn) {
	_, ok := o.downstreamConnections[agentID]
	if !ok {
		o.logger.Debug("Creating downstream connection for agent ID", zap.String("agent_id", agentID))
		o.downstreamConnections[agentID] = conn
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
	connID, ok := o.upstreamConnections[agentID]
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
