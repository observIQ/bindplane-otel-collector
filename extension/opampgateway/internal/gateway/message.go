// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gateway

import "time"

type message struct {
	data     []byte
	received time.Time
	number   int

	// done is an optional channel signaled when the message has been written to
	// the WebSocket connection. A nil channel means fire-and-forget.
	done chan error
}

// newMessage creates a new message from the data received
func newMessage(number int, data []byte) *message {
	return &message{
		data:     data,
		received: time.Now(),
		number:   number,
		done:     make(chan error, 1),
	}
}

func (m *message) elapsedTime() time.Duration {
	return time.Since(m.received)
}

// complete signals any waiter that the message write has finished.
// It is safe to call multiple times; only the first call delivers.
func (m *message) complete(err error) {
	if m.done != nil {
		select {
		case m.done <- err:
		default:
		}
	}
}
