// Copyright  Bindplane, Inc.
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

package snapshotprocessor

import (
	"context"

	"github.com/observiq/bindplane-otel-collector/internal/report"
	"github.com/observiq/bindplane-otel-collector/internal/report/snapshot"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// getSnapshotReporter is function for retrieving the SnapshotReporter.
// Meant to be overridden for tests.
var getSnapshotReporter func() *report.SnapshotReporter = report.GetSnapshotReporter

type snapshotProcessor struct {
	logger      *zap.Logger
	enabled     bool
	snapShotter snapshot.Snapshotter
	processorID string
}

// newSnapshotProcessor creates a new snapshot processor
func newSnapshotProcessor(logger *zap.Logger, cfg *Config, processorID string) *snapshotProcessor {
	return &snapshotProcessor{
		logger:      logger,
		enabled:     cfg.Enabled,
		snapShotter: getSnapshotReporter(),
		processorID: processorID,
	}
}

func (sp *snapshotProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if sp.enabled {
		newTraces := ptrace.NewTraces()
		td.CopyTo(newTraces)
		sp.snapShotter.SaveTraces(sp.processorID, newTraces)
	}

	return td, nil
}

func (sp *snapshotProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	if sp.enabled {
		newLogs := plog.NewLogs()
		ld.CopyTo(newLogs)
		sp.snapShotter.SaveLogs(sp.processorID, newLogs)
	}

	return ld, nil
}

func (sp *snapshotProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if sp.enabled {
		newMetrics := pmetric.NewMetrics()
		md.CopyTo(newMetrics)
		sp.snapShotter.SaveMetrics(sp.processorID, newMetrics)
	}

	return md, nil
}
