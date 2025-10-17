package opampgateway

import (
	"sync/atomic"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type connection struct {
	id    string
	conn  *wsConnection
	count atomic.Int32
}

func newConnection(conn *websocket.Conn, logger *zap.Logger) *connection {
	return &connection{
		conn:  newWSConnection(conn, logger),
		count: atomic.Int32{},
	}
}

func (c *connection) agentCount() int {
	return int(c.count.Load())
}

func (c *connection) incrementAgentCount() {
	c.count.Add(1)
}

func (c *connection) decrementAgentCount() {
	c.count.Add(-1)
}
