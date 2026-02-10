package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newTestAssignments creates a connectionAssignments with the given upstream connections
// already added to both the connections registry and the pool.
func newTestAssignments(upstreams []*upstreamConnection) *connectionAssignments {
	pool := newConnectionPool(len(upstreams), zap.NewNop())
	conns := newConnections[*upstreamConnection]()
	for _, u := range upstreams {
		pool.add(u)
		conns.set(u.id, u)
	}
	return newConnectionAssignments(conns, pool)
}

func TestAssignedUpstreamConnectionNewAssignment(t *testing.T) {
	u := newTestUpstreamConn("u-0", true)
	a := newTestAssignments([]*upstreamConnection{u})

	conn, ok := a.assignedUpstreamConnection("d-1")
	require.True(t, ok)
	assert.Equal(t, "u-0", conn.id)
	assert.Equal(t, 1, u.downstreamCount(), "downstream count should be incremented")
}

func TestAssignedUpstreamConnectionReturnsExisting(t *testing.T) {
	u := newTestUpstreamConn("u-0", true)
	a := newTestAssignments([]*upstreamConnection{u})

	// first call assigns
	conn1, ok := a.assignedUpstreamConnection("d-1")
	require.True(t, ok)

	// second call returns the same assignment
	conn2, ok := a.assignedUpstreamConnection("d-1")
	require.True(t, ok)
	assert.Equal(t, conn1.id, conn2.id)
	assert.Equal(t, 1, u.downstreamCount(), "count should only be incremented once")
}

func TestAssignedUpstreamConnectionReassignsStale(t *testing.T) {
	u0 := newTestUpstreamConn("u-0", true)
	u1 := newTestUpstreamConn("u-1", true)
	a := newTestAssignments([]*upstreamConnection{u0, u1})

	// assign d-1 to u-0
	conn, ok := a.assignedUpstreamConnection("d-1")
	require.True(t, ok)
	originalID := conn.id

	// remove the assigned connection from the connections registry (simulating disconnect)
	a.connections.remove(originalID)

	// next call should reassign from pool
	conn2, ok := a.assignedUpstreamConnection("d-1")
	require.True(t, ok)
	assert.NotNil(t, conn2)
}

func TestAssignedUpstreamConnectionEmptyPool(t *testing.T) {
	a := newTestAssignments(nil)

	conn, ok := a.assignedUpstreamConnection("d-1")
	assert.False(t, ok)
	assert.Nil(t, conn)
}

func TestAssignedUpstreamConnectionAllDisconnected(t *testing.T) {
	u := newTestUpstreamConn("u-0", false)
	a := newTestAssignments([]*upstreamConnection{u})

	conn, ok := a.assignedUpstreamConnection("d-1")
	assert.False(t, ok)
	assert.Nil(t, conn)
}

func TestAssignedUpstreamConnectionLeastLoaded(t *testing.T) {
	u0 := newTestUpstreamConn("u-0", true)
	u1 := newTestUpstreamConn("u-1", true)
	a := newTestAssignments([]*upstreamConnection{u0, u1})

	// assign 3 downstreams to populate counts
	a.assignedUpstreamConnection("d-1")
	a.assignedUpstreamConnection("d-2")
	a.assignedUpstreamConnection("d-3")

	// both connections should have assignments (pool picks least loaded)
	totalCount := u0.downstreamCount() + u1.downstreamCount()
	assert.Equal(t, 3, totalCount)
}

func TestRemoveDownstreamConnectionIDs(t *testing.T) {
	u0 := newTestUpstreamConn("u-0", true)
	u1 := newTestUpstreamConn("u-1", true)
	// give u1 a high count so all assignments go to u0
	u1.count.Store(100)
	a := newTestAssignments([]*upstreamConnection{u0, u1})

	a.assignedUpstreamConnection("d-1")
	a.assignedUpstreamConnection("d-2")

	removed := a.removeDownstreamConnectionIDs("u-0")
	assert.Len(t, removed, 2)
	assert.ElementsMatch(t, []string{"d-1", "d-2"}, removed)

	// assignments should be gone, so new call should reassign
	conn, ok := a.assignedUpstreamConnection("d-1")
	require.True(t, ok)
	assert.NotNil(t, conn)
}

func TestRemoveDownstreamConnectionIDsNoMatch(t *testing.T) {
	u := newTestUpstreamConn("u-0", true)
	a := newTestAssignments([]*upstreamConnection{u})

	a.assignedUpstreamConnection("d-1")

	removed := a.removeDownstreamConnectionIDs("u-nonexistent")
	assert.Empty(t, removed)
}

func TestUnassignDownstreamConnection(t *testing.T) {
	u := newTestUpstreamConn("u-0", true)
	a := newTestAssignments([]*upstreamConnection{u})

	a.assignedUpstreamConnection("d-1")
	assert.Equal(t, 1, u.downstreamCount())

	a.unassignDownstreamConnection("d-1")
	assert.Equal(t, 0, u.downstreamCount(), "count should be decremented")

	// calling again for same ID should not panic or decrement further
	a.unassignDownstreamConnection("d-1")
	assert.Equal(t, 0, u.downstreamCount())
}

func TestUnassignDownstreamConnectionNonExistent(t *testing.T) {
	a := newTestAssignments(nil)

	// should not panic
	a.unassignDownstreamConnection("d-nonexistent")
}

func TestRemoveConnection(t *testing.T) {
	u := newTestUpstreamConn("u-0", true)
	a := newTestAssignments([]*upstreamConnection{u})

	a.assignedUpstreamConnection("d-1")
	a.assignedUpstreamConnection("d-2")
	assert.Equal(t, 2, u.downstreamCount())

	a.removeConnection("u-0")
	assert.Equal(t, 0, u.downstreamCount(), "all downstream counts should be decremented")
}

func TestRemoveConnectionNonExistent(t *testing.T) {
	a := newTestAssignments(nil)

	// should not panic
	a.removeConnection("u-nonexistent")
}
