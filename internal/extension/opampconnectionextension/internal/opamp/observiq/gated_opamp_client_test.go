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
	"context"
	"testing"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/extension/opampconnectionextension/internal/opamp/mocks"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGatedOpAMPClient_WaitsForGateBeforeDelegating(t *testing.T) {
	gate := newSendGate()
	wait := 100 * time.Millisecond
	gate.block(wait)

	mockClient := new(mocks.MockOpAMPClient)
	var calledAt time.Time
	mockClient.On("SetHealth", mock.Anything).Return(nil).Run(func(_ mock.Arguments) {
		calledAt = time.Now()
	})

	gated := newGatedOpAMPClient(mockClient, gate)

	start := time.Now()
	err := gated.SetHealth(&protobufs.ComponentHealth{})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, calledAt.Sub(start), wait)

	mockClient.AssertExpectations(t)
}

func TestGatedOpAMPClient_NoDelayWhenGateOpen(t *testing.T) {
	gate := newSendGate()

	mockClient := new(mocks.MockOpAMPClient)
	mockClient.On("SendCustomMessage", mock.Anything).Return((chan struct{})(nil), nil)

	gated := newGatedOpAMPClient(mockClient, gate)

	start := time.Now()
	_, err := gated.SendCustomMessage(&protobufs.CustomMessage{})
	assert.NoError(t, err)
	assert.Less(t, time.Since(start), 50*time.Millisecond)

	mockClient.AssertExpectations(t)
}

func TestGatedOpAMPClient_StopIsNotGated(t *testing.T) {
	gate := newSendGate()
	gate.block(10 * time.Second)

	mockClient := new(mocks.MockOpAMPClient)
	mockClient.On("Stop", mock.Anything).Return(nil)

	gated := newGatedOpAMPClient(mockClient, gate)

	start := time.Now()
	err := gated.Stop(context.Background())
	assert.NoError(t, err)
	assert.Less(t, time.Since(start), 50*time.Millisecond)

	mockClient.AssertExpectations(t)
}
