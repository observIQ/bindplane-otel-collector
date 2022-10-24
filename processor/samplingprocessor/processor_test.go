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

package samplingprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func Test_processTraces(t *testing.T) {
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()

	testCases := []struct {
		desc      string
		dropRatio float64
		input     ptrace.Traces
		expected  ptrace.Traces
	}{
		{
			desc:      "Always Drop",
			dropRatio: 1.0,
			input:     td,
			expected:  ptrace.NewTraces(),
		},
		{
			desc:      "Never Drop",
			dropRatio: 0.0,
			input:     td,
			expected:  td,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := &Config{
				DropRatio: tc.dropRatio,
			}

			processor := newSamplingProcessor(zap.NewNop(), cfg)
			actual, err := processor.processTraces(context.Background(), tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func Test_processLogs(t *testing.T) {
	ld := plog.NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()

	testCases := []struct {
		desc      string
		dropRatio float64
		input     plog.Logs
		expected  plog.Logs
	}{
		{
			desc:      "Always Drop",
			dropRatio: 1.0,
			input:     ld,
			expected:  plog.NewLogs(),
		},
		{
			desc:      "Never Drop",
			dropRatio: 0.0,
			input:     ld,
			expected:  ld,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := &Config{
				DropRatio: tc.dropRatio,
			}

			processor := newSamplingProcessor(zap.NewNop(), cfg)
			actual, err := processor.processLogs(context.Background(), tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func Test_processMetrics(t *testing.T) {
	md := pmetric.NewMetrics()
	metric := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetEmptyGauge()
	metric.Gauge().DataPoints().AppendEmpty()

	testCases := []struct {
		desc      string
		dropRatio float64
		input     pmetric.Metrics
		expected  pmetric.Metrics
	}{
		{
			desc:      "Always Drop",
			dropRatio: 1.0,
			input:     md,
			expected:  pmetric.NewMetrics(),
		},
		{
			desc:      "Never Drop",
			dropRatio: 0.0,
			input:     md,
			expected:  md,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := &Config{
				DropRatio: tc.dropRatio,
			}

			processor := newSamplingProcessor(zap.NewNop(), cfg)
			actual, err := processor.processMetrics(context.Background(), tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}
