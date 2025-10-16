package opampgateway

import "sync"

type connections struct {
	agentConnections map[string]*connection
	mtx              sync.RWMutex
}

func newConnections() *connections {
	return &connections{
		agentConnections: map[string]*connection{},
	}
}

func (c *connections) get(agentID string) (conn *connection, ok bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	conn, ok = c.agentConnections[agentID]
	return
}

func (c *connections) set(agentID string, conn *connection) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.agentConnections[agentID] = conn
}

func (c *connections) size() int {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return len(c.agentConnections)
}
