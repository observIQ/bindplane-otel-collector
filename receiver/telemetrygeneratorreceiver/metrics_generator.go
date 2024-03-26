// Copyright observIQ, Inc.
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

package telemetrygeneratorreceiver //import "github.com/observiq/bindplane-agent/receiver/telemetrygeneratorreceiver"

import (
	"math"
	"math/rand"

	"github.com/mitchellh/mapstructure"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// metricsGenerator is a generator for metrics. It generates a sampling of metrics
// based on the configuration provided, adjusting the values & times of the metrics
// with each generation.
type metricsGenerator struct {
	otlpGenerator
	// dataPointUpdaters is a list of functions that update the data points with new values
	// for each batch of metrics that are generated
	dataPointUpdaters []func()
}

// metric is a convenience struct for unmarshalling the metrics config
type metric struct {
	Name       string         `mapstructure:"name"`
	Type       string         `mapstructure:"type"`
	ValueMin   int64          `mapstructure:"value_min"`
	ValueMax   int64          `mapstructure:"value_max"`
	Unit       string         `mapstructure:"unit"`
	Attributes map[string]any `mapstructure:"attributes"`
}

func newMetricsGenerator(cfg GeneratorConfig, logger *zap.Logger) metricGenerator {

	g := &metricsGenerator{
		otlpGenerator: otlpGenerator{
			cfg:     cfg,
			logger:  logger,
			logs:    plog.NewLogs(),
			metrics: pmetric.NewMetrics(),
			traces:  ptrace.NewTraces(),
		},
	}

	newResource := g.metrics.ResourceMetrics().AppendEmpty()

	// Load up Resources attributes
	resourceMap := pcommon.NewMap()
	err := resourceMap.FromRaw(cfg.ResourceAttributes)
	if err != nil {
		// should be caught in validation
		logger.Error("Error setting resource attributes in host_metrics", zap.Error(err))
	} else {
		resourceMap.CopyTo(newResource.Resource().Attributes())
	}

	metrics := cfg.AdditionalConfig["metrics"].([]any)
	newScope := newResource.ScopeMetrics().AppendEmpty()

	for _, m := range metrics {
		var metric metric
		err := mapstructure.Decode(m, &metric)
		if err != nil {
			// this should be caught in validation
			logger.Error("Error decoding metric", zap.Error(err))
			continue
		}

		newMetric := newScope.Metrics().AppendEmpty()

		newMetric.SetUnit(metric.Unit)
		newMetric.SetName(metric.Name)

		// all the host metrics are either sums or gauges
		switch metric.Type {
		case "Gauge":
			newMetric.SetEmptyGauge()
			dp := newMetric.Gauge().DataPoints().AppendEmpty()

			dp.SetTimestamp(pcommon.NewTimestampFromTime(getCurrentTime()))
			dp.SetStartTimestamp(pcommon.NewTimestampFromTime(getCurrentTime()))

			err = dp.Attributes().FromRaw(metric.Attributes)
			if err != nil {
				// should be caught in validation
				logger.Error("Error setting attributes in host_metrics", zap.String("name", metric.Name), zap.Error(err))
			}

			// All the host metric Gauges are float64
			g.dataPointUpdaters = append(g.dataPointUpdaters, func() { dp.SetDoubleValue(getRandomFloat64(metric.ValueMin, metric.ValueMax)) })
		case "Sum":
			newMetric.SetEmptySum()
			dp := newMetric.Sum().DataPoints().AppendEmpty()

			dp.SetTimestamp(pcommon.NewTimestampFromTime(getCurrentTime()))
			dp.SetStartTimestamp(pcommon.NewTimestampFromTime(getCurrentTime()))

			err = dp.Attributes().FromRaw(metric.Attributes)
			if err != nil {
				// should be caught in validation
				logger.Error("Error setting attributes in host_metrics", zap.String("name", metric.Name), zap.Error(err))
			}

			switch metric.Unit {
			case "s":
				g.dataPointUpdaters = append(g.dataPointUpdaters, func() {
					dp.SetDoubleValue(float64(math.Trunc(getRandomFloat64(metric.ValueMin, metric.ValueMax)*100)) / 100)
				})
			default:
				g.dataPointUpdaters = append(g.dataPointUpdaters, func() { dp.SetIntValue(int64(getRandomFloat64(metric.ValueMin, metric.ValueMax))) })
			}
		}
	}

	g.adjustMetricTimes()

	return g
}

// this is a variable so that it can be overridden in tests
var getRandomFloat64 = func(value_min, value_max int64) float64 {
	// #nosec G404 - we don't need a cryptographically strong random number generator here
	return rand.Float64()*(float64(value_max)-float64(value_min)) + float64(value_min)
}

func (g *metricsGenerator) generateMetrics() pmetric.Metrics {
	// Update the data points with new values
	for _, updater := range g.dataPointUpdaters {
		updater()
	}
	return g.otlpGenerator.generateMetrics()
}
