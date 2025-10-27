package opampgateway

import (
	"sync"
)

// connectionProvider is a function that returns a connection for the given agent ID. if
// no connections are available, it should block until one is available.
type connectionProvider func() *connection

type connections struct {
	byConnectionID map[string]*connection
	mtx            sync.RWMutex
}

func newConnections() *connections {
	return &connections{
		byConnectionID: make(map[string]*connection),
	}
}

// get returns a connection for the given agent ID. if the connection is not
// found, it will return nil.
func (c *connections) get(connectionID string) (*connection, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	conn, exists := c.byConnectionID[connectionID]
	return conn, exists
}

func (c *connections) set(connectionID string, conn *connection) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.byConnectionID[connectionID] = conn
	conn.incrementAgentCount()
}

func (c *connections) remove(connectionID string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	conn, exists := c.byConnectionID[connectionID]
	if !exists {
		return
	}
	conn.decrementAgentCount()
	delete(c.byConnectionID, connectionID)
}

func (c *connections) size() int {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return len(c.byConnectionID)
}
