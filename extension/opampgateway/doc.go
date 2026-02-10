// Package opampgateway implements an OpenTelemetry Collector extension that
// relays OpAMP messages between downstream agents and an upstream OpAMP server.
//
// The gateway multiplexes many downstream agent WebSocket connections over a
// smaller number of persistent upstream WebSocket connections. It transparently
// forwards AgentToServer messages upstream and ServerToAgent messages back to
// the originating agent. Agent authentication is delegated to the upstream
// server via OpampGatewayConnect/OpampGatewayConnectResult custom messages
// before the downstream WebSocket is established.
//
//go:generate mdatagen metadata.yaml
package opampgateway
