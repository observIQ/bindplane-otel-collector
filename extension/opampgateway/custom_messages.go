package opampgateway

import (
	"net/http"
)

// OpampGatewayConnect represents an authentication request sent by a gateway on behalf of an
// agent. The gateway sends this message to validate an agent's credentials before allowing
// the agent to connect through the gateway.
type OpampGatewayConnect struct {
	// RequestUID is a unique identifier for this connection request, used to correlate
	// the request with its response.
	RequestUID string `json:"request_uid,omitempty"`

	// RemoteAddress is the remote address of the connection (from the socket connection) of
	// the agent that is connecting to the gateway.
	RemoteAddress string `json:"remote_address"`

	// Headers contains HTTP-style headers the server can use to validate the agent
	// connection.
	Headers http.Header `json:"headers"`
}

// OpampGatewayConnectResult represents the server's response to an OpampGatewayConnect request.
// It indicates whether the agent's credentials were accepted and includes any response headers.
type OpampGatewayConnectResult struct {
	// RequestUID is the unique identifier from the corresponding OpampGatewayConnect request,
	// used to correlate the response with its request.
	RequestUID string `json:"request_uid,omitempty"`

	Accept          bool              `json:"accept"`
	HTTPStatusCode  int               `json:"http_status_code"`
	ResponseHeaders map[string]string `json:"response_headers"`
}

const (
	// OpampGatewayCapability is the capability identifier for opamp-gateway custom messages.
	OpampGatewayCapability = "com.bindplane.opamp-gateway"

	// OpampGatewayConnectType is sent when an agent connects to the gateway. It includes the
	// headers and client certificate that can be validated by the server to allow the
	// connection.
	OpampGatewayConnectType = "connect"

	// OpampGatewayConnectResultType is sent as a response to the connect message. It will
	// either allow or deny the agent connection.
	OpampGatewayConnectResultType = "connectResult"

	// // opampGatewayDisconnect is sent from the server to force the agent to disconnect from
	// // the gateway.
	// opampGatewayDisconnect = "disconnect"
)
