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

package report

import (
	"errors"
	"net/http"
	"testing"

	"github.com/observiq/bindplane-otel-collector/internal/report/mocks"
	"github.com/observiq/bindplane-otel-collector/internal/report/snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestNewSnapshotReporter(t *testing.T) {
	client := mocks.NewMockClient(t)

	reporter := NewSnapshotReporter(client)

	require.Equal(t, client, reporter.client)
	require.Equal(t, 100, reporter.idealPayloadSize)
	require.NotNil(t, reporter.logBuffers)
	require.NotNil(t, reporter.metricBuffers)
	require.NotNil(t, reporter.traceBuffers)
}

func TestSnapshotReporterKind(t *testing.T) {
	reporter := NewSnapshotReporter(nil)
	kind := reporter.Kind()
	require.Equal(t, snapShotKind, kind)
}

func TestSnapshotReporterReset(t *testing.T) {
	reporter := NewSnapshotReporter(nil)
	componentID := "snapshot"
	reporter.logBuffers[componentID] = snapshot.NewLogBuffer(reporter.idealPayloadSize)
	reporter.traceBuffers[componentID] = snapshot.NewTraceBuffer(reporter.idealPayloadSize)
	reporter.metricBuffers[componentID] = snapshot.NewMetricBuffer(reporter.idealPayloadSize)

	reporter.Reset()

	require.Len(t, reporter.logBuffers, 0)
	require.Len(t, reporter.traceBuffers, 0)
	require.Len(t, reporter.metricBuffers, 0)
}

func TestSnapshotReporterSaveLogs(t *testing.T) {
	componentID := "snapshot/one"

	reporter := NewSnapshotReporter(nil)

	toAdd := plog.NewLogs()
	rl := toAdd.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty()

	reporter.SaveLogs(componentID, toAdd)

	buffer, ok := reporter.logBuffers[componentID]
	require.True(t, ok)
	require.Equal(t, 1, buffer.Len())
}

func TestSnapshotReporterSaveTraces(t *testing.T) {
	componentID := "snapshot/one"

	reporter := NewSnapshotReporter(nil)

	toAdd := ptrace.NewTraces()
	rl := toAdd.ResourceSpans().AppendEmpty()
	sl := rl.ScopeSpans().AppendEmpty()
	sl.Spans().AppendEmpty()

	reporter.SaveTraces(componentID, toAdd)

	buffer, ok := reporter.traceBuffers[componentID]
	require.True(t, ok)
	require.Equal(t, 1, buffer.Len())
}

func TestSnapshotReporterSaveMetrics(t *testing.T) {
	componentID := "snapshot/one"

	reporter := NewSnapshotReporter(nil)

	toAdd := pmetric.NewMetrics()
	rm := toAdd.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetEmptyGauge()
	metric.Gauge().DataPoints().AppendEmpty()

	reporter.SaveMetrics(componentID, toAdd)

	buffer, ok := reporter.metricBuffers[componentID]
	require.True(t, ok)
	require.Equal(t, 1, buffer.Len())
}

