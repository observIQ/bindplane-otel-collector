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

package randomfailureprocessor

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestProcess_Logs(t *testing.T) {
	tests := []struct {
		name                  string
		failureRate           float64
		validateExpectedError func(t *testing.T, errCount int)
	}{
		{name: "failure_rate_1", failureRate: 1, validateExpectedError: func(t *testing.T, errCount int) {
			require.Equal(t, 1000, errCount)
		}},
		{name: "failure_rate_0", failureRate: 0, validateExpectedError: func(t *testing.T, errCount int) {
			require.Equal(t, 0, errCount)
		}},
		// technically flaky, but the odds of any one of these tests failing is basically statistically insignificant
		{name: "failure_rate_0.5", failureRate: 0.5, validateExpectedError: func(t *testing.T, errCount int) {
			require.False(t, errCount < 400 || errCount > 600, "expected error count to be between 400 and 600, got %d", errCount)
		}},
		{name: "failure_rate_0.8", failureRate: 0.8, validateExpectedError: func(t *testing.T, errCount int) {
			require.False(t, errCount < 700 || errCount > 900, "expected error count to be between 700 and 900, got %d", errCount)
		}},
		{name: "failure_rate_0.2", failureRate: 0.2, validateExpectedError: func(t *testing.T, errCount int) {
			require.False(t, errCount < 100 || errCount > 300, "expected error count to be between 100 and 300, got %d", errCount)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factory := NewFactory()
			sink := &consumertest.LogsSink{}

			config := factory.CreateDefaultConfig().(*Config)
			config.FailureRate = test.failureRate

			pSet := processortest.NewNopSettings(componentType)
			p, err := factory.CreateLogs(context.Background(), pSet, config, sink)
			require.NoError(t, err)

			require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, p.Shutdown(context.Background()))
			})

			l, err := golden.ReadLogs(filepath.Join("testdata", "logs", "w3c-logs.yaml"))
			require.NoError(t, err)

			errCount := 0
			for range 1000 {
				err = p.ConsumeLogs(context.Background(), l)
				if err != nil {
					errCount++
				}
			}

			test.validateExpectedError(t, errCount)
			require.Equal(t, 1000-errCount, len(sink.AllLogs()))
			if len(sink.AllLogs()) > 0 {
				require.Equal(t, l, sink.AllLogs()[0])
			}
		})
	}
}

func TestProcess_Metrics(t *testing.T) {
	tests := []struct {
		name                  string
		failureRate           float64
		validateExpectedError func(t *testing.T, errCount int)
	}{
		{name: "failure_rate_1", failureRate: 1, validateExpectedError: func(t *testing.T, errCount int) {
			require.Equal(t, 1000, errCount)
		}},
		{name: "failure_rate_0", failureRate: 0, validateExpectedError: func(t *testing.T, errCount int) {
			require.Equal(t, 0, errCount)
		}},
		// technically flaky, but the odds of any one of these tests failing is basically statistically insignificant
		{name: "failure_rate_0.5", failureRate: 0.5, validateExpectedError: func(t *testing.T, errCount int) {
			require.False(t, errCount < 400 || errCount > 600, "expected error count to be between 400 and 600, got %d", errCount)
		}},
		{name: "failure_rate_0.8", failureRate: 0.8, validateExpectedError: func(t *testing.T, errCount int) {
			require.False(t, errCount < 700 || errCount > 900, "expected error count to be between 700 and 900, got %d", errCount)
		}},
		{name: "failure_rate_0.2", failureRate: 0.2, validateExpectedError: func(t *testing.T, errCount int) {
			require.False(t, errCount < 100 || errCount > 300, "expected error count to be between 100 and 300, got %d", errCount)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factory := NewFactory()
			sink := &consumertest.MetricsSink{}

			config := factory.CreateDefaultConfig().(*Config)
			config.FailureRate = test.failureRate

			pSet := processortest.NewNopSettings(componentType)
			p, err := factory.CreateMetrics(context.Background(), pSet, config, sink)
			require.NoError(t, err)

			require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, p.Shutdown(context.Background()))
			})

			m, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
			require.NoError(t, err)

			errCount := 0
			for range 1000 {
				err = p.ConsumeMetrics(context.Background(), m)
				if err != nil {
					errCount++
				}
			}

			test.validateExpectedError(t, errCount)
			require.Equal(t, 1000-errCount, len(sink.AllMetrics()))
			if len(sink.AllMetrics()) > 0 {
				require.Equal(t, m, sink.AllMetrics()[0])
			}
		})
	}
}

func TestProcess_Traces(t *testing.T) {
	tests := []struct {
		name                  string
		failureRate           float64
		validateExpectedError func(t *testing.T, errCount int)
	}{
		{name: "failure_rate_1", failureRate: 1, validateExpectedError: func(t *testing.T, errCount int) {
			require.Equal(t, 1000, errCount)
		}},
		{name: "failure_rate_0", failureRate: 0, validateExpectedError: func(t *testing.T, errCount int) {
			require.Equal(t, 0, errCount)
		}},
		// technically flaky, but the odds of any one of these tests failing is basically statistically insignificant
		{name: "failure_rate_0.5", failureRate: 0.5, validateExpectedError: func(t *testing.T, errCount int) {
			require.False(t, errCount < 400 || errCount > 600, "expected error count to be between 400 and 600, got %d", errCount)
		}},
		{name: "failure_rate_0.8", failureRate: 0.8, validateExpectedError: func(t *testing.T, errCount int) {
			require.False(t, errCount < 700 || errCount > 900, "expected error count to be between 700 and 900, got %d", errCount)
		}},
		{name: "failure_rate_0.2", failureRate: 0.2, validateExpectedError: func(t *testing.T, errCount int) {
			require.False(t, errCount < 100 || errCount > 300, "expected error count to be between 100 and 300, got %d", errCount)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factory := NewFactory()
			sink := &consumertest.TracesSink{}

			config := factory.CreateDefaultConfig().(*Config)
			config.FailureRate = test.failureRate

			pSet := processortest.NewNopSettings(componentType)
			p, err := factory.CreateTraces(context.Background(), pSet, config, sink)
			require.NoError(t, err)

			require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				require.NoError(t, p.Shutdown(context.Background()))
			})

			tr, err := golden.ReadTraces(filepath.Join("testdata", "traces", "bindplane-traces.yaml"))
			require.NoError(t, err)

			errCount := 0
			for range 1000 {
				err = p.ConsumeTraces(context.Background(), tr)
				if err != nil {
					errCount++
				}
			}

			test.validateExpectedError(t, errCount)
			require.Equal(t, 1000-errCount, len(sink.AllTraces()))
			if len(sink.AllTraces()) > 0 {
				require.Equal(t, tr, sink.AllTraces()[0])
			}
		})
	}
}
