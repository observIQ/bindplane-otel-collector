package opampgateway

import (
	"sync"
)

// connectionProvider is a function that returns a connection for the given agent ID. if
// no connections are available, it should block until one is available.
type connectionProvider func() *connection

type connections struct {
	agentConnections map[string]*connection
	mtx              sync.Mutex
}

func newConnections() *connections {
	return &connections{
		agentConnections: make(map[string]*connection),
	}
}

// getOrAssign returns a connection for the given agent ID. if the connection is not found, it
// will call the provider function to obtain a connection for the agent ID. If the
// provider is nil, no connection will be returned.
func (c *connections) getOrAssign(agentID string, provider connectionProvider) (*connection, bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	conn, exists := c.agentConnections[agentID]
	if exists {
		return conn, true
	}

	if provider == nil {
		return nil, false
	}

	conn = provider()
	if conn == nil {
		return nil, false
	}

	c.agentConnections[agentID] = conn
	conn.incrementAgentCount()
	return conn, true
}

// get returns a connection for the given agent ID.
func (c *connections) get(agentID string) (*connection, bool) {
	return c.getOrAssign(agentID, nil)
}

func (c *connections) set(agentID string, conn *connection) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.agentConnections[agentID] = conn
	conn.incrementAgentCount()
}

func (c *connections) remove(agentID string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	conn, exists := c.agentConnections[agentID]
	if !exists {
		return
	}
	conn.decrementAgentCount()
	delete(c.agentConnections, agentID)
}

func (c *connections) size() int {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return len(c.agentConnections)
}
