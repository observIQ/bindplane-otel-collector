package opampgateway

type connectionPool struct {
	connections map[string]*connection
}

func newConnectionPool() *connectionPool {
	return &connectionPool{
		connections: map[string]*connection{},
	}
}

func (c *connectionPool) add() {}
