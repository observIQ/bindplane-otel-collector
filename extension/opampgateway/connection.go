package opampgateway

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type connection struct {
	id   string
	conn *wsConnection
}

func newConnection(conn *websocket.Conn, logger *zap.Logger) *connection {
	return &connection{
		conn: newWSConnection(conn, logger),
	}
}
