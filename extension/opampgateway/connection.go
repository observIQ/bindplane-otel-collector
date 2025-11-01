package opampgateway

import "context"

type ConnectionCallbacks[T any] struct {
	// OnMessage is called when a message is received.
	OnMessage func(ctx context.Context, connection T, messageNumber int, messageType int, messageBytes []byte) error

	// OnError is called when an error occurs. The connection will be closed and the context
	// will be cancelled after this call.
	OnError func(ctx context.Context, connection T, err error)

	// OnClose is called when the connection is closed.
	OnClose func(ctx context.Context, connection T) error
}
