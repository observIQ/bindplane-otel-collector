package opampgateway

import (
	"sync"

	"go.uber.org/zap"
)

type connectionPool struct {
	connections map[string]*connection
	logger      *zap.Logger
	mtx         sync.RWMutex
}

func newConnectionPool(logger *zap.Logger) *connectionPool {
	return &connectionPool{
		connections: map[string]*connection{},
		logger:      logger.Named("opamp-gateway-connection-pool"),
	}
}

func (c *connectionPool) add(conn *connection) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.connections[conn.id] = conn

	_, ok := c.connections[conn.id]
	if !ok {
		c.connections[conn.id] = conn
	} else {
		c.logger.Error("upstream connection for this id already exists")
	}
}

func (c *connectionPool) next() *connection {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	for _, conn := range c.connections {
		return conn
	}
	return nil
}
