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
	"testing"

	"github.com/observiq/bindplane-otel-collector/internal/report"
	"github.com/observiq/bindplane-otel-collector/internal/report/snapshot/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func Test_newShapshotProcessor(t *testing.T) {
	reporter := &report.SnapshotReporter{}
	unsetFunc := overwriteSnapshotSet(t, reporter)
	defer unsetFunc()

	logger := zap.NewNop()
	cfg := &Config{
		Enabled: false,
	}

	processorID := "snapshotprocessor/one"

	expected := &snapshotProcessor{
		logger:      logger,
		enabled:     cfg.Enabled,
		snapShotter: reporter,
		processorID: processorID,
	}

	actual := newSnapshotProcessor(logger, cfg, processorID)
	assert.Equal(t, expected, actual)
}

func Test_processTraces(t *testing.T) {
	testCases := []struct {
		desc       string
		enabled    bool
		setupMocks func(*mocks.MockSnapshotter)
	}{
		{
			desc:       "disabled",
			enabled:    false,
			setupMocks: func(_ *mocks.MockSnapshotter) {},
		},
		{
			desc:    "enabled",
			enabled: true,
			setupMocks: func(m *mocks.MockSnapshotter) {
				m.On("SaveTraces", mock.Anything, mock.Anything).Return()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			mockSnapshotter := mocks.NewMockSnapshotter(t)

			tc.setupMocks(mockSnapshotter)

			sp := &snapshotProcessor{
				logger:      zap.NewNop(),
				enabled:     tc.enabled,
				snapShotter: mockSnapshotter,
				processorID: componentType.String(),
			}

			td := ptrace.NewTraces()
			sp.processTraces(context.Background(), td)
		})
	}
}

func Test_processLogs(t *testing.T) {
	testCases := []struct {
		desc       string
		enabled    bool
		setupMocks func(*mocks.MockSnapshotter)
	}{
		{
			desc:       "disabled",
			enabled:    false,
			setupMocks: func(_ *mocks.MockSnapshotter) {},
		},
		{
			desc:    "enabled",
			enabled: true,
			setupMocks: func(m *mocks.MockSnapshotter) {
				m.On("SaveLogs", mock.Anything, mock.Anything).Return()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			mockSnapshotter := mocks.NewMockSnapshotter(t)

			tc.setupMocks(mockSnapshotter)

			sp := &snapshotProcessor{
				logger:      zap.NewNop(),
				enabled:     tc.enabled,
				snapShotter: mockSnapshotter,
				processorID: componentType.String(),
			}

			ld := plog.NewLogs()
			sp.processLogs(context.Background(), ld)
		})
	}
}

func Test_processMetrics(t *testing.T) {
	testCases := []struct {
		desc       string
		enabled    bool
		setupMocks func(*mocks.MockSnapshotter)
	}{
		{
			desc:       "disabled",
			enabled:    false,
			setupMocks: func(_ *mocks.MockSnapshotter) {},
		},
		{
			desc:    "enabled",
			enabled: true,
			setupMocks: func(m *mocks.MockSnapshotter) {
				m.On("SaveMetrics", mock.Anything, mock.Anything).Return()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			mockSnapshotter := mocks.NewMockSnapshotter(t)

			tc.setupMocks(mockSnapshotter)

			sp := &snapshotProcessor{
				logger:      zap.NewNop(),
				enabled:     tc.enabled,
				snapShotter: mockSnapshotter,
				processorID: componentType.String(),
			}

			md := pmetric.NewMetrics()
			sp.processMetrics(context.Background(), md)
		})
	}
}

func overwriteSnapshotSet(t *testing.T, reporterToSet *report.SnapshotReporter) (unsetFunc func()) {
	t.Helper()
	// Save original function
	oldFunc := getSnapshotReporter

	// Create new function returning new reporter
	getSnapshotReporter = func() *report.SnapshotReporter {
		return reporterToSet
	}

	// Create a function to return to the original state
	unsetFunc = func() {
		getSnapshotReporter = oldFunc
	}

	return
}
