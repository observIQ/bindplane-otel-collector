package opampgateway

import "time"

type message struct {
	data     []byte
	received time.Time
	number   int
}

// newMessage creates a new message from the data received
func newMessage(number int, data []byte) *message {
	return &message{
		data:     data,
		received: time.Now(),
		number:   number,
	}
}

func (m *message) elapsedTime() time.Duration {
	return time.Since(m.received)
}
