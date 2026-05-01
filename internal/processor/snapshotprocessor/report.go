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

import "encoding/json"

// snapshotReport is the response framing for an OpAMP custom-message
// snapshot. The on-the-wire format mirrors
// bindplane-otel-contrib/processor/snapshotprocessor's report schema.
type snapshotReport struct {
	SessionID     string `json:"session_id"`
	TelemetryType string `json:"telemetry_type"`
	// TelemetryPayload is the logs/metrics/traces in OTLP/JSON format.
	TelemetryPayload json.RawMessage `json:"telemetry_payload"`
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
