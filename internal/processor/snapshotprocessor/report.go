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

package snapshotprocessor

import (
	"encoding/json"
	"time"

	"github.com/dynatrace/dynatrace-bindplane-otel-contrib/pkg/snapshot"
	"go.uber.org/zap"
)

// snapshotReport is the response framing for an OpAMP custom-message
// snapshot. The on-the-wire format mirrors
// bindplane-otel-contrib/processor/snapshotprocessor's report schema.
type snapshotReport struct {
	SessionID     string `json:"session_id"`
	TelemetryType string `json:"telemetry_type"`
	// TelemetryPayload is the logs/metrics/traces in OTLP/JSON format.
	TelemetryPayload json.RawMessage `json:"telemetry_payload"`
	// Metadata is absent when the processor had no buffer to collect from.
	Metadata *snapshotReportMetadata `json:"metadata,omitempty"`
}

// snapshotReportMetadata tells the server how the request was served: which
// collection mode the buffer was in, whether a just-in-time fill completed,
// how long the request waited and took, and how many items the payload
// holds. The payload alone cannot show any of this.
type snapshotReportMetadata struct {
	Mode   string `json:"mode"`   // "continuous" or "on_demand"
	Filled bool   `json:"filled"` // false: the on-demand fill wait expired and the last known records were served
	// Timings are fractional milliseconds: a continuous-mode request is
	// usually served in well under one.
	FillWaitMs float64 `json:"fill_wait_ms"`
	DurationMs float64 `json:"duration_ms"`
	Records    int     `json:"records"`
}

func newReportMetadata(stats snapshot.RequestStats) *snapshotReportMetadata {
	return &snapshotReportMetadata{
		Mode:       string(stats.Mode),
		Filled:     stats.Filled,
		FillWaitMs: millis(stats.FillWait),
		DurationMs: millis(stats.Duration),
		Records:    stats.Records,
	}
}

func millis(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// logFields describes the metadata for the processor's logs.
func (m *snapshotReportMetadata) logFields() []zap.Field {
	return []zap.Field{
		zap.String("mode", m.Mode),
		zap.Bool("filled", m.Filled),
		zap.Float64("fill_wait_ms", m.FillWaitMs),
		zap.Float64("duration_ms", m.DurationMs),
		zap.Int("records", m.Records),
	}
}

func logsReport(sessionID string, payload []byte) snapshotReport {
	return snapshotReport{SessionID: sessionID, TelemetryType: "logs", TelemetryPayload: payload}
}

func metricsReport(sessionID string, payload []byte) snapshotReport {
	return snapshotReport{SessionID: sessionID, TelemetryType: "metrics", TelemetryPayload: payload}
}

func tracesReport(sessionID string, payload []byte) snapshotReport {
	return snapshotReport{SessionID: sessionID, TelemetryType: "traces", TelemetryPayload: payload}
}
