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

	"github.com/observiq/observiq-otel-collector/receiver/routereceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestProcessorStart(t *testing.T) {
	processor := &exprExtractProcessor{}
	err := processor.Start(context.Background(), nil)
	require.NoError(t, err)
}

func TestProcessorShutdown(t *testing.T) {
	processor := &exprExtractProcessor{}
	err := processor.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestProcessorCapabilities(t *testing.T) {
	processor := &exprExtractProcessor{}
	require.Equal(t, consumer.Capabilities{MutatesData: false}, processor.Capabilities())
}

func TestProcessorExtractMetrics(t *testing.T) {
	var testCases = []struct {
		name      string
		cfg       *Config
		logs      plog.Logs
		metrics   pmetric.Metrics
		noMetrics bool
	}{
		{
			name: "no match",
			cfg: &Config{
				Match:      strp("false"),
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
			noMetrics: true,
		},
		{
			name: "no extract",
			cfg: &Config{
				Match:      strp("true"),
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
			noMetrics: true,
		},
		{
			name: "invalid gauge double",
			cfg: &Config{
				Match:   strp("true"),
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
			noMetrics: true,
		},
		{
			name: "invalid gauge int",
			cfg: &Config{
				Match:   strp("true"),
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
			noMetrics: true,
		},
		{
			name: "valid gauge int",
			cfg: &Config{
				Match:   strp("true"),
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
				Match:   strp("true"),
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
				Match:   strp("true"),
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
				Match:   strp("true"),
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
		{
			name: "OTTL no match",
			cfg: &Config{
				OTTLMatch:      strp("false"),
				OTTLExtract:    "body",
				OTTLAttributes: map[string]string{},
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
			noMetrics: true,
		},
		{
			name: "OTTL no extract",
			cfg: &Config{
				OTTLMatch:      strp("true"),
				OTTLExtract:    `body["dne"]`,
				OTTLAttributes: map[string]string{},
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
			noMetrics: true,
		},
		{
			name: "OTTL invalid gauge double",
			cfg: &Config{
				OTTLMatch:   strp("true"),
				OTTLExtract: `body["value"]`,
				OTTLAttributes: map[string]string{
					"service": `attributes["service"]`,
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
			noMetrics: true,
		},
		{
			name: "OTTL invalid gauge int",
			cfg: &Config{
				OTTLMatch:   strp("true"),
				OTTLExtract: `body["value"]`,
				OTTLAttributes: map[string]string{
					"service": `attributes["service"]`,
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
			noMetrics: true,
		},
		{
			name: "OTTL valid gauge int",
			cfg: &Config{
				OTTLMatch:   strp("true"),
				OTTLExtract: `body["value"]`,
				OTTLAttributes: map[string]string{
					"service": `attributes["service"]`,
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
			name: "OTTL valid gauge double",
			cfg: &Config{
				OTTLMatch:   strp("true"),
				OTTLExtract: `body["value"]`,
				OTTLAttributes: map[string]string{
					"service": `attributes["service"]`,
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
			name: "OTTL valid counter int",
			cfg: &Config{
				OTTLMatch:   strp("true"),
				OTTLExtract: `body["value"]`,
				OTTLAttributes: map[string]string{
					"service": `attributes["service"]`,
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
			name: "OTTL valid counter double",
			cfg: &Config{
				OTTLMatch:   strp("true"),
				OTTLExtract: `body["value"]`,
				OTTLAttributes: map[string]string{
					"service": `attributes["service"]`,
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

	routeReceiverName := "TestProcessorExtractMetrics"

	routeMetrics := &consumertest.MetricsSink{}
	createSettings := receivertest.NewNopCreateSettings()
	createSettings.ID = component.NewIDWithName(component.DataTypeMetrics, routeReceiverName)

	routereceiver.NewFactory().CreateMetricsReceiver(context.Background(), createSettings, routereceiver.Config{}, routeMetrics)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(routeMetrics.Reset)

			logSink := &consumertest.LogsSink{}
			tc.cfg.Route = routeReceiverName

			factory := NewFactory()

			p, err := factory.CreateLogsProcessor(context.Background(), processortest.NewNopCreateSettings(), tc.cfg, logSink)
			require.NoError(t, err)

			logsClone := plog.NewLogs()
			tc.logs.CopyTo(logsClone)

			err = p.ConsumeLogs(context.Background(), tc.logs)
			require.NoError(t, err)

			require.NoError(t, plogtest.CompareLogs(logsClone, logSink.AllLogs()[0]))

			metrics := routeMetrics.AllMetrics()
			if tc.noMetrics {
				require.Equal(t, 0, len(metrics))
			} else {
				require.Equal(t, 1, len(metrics))
				require.NoError(t, pmetrictest.CompareMetrics(tc.metrics, metrics[0]))

			}
		})
	}
}

func TestConvertAnyToInt(t *testing.T) {
	testCases := []struct {
		name     string
		value    any
		expected int64
		err      string
	}{
		{
			name:     "int",
			value:    int(10),
			expected: 10,
		},
		{
			name:     "int32",
			value:    int32(11),
			expected: 11,
		},
		{
			name:     "int64",
			value:    int64(12),
			expected: 12,
		},
		{
			name:     "float32",
			value:    float32(13),
			expected: 13,
		},
		{
			name:     "float64",
			value:    float64(14),
			expected: 14,
		},
		{
			name:     "string",
			value:    "15",
			expected: 15,
		},
		{
			name:  "string (invalid)",
			value: "not a number",
			err:   "failed to convert string to int:",
		},
		{
			name:  "invalid type",
			value: new(chan int),
			err:   "invalid value type:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := convertAnyToInt(tc.value)
			require.Equal(t, tc.expected, v)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}

		})
	}
}

func TestConvertAnyToFloat(t *testing.T) {
	testCases := []struct {
		name     string
		value    any
		expected float64
		err      string
	}{
		{
			name:     "int",
			value:    int(10),
			expected: 10,
		},
		{
			name:     "int32",
			value:    int32(11),
			expected: 11,
		},
		{
			name:     "int64",
			value:    int64(12),
			expected: 12,
		},
		{
			name:     "float32",
			value:    float32(13),
			expected: 13,
		},
		{
			name:     "float64",
			value:    float64(14),
			expected: 14,
		},
		{
			name:     "string",
			value:    "15",
			expected: 15,
		},
		{
			name:  "string (invalid)",
			value: "not a number",
			err:   "failed to convert string to float:",
		},
		{
			name:  "invalid type",
			value: new(chan int),
			err:   "invalid value type:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := convertAnyToFloat(tc.value)
			require.Equal(t, tc.expected, v)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
