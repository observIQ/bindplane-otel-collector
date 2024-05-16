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

	// SearchQuery is an optional query string that will filter telemetry
	// such that only telemetry containing the string is reported.
	SearchQuery *string `yaml:"search_query"`

	// MinimumTimestamp is the minimum timestamp used to filter telemetry such that only telemetry
	// with a timestamp higher than specified will be reported.
	MinimumTimestamp *time.Time `yaml:"minimum_timestamp"`

	// SessionID is the identifier that the response should use to match the request with the response.
	SessionID string `yaml:"session_id"`
}