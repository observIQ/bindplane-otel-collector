package opampgateway

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/observiq/bindplane-otel-collector/extension/opampgateway/internal/metadata"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

var (
	attrUpstream   = attribute.String("direction", "upstream")
	attrDownstream = attribute.String("direction", "downstream")

	directionUpstream   = metric.WithAttributeSet(attribute.NewSet(attrUpstream))
	directionDownstream = metric.WithAttributeSet(attribute.NewSet(attrDownstream))
)

type OpAMPGateway struct {
	logger *zap.Logger
	cfg    *Config

	server *server

	client *client

	telemetry *metadata.TelemetryBuilder
}

func newOpAMPGateway(logger *zap.Logger, cfg *Config, t *metadata.TelemetryBuilder) *OpAMPGateway {
	logger = logger.Named("opamp-gateway")
	o := &OpAMPGateway{
		logger:    logger,
		cfg:       cfg,
		telemetry: t,
	}
	o.client = newClient(cfg, o.telemetry, ConnectionCallbacks[*upstreamConnection]{
		OnMessage: o.HandleUpstreamMessage,
		OnError:   o.HandleUpstreamError,
		OnClose:   o.HandleUpstreamClose,
	}, logger)
	o.server = newServer(cfg.OpAMPServer, o.telemetry, o.client, ConnectionCallbacks[*downstreamConnection]{
		OnMessage: o.HandleDownstreamMessage,
		OnError:   o.HandleDownstreamError,
		OnClose:   o.HandleDownstreamClose,
	}, logger)
	return o
}

func (o *OpAMPGateway) Start(ctx context.Context, host component.Host) error {
	o.client.Start(ctx)
	return o.server.Start()
}

func (o *OpAMPGateway) Shutdown(ctx context.Context) error {
	o.client.Stop()
	return o.server.Stop()
}

// --------------------------------------------------------------------------------------
// Downstream callbacks

// HandleDownstreamMessage handles message set from a downstream connection to the server
func (o *OpAMPGateway) HandleDownstreamMessage(ctx context.Context, connection *downstreamConnection, messageType int, msg *message) error {
	o.logger.Debug("HandleDownstreamMessage", zap.String("downstream_connection_id", connection.id), zap.Int("message_number", msg.number), zap.Int("message_type", messageType))
	if messageType != websocket.BinaryMessage {
		err := fmt.Errorf("unexpected message type: %v, must be binary message", messageType)
		o.logger.Error("Cannot process a message from WebSocket", zap.Error(err), zap.Int("message_number", msg.number), zap.Int("message_type", messageType), zap.String("message_bytes", string(msg.data)))
		return err
	}

	message := protobufs.AgentToServer{}
	err := decodeWSMessage(msg.data, &message)
	if err != nil {
		return fmt.Errorf("cannot decode message from WebSocket: %w", err)
	}

	agentID, err := parseAgentID(message.GetInstanceUid())
	if err != nil {
		return fmt.Errorf("cannot parse agent ID: %w", err)
	}

	// register the downstream connection for the agent ID so we can forward messages to the
	// agent from the server
	o.server.setAgentConnection(agentID, connection)

	// find the upstream connection
	upstreamConnection := connection.upstreamConnection

	// send the message to the upstream connection
	logMsg := fmt.Sprintf("%s => %s", connection.id, upstreamConnection.id)
	logUpstreamMessage(o.logger, logMsg, agentID, msg.number, len(msg.data), &message)
	upstreamConnection.send(msg)
	o.telemetry.OpampgatewayMessages.Add(context.Background(), 1, directionUpstream)
	o.telemetry.OpampgatewayMessageBytes.Add(context.Background(), int64(len(msg.data)), directionUpstream)
	return nil
}

func (o *OpAMPGateway) HandleDownstreamError(ctx context.Context, connection *downstreamConnection, err error) {
	o.logger.Error("HandleDownstreamError", zap.Error(err))
}

func (o *OpAMPGateway) HandleDownstreamClose(ctx context.Context, connection *downstreamConnection) error {
	o.logger.Info("HandleDownstreamClose", zap.String("downstream_connection_id", connection.id))
	o.client.unassignUpstreamConnection(connection.id)
	o.server.removeDownstreamConnection(connection)
	return nil
}

// --------------------------------------------------------------------------------------
// Upstream callbacks

