package opampgateway

import "github.com/gorilla/websocket"

type connection struct {
	id string
	*websocket.Conn
}

func newConnection(conn *websocket.Conn) *connection {
	return &connection{}
}
