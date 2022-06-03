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

package resourceattributetransposerprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type metricsProcessor struct {
	consumer consumer.Metrics
	logger   *zap.Logger
	config   *Config
}

// newMetricsProcessor returns a new resourceToMetricsAttributesProcessor
func newMetricsProcessor(logger *zap.Logger, consumer consumer.Metrics, config *Config) *metricsProcessor {
	return &metricsProcessor{
		consumer: consumer,
		logger:   logger,
		config:   config,
	}
}

// Start starts the processor. It's a noop.
func (metricsProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Capabilities returns the consumer's capabilities. Indicates that this processor mutates the incoming metrics.
func (metricsProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// ConsumeMetrics processes the incoming pdata.Metrics.
func (p metricsProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	resMetrics := md.ResourceMetrics()
	for i := 0; i < resMetrics.Len(); i++ {
		resMetric := resMetrics.At(i)
		resourceAttrs := resMetric.Resource().Attributes()
		for _, op := range p.config.Operations {
			resourceValue, ok := resourceAttrs.Get(op.From)
			if !ok {
				continue
			}

			ilms := resMetric.ScopeMetrics()
			for j := 0; j < ilms.Len(); j++ {
				ilm := ilms.At(j)
				metrics := ilm.Metrics()
				for k := 0; k < metrics.Len(); k++ {
					metric := metrics.At(k)
					setMetricAttr(metric, op.To, resourceValue)
				}
			}
		}
	}
	return p.consumer.ConsumeMetrics(ctx, md)
}

// Shutdown stops the processor. It's a noop.
func (metricsProcessor) Shutdown(_ context.Context) error {
	return nil
}

// setMetricAttr sets the attribute (attrName) to the given value for every datapoint in the metric
func setMetricAttr(metric pmetric.Metric, attrName string, value pcommon.Value) {
	switch metric.DataType() {
	case pmetric.MetricDataTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			dp.Attributes().Insert(attrName, value)
		}

	case pmetric.MetricDataTypeHistogram:
		dps := metric.Histogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			dp.Attributes().Insert(attrName, value)
		}
	case pmetric.MetricDataTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			dp.Attributes().Insert(attrName, value)
		}
	case pmetric.MetricDataTypeSummary:
		dps := metric.Summary().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			dp.Attributes().Insert(attrName, value)
		}
	default:
		// skip metric if None or unknown type
	}
}
