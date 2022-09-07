// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package snapshot defines contract for collecting snapshots as well as helper structures
package snapshot

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Snapshotter represents an interface to save logs, metrics, and traces for snapshots
//
// //go:generate mockery --name Snapshotter --filename mock_snapshotter.go --structname MockSnapshotter
// No go generate for this as it requires internal otel structures that don't import correctly with mockery. Can uncomment above and run then modify by hand.
type Snapshotter interface {
	// SaveLogs saves off logs in a snapshot
	SaveLogs(componentID string, ld plog.Logs)

	// SaveTraces saves off traces in a snapshot
	SaveTraces(componentID string, td ptrace.Traces)

	// SaveMetrics saves off metrics in a snapshot
	SaveMetrics(componentID string, md pmetric.Metrics)
}