// HandleUpstreamMessage handles message set from the upstream connection to a downstream connection
func (o *OpAMPGateway) HandleUpstreamMessage(ctx context.Context, connection *upstreamConnection, messageType int, message *message) error {
	o.logger.Debug("HandleUpstreamMessage", zap.String("upstream_connection_id", connection.id), zap.Int("message_number", message.number), zap.Int("message_type", messageType), zap.String("message_bytes", string(message.data)))
	if messageType != websocket.BinaryMessage {
		err := fmt.Errorf("unexpected message type: %v, must be binary message", messageType)
		o.logger.Error("Cannot process a message from WebSocket", zap.Error(err), zap.Int("message_number", message.number), zap.Int("message_type", messageType), zap.String("message_bytes", string(message.data)))
		return err
	}

	m := protobufs.ServerToAgent{}
	if err := decodeWSMessage(message.data, &m); err != nil {
		return fmt.Errorf("failed to decode ws message: %w", err)
	}

	agentID, err := parseAgentID(m.GetInstanceUid())
	if err != nil {
		return fmt.Errorf("parse agent id: %w, %s", err, m.String())
	}

	// find the downstream connection from the server
	conn, ok := o.server.getAgentConnection(agentID)
	if !ok {
		// downstream connection no longer exists. just ignore the message for now.
		return nil
	}

	// forward the message to the downstream connection
	msg := fmt.Sprintf("%s <= %s", conn.id, connection.id)
	logDownstreamMessage(o.logger, msg, agentID, message.number, len(message.data), &m)
	conn.send(message)
	o.telemetry.OpampgatewayMessages.Add(context.Background(), 1, directionDownstream)
	o.telemetry.OpampgatewayMessageBytes.Add(context.Background(), int64(len(message.data)), directionDownstream)
	return nil
}

func (o *OpAMPGateway) HandleUpstreamError(ctx context.Context, connection *upstreamConnection, err error) {
	o.logger.Error("HandleUpstreamError", zap.Error(err))
}

func (o *OpAMPGateway) HandleUpstreamClose(ctx context.Context, connection *upstreamConnection) error {
	o.logger.Info("HandleUpstreamClose", zap.String("upstream_connection_id", connection.id))
	// close all downstream connections associated with this upstream connection
	downstreamConnectionIDs := o.client.connectionAssignments.removeDownstreamConnectionIDs(connection.id)
	o.server.closeDownstreamConnections(downstreamConnectionIDs)
	return nil
}

// --------------------------------------------------------------------------------------
// logging helpers

func logDownstreamMessage(logger *zap.Logger, msg string, agentID string, messageNumber int, messageBytes int, message *protobufs.ServerToAgent) {
	logger.Info(msg, zap.String("agent.id", agentID), zap.Int("message.number", messageNumber), zap.Int("message.bytes", messageBytes), zap.Strings("components", downstreamMessageComponents(message)))
}

func logUpstreamMessage(logger *zap.Logger, msg string, agentID string, messageNumber int, messageBytes int, message *protobufs.AgentToServer) {
	logger.Info(msg, zap.String("agent.id", agentID), zap.Int("message.number", messageNumber), zap.Int("message.bytes", messageBytes), zap.Strings("components", upstreamMessageComponents(message)))
}

func downstreamMessageComponents(serverToAgent *protobufs.ServerToAgent) []string {
	var components []string
	components = includeComponent(components, serverToAgent.ErrorResponse, "ErrorResponse")
	components = includeComponent(components, serverToAgent.RemoteConfig, "RemoteConfig")
	components = includeComponent(components, serverToAgent.ConnectionSettings, "ConnectionSettings")
	components = includeComponent(components, serverToAgent.PackagesAvailable, "PackagesAvailable")
	components = includeComponent(components, serverToAgent.AgentIdentification, "AgentIdentification")
	components = includeComponent(components, serverToAgent.Command, "Command")
	components = includeCustomMessage(components, serverToAgent.CustomMessage)
	return components
}

// upstreamMessageComponents returns the names of the components in the message
func upstreamMessageComponents(agentToServer *protobufs.AgentToServer) []string {
	var components []string
	components = includeComponent(components, agentToServer.AgentDescription, "AgentDescription")
	components = includeComponent(components, agentToServer.EffectiveConfig, "EffectiveConfig")
	components = includeComponent(components, agentToServer.RemoteConfigStatus, "RemoteConfigStatus")
	components = includeComponent(components, agentToServer.PackageStatuses, "PackageStatuses")
	components = includeComponent(components, agentToServer.AvailableComponents, "AvailableComponents")
	components = includeComponent(components, agentToServer.Health, "Health")
	components = includeCustomMessage(components, agentToServer.CustomMessage)
	return components
}

func includeComponent[T any](components []string, msg *T, name string) []string {
	if msg != nil {
		components = append(components, name)
	}
	return components
}

func includeCustomMessage(components []string, msg *protobufs.CustomMessage) []string {
	if msg != nil {
		components = append(components, msg.Type)
	}
	return components
}
