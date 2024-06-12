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
	"time"

	"go.opentelemetry.io/collector/component"
)

// snapshotRequest specifies what snapshots to collect
type snapshotRequest struct {
	// Processor is the full ComponentID of the snapshot processor
	Processor component.ID `yaml:"processor"`

	// PipelineType will be "logs", "metrics", or "traces"
	PipelineType string `yaml:"pipeline_type"`

	// SessionID is the identifier that can be used to match the request with the response.
	SessionID string `yaml:"session_id"`

	// SearchQuery is an optional query string that will filter telemetry
	// such that only telemetry containing the string is reported.
	SearchQuery *string `yaml:"search_query"`

	// MinimumTimestamp is the minimum timestamp used to filter telemetry such that only telemetry
	// with a timestamp higher than specified will be reported.
	MinimumTimestamp *time.Time `yaml:"minimum_timestamp"`
}
