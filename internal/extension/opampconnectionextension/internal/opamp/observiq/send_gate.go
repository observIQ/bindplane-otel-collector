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

package observiq

import (
	"sync"
	"time"
)

// sendGate blocks outgoing OpAMP messages until a server-requested retry
// delay (see ServerErrorResponse.RetryInfo) has elapsed. It is shared between
// the Client and its gatedOpAMPClient so that a delay observed in
// onErrorHandler is enforced on every subsequent send attempt, rather than
// just delaying receipt of the next server message.
type sendGate struct {
	mu        sync.Mutex
	notBefore time.Time

	closeOnce sync.Once
	closed    chan struct{}
}

func newSendGate() *sendGate {
	return &sendGate{closed: make(chan struct{})}
}

// block delays outgoing messages until d has elapsed. If a longer delay is
// already in effect, the longer one wins.
func (g *sendGate) block(d time.Duration) {
	if d <= 0 {
		return
	}

	notBefore := time.Now().Add(d)

	g.mu.Lock()
	defer g.mu.Unlock()
	if notBefore.After(g.notBefore) {
		g.notBefore = notBefore
	}
}

// wait blocks until any delay set by block has elapsed, or the gate is closed.
func (g *sendGate) wait() {
	for {
		g.mu.Lock()
		remaining := time.Until(g.notBefore)
		g.mu.Unlock()

		if remaining <= 0 {
			return
		}

		select {
		case <-time.After(remaining):
		case <-g.closed:
			return
		}
	}
}

// close releases any goroutines currently blocked in wait. Safe to call
// multiple times.
func (g *sendGate) close() {
	g.closeOnce.Do(func() { close(g.closed) })
}
