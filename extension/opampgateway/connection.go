package opampgateway

import (
	"github.com/gorilla/websocket"
)

type connection struct {
	id   string
	conn *wsConnection
}

func newConnection(conn *websocket.Conn) *connection {
	return &connection{
		conn: newWSConnection(conn),
	}
}
