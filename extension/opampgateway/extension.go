package opampgateway

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/observiq/bindplane-otel-collector/extension/opampgateway/internal/metadata"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type OpAMPGateway struct {
	logger *zap.Logger
	cfg    *Config

	server *server

	client *client

	telemetry *metadata.TelemetryBuilder
}

func newOpAMPGateway(logger *zap.Logger, cfg *Config, t *metadata.TelemetryBuilder) *OpAMPGateway {
	o := &OpAMPGateway{
		logger:    logger,
		cfg:       cfg,
		telemetry: t,
	}
	o.server = newServer(cfg.OpAMPServer, logger.Named("opamp-server"), ConnectionCallbacks[*downstreamConnection]{
		OnMessage: o.HandleAgentMessage,
		OnError:   o.HandleAgentError,
		OnClose:   o.HandleAgentClose,
	})
	o.client = newClient(cfg, logger.Named("opamp-client"), ConnectionCallbacks[*upstreamConnection]{
		OnMessage: o.HandleServerMessage,
		OnError:   o.HandleServerError,
		OnClose:   o.HandleServerClose,
	})
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
// Agent callbacks

// HandleAgentMessage handles message set from an agent to the server
func (o *OpAMPGateway) HandleAgentMessage(ctx context.Context, connection *downstreamConnection, messageNumber int, messageType int, messageBytes []byte) error {
	if messageType != websocket.BinaryMessage {
		err := fmt.Errorf("unexpected message type: %v, must be binary message", messageType)
		o.logger.Error("Cannot process a message from WebSocket", zap.Error(err), zap.Int("message_number", messageNumber), zap.Int("message_type", messageType), zap.String("message_bytes", string(messageBytes)))
		return err
	}

	message := protobufs.AgentToServer{}
	err := decodeWSMessage(messageBytes, &message)
	if err != nil {
		return fmt.Errorf("cannot decode message from WebSocket: %w", err)
	}

	agentID, err := parseAgentID(message.GetInstanceUid())
	if err != nil {
		return fmt.Errorf("cannot parse agent ID: %w", err)
	}

	if messageNumber == 0 {
		// this is the first message, so we need to register the downstream connection for the
		// agent ID so we can forward messages to the agent from the server
		o.server.addDownstreamConnection(agentID, connection)
		o.telemetry.OpampgatewayDownstreamConnections.Add(ctx, 1)
	}

	// find the upstream connection
	conn, err := o.client.assignedUpstreamConnection(agentID)
	if err != nil {
		return fmt.Errorf("get upstream connection: %w", err)
	}

	// send the message to the upstream connection
	o.logger.Info("Forwarding message upstream", zap.String("agent_id", agentID), zap.String("connection_id", conn.id), zap.String("message", message.String()))
	conn.send(messageBytes)
	o.telemetry.OpampgatewayUpstreamMessages.Add(context.Background(), 1)
	o.telemetry.OpampgatewayUpstreamMessageSize.Add(context.Background(), int64(len(messageBytes)))
	return nil
}

func (o *OpAMPGateway) HandleAgentError(ctx context.Context, connection *downstreamConnection, err error) {
	o.logger.Error("Error in agent connection", zap.Error(err))
}

func (o *OpAMPGateway) HandleAgentClose(ctx context.Context, connection *downstreamConnection) error {
	agentID := connection.id
	o.client.unassignUpstreamConnection(agentID)
	o.server.removeDownstreamConnection(connection)
	return nil
}

// --------------------------------------------------------------------------------------
// Server callbacks

// HandleServerMessage handles message set from the server to an agent
func (o *OpAMPGateway) HandleServerMessage(ctx context.Context, connection *upstreamConnection, messageNumber int, messageType int, messageBytes []byte) error {
	if messageType != websocket.BinaryMessage {
		err := fmt.Errorf("unexpected message type: %v, must be binary message", messageType)
		o.logger.Error("Cannot process a message from WebSocket", zap.Error(err), zap.Int("message_number", messageNumber), zap.Int("message_type", messageType), zap.String("message_bytes", string(messageBytes)))
		return err
	}

	message := protobufs.ServerToAgent{}
	if err := decodeWSMessage(messageBytes, &message); err != nil {
		return fmt.Errorf("failed to decode ws message: %w", err)
	}

	agentID, err := parseAgentID(message.GetInstanceUid())
	if err != nil {
		return fmt.Errorf("failed to parse agent id: %w", err)
	}

	// find the downstream connection from the server
	conn, ok := o.server.getDownstreamConnection(agentID)
	if !ok {
		// downstream connection no longer exists. just ignore the message for now.
		return nil
	}

	// forward the message to the downstream connection
	conn.send(messageBytes)
	o.telemetry.OpampgatewayDownstreamMessages.Add(context.Background(), 1)
	o.telemetry.OpampgatewayDownstreamMessageSize.Add(context.Background(), int64(len(messageBytes)))
	return nil
}

func (o *OpAMPGateway) HandleServerError(ctx context.Context, connection *upstreamConnection, err error) {
	o.logger.Error("Error in server connection", zap.Error(err))
}

func (o *OpAMPGateway) HandleServerClose(ctx context.Context, connection *upstreamConnection) error {
	o.logger.Info("Server connection closed", zap.String("connection_id", connection.id))
	return nil
}
