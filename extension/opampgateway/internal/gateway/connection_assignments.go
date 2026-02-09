package gateway

import (
	"sync"
)

// connectionAssignments is a map of downstream connection IDs to upstream connection IDs.
// it is used to track the assignments of downstream connections to upstream connections.
// it is also used to assign upstream connections to downstream connections and remove
// assignments when they are no longer valid.
type connectionAssignments struct {
	downstreamToUpstream map[string]string
	connections          *connections[*upstreamConnection]
	pool                 *connectionPool
	mtx                  sync.Mutex
}

func newConnectionAssignments(connections *connections[*upstreamConnection], pool *connectionPool) *connectionAssignments {
	return &connectionAssignments{
		downstreamToUpstream: make(map[string]string),
		connections:          connections,
		pool:                 pool,
	}
}

// assignedUpstreamConnection returns the connection for the given downstream connection
// ID. if the upstream connection is not found, it will assign a new connection from the
// pool and return it.
func (a *connectionAssignments) assignedUpstreamConnection(downstreamConnectionID string) (*upstreamConnection, bool) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	connectionID, exists := a.downstreamToUpstream[downstreamConnectionID]
	if exists {
		// make sure the connection is still valid
		conn, exists := a.connections.get(connectionID)
		if exists {
			return conn, true
		}
	}
	// if no existing assignment, assign a new connection from the pool
	conn, exists := a.pool.next()
	if !exists {
		return nil, false
	}
	a.downstreamToUpstream[downstreamConnectionID] = conn.id
	conn.incrementDownstreamCount()
	return conn, exists
}

// removeDownstreamConnectionIDs removes the downstream connection IDs for the given
// upstream connection ID. it will return a sequence of downstream connection IDs that
// were removed.
func (a *connectionAssignments) removeDownstreamConnectionIDs(upstreamConnectionID string) []string {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	ids := []string{}
	for down, up := range a.downstreamToUpstream {
		if up == upstreamConnectionID {
			delete(a.downstreamToUpstream, down)
			ids = append(ids, down)
		}
	}
	return ids
}

// unassignDownstreamConnection unassigns the connection for the given downstream
// connection ID. it will not remove the connection from the list of connections or the
// pool.
func (a *connectionAssignments) unassignDownstreamConnection(downstreamConnectionID string) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	connectionID, exists := a.downstreamToUpstream[downstreamConnectionID]
	if exists {
		conn, exists := a.connections.get(connectionID)
		if exists {
			conn.decrementDownstreamCount()
		}
	}
	delete(a.downstreamToUpstream, downstreamConnectionID)
}

// removeConnection removes the connection for the given connection ID. it will also
// remove all assignments for this connection. it will not remove the connection from the
// list of connections or the pool.
func (a *connectionAssignments) removeConnection(connectionID string) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	conn, exists := a.connections.get(connectionID)
	// remove all assignments for this connection
	for agentID, id := range a.downstreamToUpstream {
		if id == connectionID {
			delete(a.downstreamToUpstream, agentID)

			// decrement the agent count for the connection
			if exists {
				conn.decrementDownstreamCount()
			}
		}
	}
}
