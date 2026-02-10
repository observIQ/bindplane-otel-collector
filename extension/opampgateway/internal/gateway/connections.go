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

import (
	"sync"
)

// Connection is the interface for connections managed by the connections registry.
type Connection interface {
}

type connections[T Connection] struct {
	byConnectionID map[string]T
	mtx            sync.RWMutex
}

func newConnections[T Connection]() *connections[T] {
	return &connections[T]{
		byConnectionID: make(map[string]T),
	}
}

// get returns a connection for the given agent ID. if the connection is not
// found, it will return nil.
func (c *connections[T]) get(connectionID string) (T, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	conn, exists := c.byConnectionID[connectionID]
	return conn, exists
}

func (c *connections[T]) set(connectionID string, conn T) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.byConnectionID[connectionID] = conn
}

func (c *connections[T]) remove(connectionID string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	delete(c.byConnectionID, connectionID)
}

func (c *connections[T]) size() int {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return len(c.byConnectionID)
}
