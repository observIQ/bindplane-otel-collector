// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gateway

import "context"

// ConnectionCallbacks defines the callback functions for handling connection events.
type ConnectionCallbacks[T any] struct {
	// OnMessage is called when a message is received.
	OnMessage func(ctx context.Context, connection T, messageType int, message *message) error

	// OnError is called when an error occurs. The connection will be closed and the context
	// will be cancelled after this call.
	OnError func(ctx context.Context, connection T, err error)

	// OnClose is called when the connection is closed.
	OnClose func(ctx context.Context, connection T) error
}
