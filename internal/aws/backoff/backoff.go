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

// Package backoff provides a backoff calculator for the AWS S3 event receiver.
package backoff // import "github.com/observiq/bindplane-otel-collector/internal/aws/backoff"

import (
	"time"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

// Backoff implements the PollBackoff interface with exponential backoff
type Backoff struct {
	telemetry        component.TelemetrySettings
	standardInterval time.Duration
	maxInterval      time.Duration
	backoffFactor    float64
	currentInterval  time.Duration
}

// New creates a new backoff calculator
func New(
	telemetry component.TelemetrySettings,
	standardPollInterval time.Duration,
	maxPollInterval time.Duration,
	backoffFactor float64,
) *Backoff {
	return &Backoff{
		telemetry:        telemetry,
		standardInterval: standardPollInterval,
		maxInterval:      maxPollInterval,
		backoffFactor:    backoffFactor,
		currentInterval:  standardPollInterval,
	}
}

// Update calculates the next poll interval based on message count
// If messages are found (numMessages > 0), resets to standard interval
// If no messages are found, increases by the backoff factor up to the maximum
func (b *Backoff) Update(numMessages int) time.Duration {
	if numMessages > 0 {
		b.telemetry.Logger.Debug("messages found, scheduling standard interval", zap.Duration("interval", b.standardInterval))
		b.currentInterval = b.standardInterval
		return b.currentInterval
	}

	if next := time.Duration(float64(b.currentInterval) * b.backoffFactor); next < b.maxInterval {
		b.telemetry.Logger.Debug("no messages found, scheduling longer interval", zap.Duration("interval", next))
		b.currentInterval = next
		return b.currentInterval
	}

	b.telemetry.Logger.Debug("no messages found, scheduling max interval", zap.Duration("interval", b.maxInterval))
	b.currentInterval = b.maxInterval
	return b.currentInterval
}
