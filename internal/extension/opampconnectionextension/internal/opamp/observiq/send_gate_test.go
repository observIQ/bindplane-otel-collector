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
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSendGate_WaitWithoutBlockReturnsImmediately(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newSendGate()

		start := time.Now()
		g.wait()
		assert.Equal(t, time.Duration(0), time.Since(start))
	})
}

func TestSendGate_BlockDelaysWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newSendGate()

		wait := 100 * time.Millisecond
		g.block(wait)

		start := time.Now()
		g.wait()
		assert.Equal(t, wait, time.Since(start))
	})
}

func TestSendGate_LongerBlockWins(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newSendGate()

		g.block(50 * time.Millisecond)
		g.block(200 * time.Millisecond)

		start := time.Now()
		g.wait()
		assert.Equal(t, 200*time.Millisecond, time.Since(start))
	})
}

func TestSendGate_ShorterBlockDoesNotShortenExistingDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newSendGate()

		g.block(200 * time.Millisecond)
		g.block(50 * time.Millisecond)

		start := time.Now()
		g.wait()
		assert.Equal(t, 200*time.Millisecond, time.Since(start))
	})
}

func TestSendGate_CloseUnblocksWait(t *testing.T) {
	synctest.Test(t, func(_ *testing.T) {
		g := newSendGate()
		g.block(10 * time.Second)

		done := make(chan struct{})
		go func() {
			g.wait()
			close(done)
		}()

		g.close()
		<-done
	})
}

func TestSendGate_CloseIsIdempotent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newSendGate()
		assert.NotPanics(t, func() {
			g.close()
			g.close()
		})
	})
}

func TestSendGate_NonPositiveBlockIsNoop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := newSendGate()
		g.block(0)
		g.block(-1 * time.Second)

		start := time.Now()
		g.wait()
		assert.Equal(t, time.Duration(0), time.Since(start))
	})
}
