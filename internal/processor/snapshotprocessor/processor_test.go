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

package snapshotprocessor

import (
	"context"
	"testing"

	"github.com/observiq/bindplane-otel-collector/internal/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func Test_newSnapshotProcessor(t *testing.T) {
	reporter := report.NewSnapshotReporter(nil)
	defer overwriteSnapshotSet(t, reporter)()

	logger := zap.NewNop()
	cfg := &Config{Enabled: false}
	processorID := component.MustNewIDWithName("snapshotprocessor", "one")

	got := newSnapshotProcessor(logger, cfg, processorID)

	require.Equal(t, logger, got.logger)
	require.Equal(t, cfg.Enabled, got.enabled)
	require.Same(t, reporter, got.snapShotter)
	require.Equal(t, processorID, got.processorID)
}

func Test_processTraces(t *testing.T) {
	cases := []struct {
		name      string
		enabled   bool
		wantSaved bool
	}{
		{name: "disabled", enabled: false, wantSaved: false},
		{name: "enabled", enabled: true, wantSaved: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reporter := report.NewSnapshotReporter(nil)
			processorID := component.MustNewIDWithName("snapshotprocessor", "x")

			sp := &snapshotProcessor{
				logger:      zap.NewNop(),
				enabled:     tc.enabled,
				snapShotter: reporter,
				processorID: processorID,
			}

			td := ptrace.NewTraces()
			td.ResourceSpans().AppendEmpty()
			_, err := sp.processTraces(context.Background(), td)
			require.NoError(t, err)

			buf := reporter.TraceBufferFor(processorID.String())
			if tc.wantSaved {
				assert.NotNil(t, buf, "expected a buffer to exist after Save")
			} else {
				assert.Nil(t, buf, "expected no buffer when disabled")
			}
		})
	}
}

func Test_processLogs(t *testing.T) {
	cases := []struct {
		name      string
		enabled   bool
		wantSaved bool
	}{
		{name: "disabled", enabled: false, wantSaved: false},
		{name: "enabled", enabled: true, wantSaved: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reporter := report.NewSnapshotReporter(nil)
			processorID := component.MustNewIDWithName("snapshotprocessor", "x")

			sp := &snapshotProcessor{
				logger:      zap.NewNop(),
				enabled:     tc.enabled,
				snapShotter: reporter,
				processorID: processorID,
			}

			ld := plog.NewLogs()
			ld.ResourceLogs().AppendEmpty()
			_, err := sp.processLogs(context.Background(), ld)
			require.NoError(t, err)

			buf := reporter.LogBufferFor(processorID.String())
			if tc.wantSaved {
				assert.NotNil(t, buf)
			} else {
				assert.Nil(t, buf)
			}
		})
	}
}

func Test_processMetrics(t *testing.T) {
	cases := []struct {
		name      string
		enabled   bool
		wantSaved bool
	}{
		{name: "disabled", enabled: false, wantSaved: false},
		{name: "enabled", enabled: true, wantSaved: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reporter := report.NewSnapshotReporter(nil)
			processorID := component.MustNewIDWithName("snapshotprocessor", "x")

			sp := &snapshotProcessor{
				logger:      zap.NewNop(),
				enabled:     tc.enabled,
				snapShotter: reporter,
				processorID: processorID,
			}

			md := pmetric.NewMetrics()
			md.ResourceMetrics().AppendEmpty()
			_, err := sp.processMetrics(context.Background(), md)
			require.NoError(t, err)

			buf := reporter.MetricBufferFor(processorID.String())
			if tc.wantSaved {
				assert.NotNil(t, buf)
			} else {
				assert.Nil(t, buf)
			}
		})
	}
}

// overwriteSnapshotSet replaces the package-level getSnapshotReporter
// for the duration of a test and returns a function that restores it.
func overwriteSnapshotSet(t *testing.T, reporter *report.SnapshotReporter) (restore func()) {
	t.Helper()
	old := getSnapshotReporter
	getSnapshotReporter = func() *report.SnapshotReporter { return reporter }
	return func() { getSnapshotReporter = old }
}
