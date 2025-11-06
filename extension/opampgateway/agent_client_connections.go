package opampgateway

import (
	"sync"
)

// agentClientConnections is a map of agent IDs to connection IDs. it is used to track the
// connections for the agents. it is also used to assign connections to agents and remove
// connections when they are no longer valid.
type agentClientConnections struct {
	agentToConnectionID map[string]string
	connections         *connections[*upstreamConnection]
	pool                *connectionPool
	mtx                 sync.Mutex
}

func newAgentClientConnections(connections *connections[*upstreamConnection], pool *connectionPool) *agentClientConnections {
	return &agentClientConnections{
		agentToConnectionID: make(map[string]string),
		connections:         connections,
		pool:                pool,
	}
}

// assignedAgentConnection returns the connection for the given agent ID. if the connection is not
// found, it will assign a new connection from the pool and return it.
func (a *agentClientConnections) assignedAgentConnection(agentID string) (*upstreamConnection, bool) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	connectionID, exists := a.agentToConnectionID[agentID]
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
	a.agentToConnectionID[agentID] = conn.id
	conn.incrementAgentCount()
	return conn, exists
}

// unassignAgentConnection unassigns the connection for the given agent ID. it will not
// remove the connection from the list of connections or the pool.
func (a *agentClientConnections) unassignAgentConnection(agentID string) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	connectionID, exists := a.agentToConnectionID[agentID]
	if exists {
		conn, exists := a.connections.get(connectionID)
		if exists {
			conn.decrementAgentCount()
		}
	}
	delete(a.agentToConnectionID, agentID)
}

// removeConnection removes the connection for the given connection ID. it will also
// remove all assignments for this connection. it will not remove the connection from the
// list of connections or the pool.
func (a *agentClientConnections) removeConnection(connectionID string) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	conn, exists := a.connections.get(connectionID)
	// remove all assignments for this connection
	for agentID, id := range a.agentToConnectionID {
		if id == connectionID {
			delete(a.agentToConnectionID, agentID)

			// decrement the agent count for the connection
			if exists {
				conn.decrementAgentCount()
			}
		}
	}
}