func TestSnapshotReporterReport(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Non-snapshot config passed",
			testFunc: func(t *testing.T) {
				notCfg := &struct{}{}

				reporter := NewSnapshotReporter(nil)
				err := reporter.Report(notCfg)
				assert.ErrorContains(t, err, "invalid config type")
			},
		},
		{
			desc: "Client returns error",
			testFunc: func(t *testing.T) {
				cfg := &snapshotConfig{
					Endpoint: &endpointConfig{
						URL:     "http://someurl:9001",
						Headers: map[string][]string{},
					},
					Processor:    "snapshot",
					PipelineType: "logs",
				}

				client := mocks.NewMockClient(t)
				client.On("Do", mock.Anything).Return(nil, errors.New("bad")).Run(func(args mock.Arguments) {
					req := args.Get(0).(*http.Request)

					assert.Equal(t, http.MethodPost, req.Method)
					assert.Equal(t, "someurl:9001", req.URL.Host)
					assert.Equal(t, "application/protobuf", req.Header.Get("Content-Type"))
					assert.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
					assert.Equal(t, cfg.Processor, req.Header.Get("Component-ID"))
				})

				reporter := NewSnapshotReporter(client)
				err := reporter.Report(cfg)
				assert.ErrorContains(t, err, "snapshot request failed: bad")
			},
		},
		{
			desc: "Non-200 response",
			testFunc: func(t *testing.T) {
				cfg := &snapshotConfig{
					Endpoint: &endpointConfig{
						URL:     "http://someurl:9001",
						Headers: map[string][]string{},
					},
					Processor:    "snapshot",
					PipelineType: "logs",
				}

				resp := &http.Response{
					StatusCode: http.StatusBadRequest,
				}

				client := mocks.NewMockClient(t)
				client.On("Do", mock.Anything).Return(resp, nil).Run(func(args mock.Arguments) {
					req := args.Get(0).(*http.Request)

					assert.Equal(t, http.MethodPost, req.Method)
					assert.Equal(t, "someurl:9001", req.URL.Host)
					assert.Equal(t, "application/protobuf", req.Header.Get("Content-Type"))
					assert.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
					assert.Equal(t, cfg.Processor, req.Header.Get("Component-ID"))
				})

				reporter := NewSnapshotReporter(client)
				err := reporter.Report(cfg)
				assert.ErrorContains(t, err, "non-200 response")
			},
		},
		{
			desc: "Valid logs report, no snapshot",
			testFunc: func(t *testing.T) {
				cfg := &snapshotConfig{
					Endpoint: &endpointConfig{
						URL: "http://someurl:9001",
						Headers: map[string][]string{
							"test": {"value"},
						},
					},
					Processor:    "snapshot",
					PipelineType: "logs",
				}

				resp := &http.Response{
					StatusCode: http.StatusOK,
				}

				client := mocks.NewMockClient(t)
				client.On("Do", mock.Anything).Return(resp, nil).Run(func(args mock.Arguments) {
					req := args.Get(0).(*http.Request)

					assert.Equal(t, http.MethodPost, req.Method)
					assert.Equal(t, "someurl:9001", req.URL.Host)
					assert.Equal(t, "application/protobuf", req.Header.Get("Content-Type"))
					assert.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
					assert.Equal(t, cfg.Processor, req.Header.Get("Component-ID"))
					assert.Equal(t, "value", req.Header.Get("test"))
				})

				reporter := NewSnapshotReporter(client)
				err := reporter.Report(cfg)
				assert.NoError(t, err)
			},
		},
		{
			desc: "Valid trace report, no snapshot",
			testFunc: func(t *testing.T) {
				cfg := &snapshotConfig{
					Endpoint: &endpointConfig{
						URL: "http://someurl:9001",
						Headers: map[string][]string{
							"test": {"value"},
						},
					},
					Processor:    "snapshot",
					PipelineType: "traces",
				}

				resp := &http.Response{
					StatusCode: http.StatusOK,
				}

				client := mocks.NewMockClient(t)
				client.On("Do", mock.Anything).Return(resp, nil).Run(func(args mock.Arguments) {
					req := args.Get(0).(*http.Request)

					assert.Equal(t, http.MethodPost, req.Method)
					assert.Equal(t, "someurl:9001", req.URL.Host)
					assert.Equal(t, "application/protobuf", req.Header.Get("Content-Type"))
					assert.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
					assert.Equal(t, cfg.Processor, req.Header.Get("Component-ID"))
					assert.Equal(t, "value", req.Header.Get("test"))
				})

				reporter := NewSnapshotReporter(client)
				err := reporter.Report(cfg)
				assert.NoError(t, err)
			},
		},
		{
			desc: "Valid metrics report, no snapshot",
			testFunc: func(t *testing.T) {
				cfg := &snapshotConfig{
					Endpoint: &endpointConfig{
						URL: "http://someurl:9001",
						Headers: map[string][]string{
							"test": {"value"},
						},
					},
					Processor:    "snapshot",
					PipelineType: "metrics",
				}

				resp := &http.Response{
					StatusCode: http.StatusOK,
				}

				client := mocks.NewMockClient(t)
				client.On("Do", mock.Anything).Return(resp, nil).Run(func(args mock.Arguments) {
					req := args.Get(0).(*http.Request)

					assert.Equal(t, http.MethodPost, req.Method)
					assert.Equal(t, "someurl:9001", req.URL.Host)
					assert.Equal(t, "application/protobuf", req.Header.Get("Content-Type"))
					assert.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
					assert.Equal(t, cfg.Processor, req.Header.Get("Component-ID"))
					assert.Equal(t, "value", req.Header.Get("test"))
				})

				reporter := NewSnapshotReporter(client)
				err := reporter.Report(cfg)
				assert.NoError(t, err)
			},
		},
		{
			desc: "Valid logs report, snapshot",
			testFunc: func(t *testing.T) {
				cfg := &snapshotConfig{
					Endpoint: &endpointConfig{
						URL: "http://someurl:9001",
						Headers: map[string][]string{
							"test": {"value"},
						},
					},
					Processor:    "snapshot",
					PipelineType: "logs",
				}

				resp := &http.Response{
					StatusCode: http.StatusOK,
				}

				client := mocks.NewMockClient(t)
				client.On("Do", mock.Anything).Return(resp, nil).Run(func(args mock.Arguments) {
					req := args.Get(0).(*http.Request)

					assert.Equal(t, http.MethodPost, req.Method)
					assert.Equal(t, "someurl:9001", req.URL.Host)
					assert.Equal(t, "application/protobuf", req.Header.Get("Content-Type"))
					assert.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
					assert.Equal(t, cfg.Processor, req.Header.Get("Component-ID"))
					assert.Equal(t, "value", req.Header.Get("test"))
				})

				reporter := NewSnapshotReporter(client)

				// save a snapshot
				toAdd := plog.NewLogs()
				rl := toAdd.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()
				sl.LogRecords().AppendEmpty()
				reporter.SaveLogs(cfg.Processor, toAdd)

				err := reporter.Report(cfg)
				assert.NoError(t, err)
			},
		},
		{
			desc: "Valid trace report, snapshot",
			testFunc: func(t *testing.T) {
				cfg := &snapshotConfig{
					Endpoint: &endpointConfig{
						URL: "http://someurl:9001",
						Headers: map[string][]string{
							"test": {"value"},
						},
					},
					Processor:    "snapshot",
					PipelineType: "traces",
				}

				resp := &http.Response{
					StatusCode: http.StatusOK,
				}

				client := mocks.NewMockClient(t)
				client.On("Do", mock.Anything).Return(resp, nil).Run(func(args mock.Arguments) {
					req := args.Get(0).(*http.Request)

					assert.Equal(t, http.MethodPost, req.Method)
					assert.Equal(t, "someurl:9001", req.URL.Host)
					assert.Equal(t, "application/protobuf", req.Header.Get("Content-Type"))
					assert.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
					assert.Equal(t, cfg.Processor, req.Header.Get("Component-ID"))
					assert.Equal(t, "value", req.Header.Get("test"))
				})

				reporter := NewSnapshotReporter(client)

				// save a snapshot
				toAdd := ptrace.NewTraces()
				rl := toAdd.ResourceSpans().AppendEmpty()
				sl := rl.ScopeSpans().AppendEmpty()
				sl.Spans().AppendEmpty()
				reporter.SaveTraces(cfg.Processor, toAdd)

				err := reporter.Report(cfg)
				assert.NoError(t, err)
			},
		},
		{
			desc: "Valid metrics report, snapshot",
			testFunc: func(t *testing.T) {
				cfg := &snapshotConfig{
					Endpoint: &endpointConfig{
						URL: "http://someurl:9001",
						Headers: map[string][]string{
							"test": {"value"},
						},
					},
					Processor:    "snapshot",
					PipelineType: "metrics",
				}

				resp := &http.Response{
					StatusCode: http.StatusOK,
				}

				client := mocks.NewMockClient(t)
				client.On("Do", mock.Anything).Return(resp, nil).Run(func(args mock.Arguments) {
					req := args.Get(0).(*http.Request)

					assert.Equal(t, http.MethodPost, req.Method)
					assert.Equal(t, "someurl:9001", req.URL.Host)
					assert.Equal(t, "application/protobuf", req.Header.Get("Content-Type"))
					assert.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
					assert.Equal(t, cfg.Processor, req.Header.Get("Component-ID"))
					assert.Equal(t, "value", req.Header.Get("test"))
				})

				reporter := NewSnapshotReporter(client)

				// save a snapshot
				toAdd := pmetric.NewMetrics()
				rm := toAdd.ResourceMetrics().AppendEmpty()
				sm := rm.ScopeMetrics().AppendEmpty()
				metric := sm.Metrics().AppendEmpty()
				metric.SetEmptyGauge()
				metric.Gauge().DataPoints().AppendEmpty()
				reporter.SaveMetrics(cfg.Processor, toAdd)

				err := reporter.Report(cfg)
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
