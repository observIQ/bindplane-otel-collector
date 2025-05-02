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

package backoff_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component/componenttest"

	"github.com/observiq/bindplane-otel-collector/internal/aws/backoff"
)

func TestExponentialBackoff(t *testing.T) {
	tests := []struct {
		name                 string
		standardPollInterval time.Duration
		maxPollInterval      time.Duration
		backoffFactor        float64
		messagesSequence     []int // 0 means no messages, non-zero means messages found
		expectedIntervals    []time.Duration
	}{
		{
			name:                 "basic_backoff_then_reset",
			standardPollInterval: 5 * time.Second,
			maxPollInterval:      30 * time.Second,
			backoffFactor:        2.0,
			messagesSequence:     []int{0, 0, 0, 5, 0},
			expectedIntervals:    []time.Duration{10 * time.Second, 20 * time.Second, 30 * time.Second, 5 * time.Second, 10 * time.Second},
		},
		{
			name:                 "cap_at_max_interval",
			standardPollInterval: 10 * time.Second,
			maxPollInterval:      25 * time.Second,
			backoffFactor:        3.0,
			messagesSequence:     []int{0, 0, 0},
			expectedIntervals:    []time.Duration{25 * time.Second, 25 * time.Second, 25 * time.Second},
		},
		{
			name:                 "equal_min_max_interval",
			standardPollInterval: 15 * time.Second,
			maxPollInterval:      15 * time.Second,
			backoffFactor:        1.5,
			messagesSequence:     []int{0, 0, 0},
			expectedIntervals:    []time.Duration{15 * time.Second, 15 * time.Second, 15 * time.Second},
		},
		{
			name:                 "fractional_backoff_factor",
			standardPollInterval: 10 * time.Second,
			maxPollInterval:      30 * time.Second,
			backoffFactor:        1.5,
			messagesSequence:     []int{0, 0, 0, 0},
			expectedIntervals:    []time.Duration{15 * time.Second, time.Duration(22.5 * float64(time.Second)), 30 * time.Second, 30 * time.Second},
		},
		{
			name:                 "multiple_resets",
			standardPollInterval: 5 * time.Second,
			maxPollInterval:      60 * time.Second,
			backoffFactor:        2.0,
			messagesSequence:     []int{0, 0, 3, 0, 0, 1, 0},
			expectedIntervals:    []time.Duration{10 * time.Second, 20 * time.Second, 5 * time.Second, 10 * time.Second, 20 * time.Second, 5 * time.Second, 10 * time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telemetry := componenttest.NewNopTelemetrySettings()
			bkoff := backoff.New(telemetry, tt.standardPollInterval, tt.maxPollInterval, tt.backoffFactor)

			actualIntervals := make([]time.Duration, 0, len(tt.messagesSequence))
			for _, msgCount := range tt.messagesSequence {
				actualIntervals = append(actualIntervals, bkoff.Update(msgCount))
			}
			assert.Equal(t, tt.expectedIntervals, actualIntervals)
		})
	}
}
