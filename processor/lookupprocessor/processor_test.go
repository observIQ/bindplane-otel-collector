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

package lookupprocessor

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func TestProcessLogs(t *testing.T) {
	testCases := []struct {
		name        string
		context     string
		field       string
		csvContents map[string]any
		createLogs  func() plog.Logs
		validate    func(t *testing.T, results plog.Logs, err error)
	}{
		{
			name:    "logs with resource context",
			context: resourceContext,
			field:   "ip",
			csvContents: map[string]any{
				"ip":     "0.0.0.0",
				"env":    "prod",
				"region": "us-west",
			},
			createLogs: func() plog.Logs {
				ld := plog.NewLogs()
				resourceLogs := ld.ResourceLogs().AppendEmpty()
				resourceLogs.Resource().Attributes().PutStr("ip", "0.0.0.0")
				return ld
			},
			validate: func(t *testing.T, results plog.Logs, err error) {
				require.NoError(t, err)
				resourceOutput := results.ResourceLogs().At(0)
				attrs := resourceOutput.Resource().Attributes().AsRaw()
				require.Equal(t, "0.0.0.0", attrs["ip"])
				require.Equal(t, "prod", attrs["env"])
				require.Equal(t, "us-west", attrs["region"])
			},
		},
		{
			name:    "logs with attributes context",
			context: attributesContext,
			field:   "ip",
			csvContents: map[string]any{
				"ip":     "0.0.0.0",
				"env":    "prod",
				"region": "us-west",
			},
			createLogs: func() plog.Logs {
				ld := plog.NewLogs()
				resourceLogs := ld.ResourceLogs().AppendEmpty()
				scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
				logs := scopeLogs.LogRecords().AppendEmpty()
				logs.Attributes().PutStr("ip", "0.0.0.0")
				return ld
			},
			validate: func(t *testing.T, results plog.Logs, err error) {
				require.NoError(t, err)
				resourceOutput := results.ResourceLogs().At(0)
				scopeOutput := resourceOutput.ScopeLogs().At(0)
				logs := scopeOutput.LogRecords().At(0)
				attrs := logs.Attributes().AsRaw()
				require.Equal(t, "0.0.0.0", attrs["ip"])
				require.Equal(t, "prod", attrs["env"])
				require.Equal(t, "us-west", attrs["region"])
			},
		},
		{
			name:    "logs with body context",
			context: bodyContext,
			field:   "ip",
			csvContents: map[string]any{
				"ip":     "0.0.0.0",
				"env":    "prod",
				"region": "us-west",
			},
			createLogs: func() plog.Logs {
				ld := plog.NewLogs()
				resourceLogs := ld.ResourceLogs().AppendEmpty()
				scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
				logs := scopeLogs.LogRecords().AppendEmpty()
				body := logs.Body().SetEmptyMap()
				body.PutStr("ip", "0.0.0.0")
				return ld
			},
			validate: func(t *testing.T, results plog.Logs, err error) {
				require.NoError(t, err)
				resourceOutput := results.ResourceLogs().At(0)
				scopeOutput := resourceOutput.ScopeLogs().At(0)
				logs := scopeOutput.LogRecords().At(0)
				body := logs.Body().Map().AsRaw()
				require.Equal(t, "0.0.0.0", body["ip"])
				require.Equal(t, "prod", body["env"])
				require.Equal(t, "us-west", body["region"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			csvPath := createTestCSVFile(t, tc.csvContents)
			csvFile := NewCSVFile(csvPath, tc.field)
			err := csvFile.Load()
			require.NoError(t, err)

			processor := lookupProcessor{
				logger:  zap.NewNop(),
				csvFile: csvFile,
				context: tc.context,
				field:   tc.field,
			}

			results, err := processor.processLogs(nil, tc.createLogs())
			tc.validate(t, results, err)
		})
	}
}

func TestProcessTraces(t *testing.T) {
	testCases := []struct {
		name         string
		context      string
		field        string
		csvContents  map[string]any
		createTraces func() ptrace.Traces
		validate     func(t *testing.T, results ptrace.Traces, err error)
	}{
		{
			name:    "traces with resource context",
			context: resourceContext,
			field:   "ip",
			csvContents: map[string]any{
				"ip":     "0.0.0.0",
				"env":    "prod",
				"region": "us-west",
			},
			createTraces: func() ptrace.Traces {
				traces := ptrace.NewTraces()
				resourceSpans := traces.ResourceSpans().AppendEmpty()
				resourceSpans.Resource().Attributes().PutStr("ip", "0.0.0.0")
				return traces
			},
			validate: func(t *testing.T, results ptrace.Traces, err error) {
				require.NoError(t, err)
				resourceOutput := results.ResourceSpans().At(0)
				attrs := resourceOutput.Resource().Attributes().AsRaw()
				require.Equal(t, "0.0.0.0", attrs["ip"])
				require.Equal(t, "prod", attrs["env"])
				require.Equal(t, "us-west", attrs["region"])
			},
		},
		{
			name:    "traces with attributes context",
			context: attributesContext,
			field:   "ip",
			csvContents: map[string]any{
				"ip":     "0.0.0.0",
				"env":    "prod",
				"region": "us-west",
			},
			createTraces: func() ptrace.Traces {
				traces := ptrace.NewTraces()
				resourceSpans := traces.ResourceSpans().AppendEmpty()
				scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()
				spans := scopeSpans.Spans().AppendEmpty()
				spans.Attributes().PutStr("ip", "0.0.0.0")
				return traces
			},
			validate: func(t *testing.T, results ptrace.Traces, err error) {
				require.NoError(t, err)
				resourceOutput := results.ResourceSpans().At(0)
				scopeOutput := resourceOutput.ScopeSpans().At(0)
				spans := scopeOutput.Spans().At(0)
				attrs := spans.Attributes().AsRaw()
				require.Equal(t, "0.0.0.0", attrs["ip"])
				require.Equal(t, "prod", attrs["env"])
				require.Equal(t, "us-west", attrs["region"])
			},
		},
		{
			name:    "traces with body context",
			context: bodyContext,
			field:   "ip",
			csvContents: map[string]any{
				"ip":     "0.0.0.0",
				"env":    "prod",
				"region": "us-west",
			},
			createTraces: func() ptrace.Traces {
				return ptrace.NewTraces()
			},
			validate: func(t *testing.T, _ ptrace.Traces, err error) {
				require.Error(t, err)
				require.ErrorIs(t, err, errInvalidContext)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			csvPath := createTestCSVFile(t, tc.csvContents)
			csvFile := NewCSVFile(csvPath, tc.field)
			err := csvFile.Load()
			require.NoError(t, err)

			processor := lookupProcessor{
				logger:  zap.NewNop(),
				csvFile: csvFile,
				context: tc.context,
				field:   tc.field,
			}

			results, err := processor.processTraces(nil, tc.createTraces())
			tc.validate(t, results, err)
		})
	}
}

func TestProcessMetrics(t *testing.T) {
	testCases := []struct {
		name          string
		context       string
		field         string
		csvContents   map[string]any
		createMetrics func() pmetric.Metrics
		validate      func(t *testing.T, results pmetric.Metrics, err error)
	}{
		{
			name:    "metrics with resource context",
			context: resourceContext,
			field:   "ip",
			csvContents: map[string]any{
				"ip":     "0.0.0.0",
				"env":    "prod",
				"region": "us-west",
			},
			createMetrics: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
				resourceMetrics.Resource().Attributes().PutStr("ip", "0.0.0.0")
				return metrics
			},
			validate: func(t *testing.T, results pmetric.Metrics, err error) {
				require.NoError(t, err)
				resourceOutput := results.ResourceMetrics().At(0)
				attrs := resourceOutput.Resource().Attributes().AsRaw()
				require.Equal(t, "0.0.0.0", attrs["ip"])
				require.Equal(t, "prod", attrs["env"])
				require.Equal(t, "us-west", attrs["region"])
			},
		},
		{
			name:    "sum metrics with attributes context",
			context: attributesContext,
			field:   "ip",
			csvContents: map[string]any{
				"ip":     "0.0.0.0",
				"env":    "prod",
				"region": "us-west",
			},
			createMetrics: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
				scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
				metric := scopeMetrics.Metrics().AppendEmpty()
				sum := metric.SetEmptySum()
				datapoint := sum.DataPoints().AppendEmpty()
				datapoint.Attributes().PutStr("ip", "0.0.0.0")
				return metrics
			},
			validate: func(t *testing.T, results pmetric.Metrics, err error) {
				require.NoError(t, err)
				resourceOutput := results.ResourceMetrics().At(0)
				scopeOutput := resourceOutput.ScopeMetrics().At(0)
				metric := scopeOutput.Metrics().At(0)
				datapoint := metric.Sum().DataPoints().At(0)
				attrs := datapoint.Attributes().AsRaw()
				require.Equal(t, "0.0.0.0", attrs["ip"])
				require.Equal(t, "prod", attrs["env"])
				require.Equal(t, "us-west", attrs["region"])
			},
		},
		{
			name:    "gauge metrics with attributes context",
			context: attributesContext,
			field:   "ip",
			csvContents: map[string]any{
				"ip":     "0.0.0.0",
				"env":    "prod",
				"region": "us-west",
			},
			createMetrics: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
				scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
				metric := scopeMetrics.Metrics().AppendEmpty()
				gauge := metric.SetEmptyGauge()
				datapoint := gauge.DataPoints().AppendEmpty()
				datapoint.Attributes().PutStr("ip", "0.0.0.0")
				return metrics
			},
			validate: func(t *testing.T, results pmetric.Metrics, err error) {
				require.NoError(t, err)
				resourceOutput := results.ResourceMetrics().At(0)
				scopeOutput := resourceOutput.ScopeMetrics().At(0)
				metric := scopeOutput.Metrics().At(0)
				datapoint := metric.Gauge().DataPoints().At(0)
				attrs := datapoint.Attributes().AsRaw()
				require.Equal(t, "0.0.0.0", attrs["ip"])
				require.Equal(t, "prod", attrs["env"])
				require.Equal(t, "us-west", attrs["region"])
			},
		},
		{
			name:    "metrics with body context",
			context: bodyContext,
			field:   "ip",
			csvContents: map[string]any{
				"ip":     "0.0.0.0",
				"env":    "prod",
				"region": "us-west",
			},
			createMetrics: func() pmetric.Metrics {
				return pmetric.NewMetrics()
			},
			validate: func(t *testing.T, _ pmetric.Metrics, err error) {
				require.Error(t, err)
				require.ErrorIs(t, err, errInvalidContext)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			csvPath := createTestCSVFile(t, tc.csvContents)
			csvFile := NewCSVFile(csvPath, tc.field)
			err := csvFile.Load()
			require.NoError(t, err)

			processor := lookupProcessor{
				logger:  zap.NewNop(),
				csvFile: csvFile,
				context: tc.context,
				field:   tc.field,
			}

			results, err := processor.processMetrics(nil, tc.createMetrics())
			tc.validate(t, results, err)
		})
	}
}

