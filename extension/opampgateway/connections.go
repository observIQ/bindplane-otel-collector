package opampgateway

import (
	"sync"
	"sync/atomic"
)

type connections struct {
	agentConnections sync.Map
	count            atomic.Int32
}

func newConnections() *connections {
	return &connections{}
}

func (c *connections) get(agentID string) (conn *connection, ok bool) {
	result, ok := c.agentConnections.Load(agentID)
	if !ok {
		return nil, false
	}
	conn = result.(*connection)
	return conn, true
}

func (c *connections) set(agentID string, conn *connection) {
	c.agentConnections.Store(agentID, conn)
	c.count.Add(1)
	conn.incrementAgentCount()
}

func (c *connections) remove(agentID string) {
	result, ok := c.agentConnections.LoadAndDelete(agentID)
	if !ok {
		return
	}
	conn := result.(*connection)
	c.count.Add(-1)
	conn.decrementAgentCount()
}

func (c *connections) size() int {
	return int(c.count.Load())
}
