package opampgateway

import (
	"errors"
	"math"
	"sync"

	"go.uber.org/zap"
)

// ErrNoUpstreamConnectionsAvailable is returned when no upstream connections that are
// connected to the upstream OpAMP server are available.
var ErrNoUpstreamConnectionsAvailable = errors.New("no upstream connections available")

type connectionPool struct {
	connections map[string]*upstreamConnection
	logger      *zap.Logger
	mtx         sync.RWMutex
}

func newConnectionPool(size int, logger *zap.Logger) *connectionPool {
	return &connectionPool{
		connections: make(map[string]*upstreamConnection, size),
		logger:      logger.Named("opamp-gateway-connection-pool"),
	}
}

func (c *connectionPool) add(conn *upstreamConnection) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	_, ok := c.connections[conn.id]
	if !ok {
		c.connections[conn.id] = conn
	} else {
		c.logger.Error("upstream connection for this id already exists")
	}
}

func (c *connectionPool) size() int {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return len(c.connections)
}

func (c *connectionPool) remove(conn *upstreamConnection) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	delete(c.connections, conn.id)
}

// next returns the next connection from the pool. if no connection is found, it will return nil and false.
func (c *connectionPool) next() (*upstreamConnection, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	// find the connection with the lowest count
	minCount := math.MaxInt32
	var minConn *upstreamConnection
	for _, conn := range c.connections {
		// if the connection is not connected, skip it
		if !conn.isConnected() {
			continue
		}
		count := conn.downstreamCount()
		if count < minCount {
			minCount = count
			minConn = conn
		}
	}
	return minConn, minConn != nil
}