func TestAddLookupValues(t *testing.T) {
	csvContents := map[string]any{
		"ip":     "0.0.0.0",
		"env":    "prod",
		"region": "us-west",
	}
	csvPath := createTestCSVFile(t, csvContents)
	csvFile := NewCSVFile(csvPath, "ip")
	err := csvFile.Load()
	require.NoError(t, err)

	sourceMap := pcommon.NewMap()
	err = sourceMap.FromRaw(map[string]any{
		"ip":   "0.0.0.0",
		"host": "localhost",
	})
	require.NoError(t, err)

	processor := lookupProcessor{
		logger:  zap.NewNop(),
		csvFile: csvFile,
		field:   "ip",
	}

	processor.addLookupValues(sourceMap)
	expectedMap := map[string]any{
		"ip":     "0.0.0.0",
		"env":    "prod",
		"region": "us-west",
		"host":   "localhost",
	}

	require.Equal(t, expectedMap, sourceMap.AsRaw())
}

func TestShutdownBeforeStart(t *testing.T) {
	processor := lookupProcessor{
		wg:     &sync.WaitGroup{},
		logger: zap.NewNop(),
	}
	require.NotPanics(t, func() {
		processor.shutdown(context.Background())
	})
}

// createTestCSVFile is a helper function to create a CSV file from a map
func createTestCSVFile(t *testing.T, contents map[string]any) string {
	tempDir := t.TempDir()
	filePath := tempDir + "/test.csv"
	file, err := os.Create(filePath)
	require.NoError(t, err)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := make([]string, 0, len(contents))
	for k := range contents {
		headers = append(headers, k)
	}

	err = writer.Write(headers)
	require.NoError(t, err)

	values := make([]string, 0, len(contents))
	for _, k := range headers {
		values = append(values, fmt.Sprintf("%v", contents[k]))
	}

	err = writer.Write(values)
	require.NoError(t, err)

	return filePath
}
