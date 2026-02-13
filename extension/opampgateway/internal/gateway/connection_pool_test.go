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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestUpstreamConn(id string, connected bool) *upstreamConnection {
	conn := &upstreamConnection{id: id}
	conn.connected.Store(connected)
	return conn
}

func TestConnectionPoolAdd(t *testing.T) {
	pool := newConnectionPool(2, zap.NewNop())
	conn := newTestUpstreamConn("u-0", true)

	pool.add(conn)
	assert.Equal(t, 1, pool.size())
}

func TestConnectionPoolAddDuplicate(t *testing.T) {
	pool := newConnectionPool(2, zap.NewNop())
	conn := newTestUpstreamConn("u-0", true)

	pool.add(conn)
	pool.add(conn)
	assert.Equal(t, 1, pool.size(), "duplicate add should not increase size")
}

func TestConnectionPoolRemove(t *testing.T) {
	pool := newConnectionPool(2, zap.NewNop())
	conn := newTestUpstreamConn("u-0", true)

	pool.add(conn)
	assert.Equal(t, 1, pool.size())

	pool.remove(conn)
	assert.Equal(t, 0, pool.size())
}

func TestConnectionPoolRemoveNonExistent(t *testing.T) {
	pool := newConnectionPool(2, zap.NewNop())
	conn := newTestUpstreamConn("u-0", true)

	// removing from empty pool should not panic
	pool.remove(conn)
	assert.Equal(t, 0, pool.size())
}

func TestConnectionPoolSizeEmpty(t *testing.T) {
	pool := newConnectionPool(2, zap.NewNop())
	assert.Equal(t, 0, pool.size())
}

func TestConnectionPoolNextEmpty(t *testing.T) {
	pool := newConnectionPool(2, zap.NewNop())

	conn, ok := pool.next()
	assert.False(t, ok)
	assert.Nil(t, conn)
}

func TestConnectionPoolNextAllDisconnected(t *testing.T) {
	pool := newConnectionPool(2, zap.NewNop())
	pool.add(newTestUpstreamConn("u-0", false))
	pool.add(newTestUpstreamConn("u-1", false))

	conn, ok := pool.next()
	assert.False(t, ok)
	assert.Nil(t, conn)
}

func TestConnectionPoolNextSingleConnected(t *testing.T) {
	pool := newConnectionPool(2, zap.NewNop())
	c := newTestUpstreamConn("u-0", true)
	pool.add(c)

	conn, ok := pool.next()
	require.True(t, ok)
	assert.Equal(t, "u-0", conn.id)
}

func TestConnectionPoolNextSkipsDisconnected(t *testing.T) {
	pool := newConnectionPool(2, zap.NewNop())
	pool.add(newTestUpstreamConn("u-0", false))
	connected := newTestUpstreamConn("u-1", true)
	pool.add(connected)

	conn, ok := pool.next()
	require.True(t, ok)
	assert.Equal(t, "u-1", conn.id)
}

func TestConnectionPoolNextLeastConnections(t *testing.T) {
	pool := newConnectionPool(3, zap.NewNop())

	c0 := newTestUpstreamConn("u-0", true)
	c0.count.Store(5)

	c1 := newTestUpstreamConn("u-1", true)
	c1.count.Store(2)

	c2 := newTestUpstreamConn("u-2", true)
	c2.count.Store(3)

	pool.add(c0)
	pool.add(c1)
	pool.add(c2)

	conn, ok := pool.next()
	require.True(t, ok)
	assert.Equal(t, "u-1", conn.id, "should return connection with lowest downstream count")
}
