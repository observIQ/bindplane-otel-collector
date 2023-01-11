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

package metricextractprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

func TestProcessorStart(t *testing.T) {
	processor := &extractProcessor{}
	err := processor.Start(context.Background(), nil)
	require.NoError(t, err)
}

func TestProcessorShutdown(t *testing.T) {
	processor := &extractProcessor{}
	err := processor.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestProcessorCapabilities(t *testing.T) {
	processor := &extractProcessor{}
	require.Equal(t, consumer.Capabilities{MutatesData: false}, processor.Capabilities())
}

func TestProcessorExtractMetrics(t *testing.T) {
	var testCases = []struct {
		name    string
		cfg     *Config
		logs    plog.Logs
		metrics pmetric.Metrics
	}{
		{
			name: "no match",
			cfg: &Config{
				Match:      "false",
				Extract:    "body",
				Attributes: map[string]string{},
			},
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				resourceLogs := logs.ResourceLogs().AppendEmpty()
				resourceLogs.Resource().Attributes().FromRaw(map[string]any{"host": "test"})

				record := resourceLogs.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{"value": 20})
				record.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)))
				record.Attributes().FromRaw(map[string]any{"service": "test-service"})

				return logs
			}(),
			metrics: pmetric.NewMetrics(),
		},
		{
			name: "no extract",
			cfg: &Config{
				Match:      "true",
				Extract:    "body.missing",
				Attributes: map[string]string{},
			},
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				resourceLogs := logs.ResourceLogs().AppendEmpty()
				resourceLogs.Resource().Attributes().FromRaw(map[string]any{"host": "test"})

				record := resourceLogs.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{"value": 20})
				record.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)))
				record.Attributes().FromRaw(map[string]any{"service": "test-service"})

				return logs
			}(),
			metrics: pmetric.NewMetrics(),
		},
		{
			name: "invalid gauge double",
			cfg: &Config{
				Match:   "true",
				Extract: "body.value",
				Attributes: map[string]string{
					"service": "attributes.service",
				},
				MetricType: gaugeDoubleType,
				MetricName: "test.metric",
				MetricUnit: "unitless",
			},
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				resourceLogs := logs.ResourceLogs().AppendEmpty()
				resourceLogs.Resource().Attributes().FromRaw(map[string]any{"host": "test"})

				record := resourceLogs.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{"value": "test"})
				record.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)))
				record.Attributes().FromRaw(map[string]any{"service": "test-service"})

				return logs
			}(),
			metrics: pmetric.NewMetrics(),
		},
		{
			name: "invalid gauge int",
			cfg: &Config{
				Match:   "true",
				Extract: "body.value",
				Attributes: map[string]string{
					"service": "attributes.service",
				},
				MetricType: gaugeIntType,
				MetricName: "test.metric",
				MetricUnit: "unitless",
			},
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				resourceLogs := logs.ResourceLogs().AppendEmpty()
				resourceLogs.Resource().Attributes().FromRaw(map[string]any{"host": "test"})

				record := resourceLogs.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{"value": "test"})
				record.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)))
				record.Attributes().FromRaw(map[string]any{"service": "test-service"})

				return logs
			}(),
			metrics: pmetric.NewMetrics(),
		},
		{
			name: "valid gauge int",
			cfg: &Config{
				Match:   "true",
				Extract: "body.value",
				Attributes: map[string]string{
					"service": "attributes.service",
				},
				MetricType: gaugeIntType,
				MetricName: "test.metric",
				MetricUnit: "unitless",
			},
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				resourceLogs := logs.ResourceLogs().AppendEmpty()
				resourceLogs.Resource().Attributes().FromRaw(map[string]any{"host": "test"})

				record := resourceLogs.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{"value": 20})
				record.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)))
				record.Attributes().FromRaw(map[string]any{"service": "test-service"})

				return logs
			}(),
			metrics: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
				resourceMetrics.Resource().Attributes().FromRaw(map[string]any{"host": "test"})

				scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
				scopeMetrics.Scope().SetName(typeStr)

				metric := scopeMetrics.Metrics().AppendEmpty()
				metric.SetName("test.metric")
				metric.SetUnit("unitless")

				dataPoint := metric.SetEmptyGauge().DataPoints().AppendEmpty()
				dataPoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)))
				dataPoint.Attributes().FromRaw(map[string]any{"service": "test-service"})
				dataPoint.SetIntValue(20)

				return metrics
			}(),
		},
		{
			name: "valid gauge double",
			cfg: &Config{
				Match:   "true",
				Extract: "body.value",
				Attributes: map[string]string{
					"service": "attributes.service",
				},
				MetricType: gaugeDoubleType,
				MetricName: "test.metric",
				MetricUnit: "unitless",
			},
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				resourceLogs := logs.ResourceLogs().AppendEmpty()
				resourceLogs.Resource().Attributes().FromRaw(map[string]any{"host": "test"})

				record := resourceLogs.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{"value": 20.5})
				record.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)))
				record.Attributes().FromRaw(map[string]any{"service": "test-service"})

				return logs
			}(),
			metrics: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
				resourceMetrics.Resource().Attributes().FromRaw(map[string]any{"host": "test"})

				scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
				scopeMetrics.Scope().SetName(typeStr)

				metric := scopeMetrics.Metrics().AppendEmpty()
				metric.SetName("test.metric")
				metric.SetUnit("unitless")

				dataPoint := metric.SetEmptyGauge().DataPoints().AppendEmpty()
				dataPoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)))
				dataPoint.Attributes().FromRaw(map[string]any{"service": "test-service"})
				dataPoint.SetDoubleValue(20.5)

				return metrics
			}(),
		},
		{
			name: "valid counter int",
			cfg: &Config{
				Match:   "true",
				Extract: "body.value",
				Attributes: map[string]string{
					"service": "attributes.service",
				},
				MetricType: counterIntType,
				MetricName: "test.metric",
				MetricUnit: "unitless",
			},
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				resourceLogs := logs.ResourceLogs().AppendEmpty()
				resourceLogs.Resource().Attributes().FromRaw(map[string]any{"host": "test"})

				record := resourceLogs.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{"value": 20})
				record.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)))
				record.Attributes().FromRaw(map[string]any{"service": "test-service"})

				return logs
			}(),
			metrics: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
				resourceMetrics.Resource().Attributes().FromRaw(map[string]any{"host": "test"})

				scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
				scopeMetrics.Scope().SetName(typeStr)

				metric := scopeMetrics.Metrics().AppendEmpty()
				metric.SetName("test.metric")
				metric.SetUnit("unitless")

				dataPoint := metric.SetEmptySum().DataPoints().AppendEmpty()
				dataPoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)))
				dataPoint.Attributes().FromRaw(map[string]any{"service": "test-service"})
				dataPoint.SetIntValue(20)

				return metrics
			}(),
		},
		{
			name: "valid counter double",
			cfg: &Config{
				Match:   "true",
				Extract: "body.value",
				Attributes: map[string]string{
					"service": "attributes.service",
				},
				MetricType: counterDoubleType,
				MetricName: "test.metric",
				MetricUnit: "unitless",
			},
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				resourceLogs := logs.ResourceLogs().AppendEmpty()
				resourceLogs.Resource().Attributes().FromRaw(map[string]any{"host": "test"})

				record := resourceLogs.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				record.Body().SetEmptyMap().FromRaw(map[string]any{"value": 20.5})
				record.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)))
				record.Attributes().FromRaw(map[string]any{"service": "test-service"})

				return logs
			}(),
			metrics: func() pmetric.Metrics {
				metrics := pmetric.NewMetrics()
				resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
				resourceMetrics.Resource().Attributes().FromRaw(map[string]any{"host": "test"})

				scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
				scopeMetrics.Scope().SetName(typeStr)

				metric := scopeMetrics.Metrics().AppendEmpty()
				metric.SetName("test.metric")
				metric.SetUnit("unitless")

				dataPoint := metric.SetEmptySum().DataPoints().AppendEmpty()
				dataPoint.SetTimestamp(pcommon.NewTimestampFromTime(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)))
				dataPoint.Attributes().FromRaw(map[string]any{"service": "test-service"})
				dataPoint.SetDoubleValue(20.5)

				return metrics
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory := NewFactory()
			p, err := factory.CreateLogsProcessor(context.Background(), processor.CreateSettings{TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()}}, tc.cfg, nil)
			require.NoError(t, err)

			processor := p.(*extractProcessor)
			metrics := processor.extractMetrics(tc.logs)
			require.Equal(t, tc.metrics, metrics)
		})
	}
}
