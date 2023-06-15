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

package datapointcountprocessor

import (
	"context"
	"sync"
	"time"

	"github.com/observiq/observiq-otel-collector/counter"
	"github.com/observiq/observiq-otel-collector/expr"
	"github.com/observiq/observiq-otel-collector/receiver/routereceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// metricCountProcessor is a processor that counts logs.
type metricCountProcessor struct {
	config    *Config
	match     *expr.Expression
	attrs     *expr.ExpressionMap
	OTTLmatch *expr.OTTLCondition[ottldatapoint.TransformContext]
	OTTLattrs *expr.OTTLAttributeMap[ottldatapoint.TransformContext]
	counter   *counter.TelemetryCounter
	consumer  consumer.Metrics
	logger    *zap.Logger
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mux       sync.Mutex
}

// newExprProcessor returns a new processor with expr expressions.
func newExprProcessor(config *Config,
	consumer consumer.Metrics,
	match *expr.Expression,
	attrs *expr.ExpressionMap,
	logger *zap.Logger,
) *metricCountProcessor {
	return &metricCountProcessor{
		config:   config,
		match:    match,
		attrs:    attrs,
		counter:  counter.NewTelemetryCounter(),
		consumer: consumer,
		logger:   logger,
	}
}

// newOTTLProcessor returns a new processor with OTTL expressions
func newOTTLProcessor(config *Config,
	consumer consumer.Metrics,
	match *expr.OTTLCondition[ottldatapoint.TransformContext],
	attrs *expr.OTTLAttributeMap[ottldatapoint.TransformContext],
	logger *zap.Logger,
) *metricCountProcessor {
	return &metricCountProcessor{
		config:    config,
		OTTLmatch: match,
		OTTLattrs: attrs,
		counter:   counter.NewTelemetryCounter(),
		consumer:  consumer,
		logger:    logger,
	}
}

func (p *metricCountProcessor) isOTTL() bool {
	return p.OTTLmatch != nil
}

// Start starts the processor.
func (p *metricCountProcessor) Start(_ context.Context, _ component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	p.wg.Add(1)
	go p.handleMetricInterval(ctx)

	return nil
}

// Capabilities returns the consumer's capabilities.
func (p *metricCountProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// Shutdown stops the processor.
func (p *metricCountProcessor) Shutdown(_ context.Context) error {
	p.cancel()
	p.wg.Wait()
	return nil
}

// ConsumeMetrics processes the metrics.
func (p *metricCountProcessor) ConsumeMetrics(ctx context.Context, m pmetric.Metrics) error {
	p.mux.Lock()
	defer p.mux.Unlock()

	if p.isOTTL() {
		p.consumeMetricsOTTL(ctx, m)
	} else {
		p.consumeMetricsExpr(m)
	}

	return p.consumer.ConsumeMetrics(ctx, m)
}

// consumeMetricsOTTL processes the metrics using configured expr expressions
func (p *metricCountProcessor) consumeMetricsExpr(m pmetric.Metrics) {
	resourceGroups := expr.ConvertToDatapointResourceGroup(m)
	for _, group := range resourceGroups {
		resource := group.Resource
		for _, dp := range group.Datapoints {
			match, err := p.match.Match(dp)
			if err != nil {
				p.logger.Error("Error while matching datapoint", zap.Error(err))
				continue
			}

			if match {
				attrs := p.attrs.Extract(dp)
				p.counter.Add(resource, attrs)
			}
		}
	}
}

// consumeMetricsOTTL processes the metrics using configured OTTL expressions
func (p *metricCountProcessor) consumeMetricsOTTL(ctx context.Context, m pmetric.Metrics) {
	resourceMetrics := m.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		resourceMetric := resourceMetrics.At(i)
		resource := resourceMetric.Resource()
		scopeMetrics := resourceMetric.ScopeMetrics()
		for j := 0; j < scopeMetrics.Len(); j++ {
			scopeMetric := scopeMetrics.At(j)
			metrics := scopeMetric.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				eachDatapoint(metric, func(dp any) {
					tCtx := ottldatapoint.NewTransformContext(dp, metric, metrics, scopeMetric.Scope(), resource)
					match, err := p.OTTLmatch.Match(ctx, tCtx)
					if err != nil {
						p.logger.Error("Error while matching OTTL datapoint", zap.Error(err))
						return
					}

					if match {
						attrs := p.OTTLattrs.ExtractAttributes(ctx, tCtx)
						p.counter.Add(resource.Attributes().AsRaw(), attrs)
					}
				})
			}
		}
	}
}

// handleMetricInterval sends metrics at the configured interval.
func (p *metricCountProcessor) handleMetricInterval(ctx context.Context) {
	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.wg.Done()
			return
		case <-ticker.C:
			p.sendMetrics(ctx)
		}
	}
}

// sendMetrics sends metrics to the consumer.
func (p *metricCountProcessor) sendMetrics(ctx context.Context) {
	metrics := p.createMetrics()
	if metrics.ResourceMetrics().Len() == 0 {
		return
	}

	if err := routereceiver.RouteMetrics(ctx, p.config.Route, metrics); err != nil {
		p.logger.Error("Failed to send metrics", zap.Error(err))
	}
}

// createMetrics creates metrics from the counter. The counter is reset after the metrics are created.
func (p *metricCountProcessor) createMetrics() pmetric.Metrics {
	p.mux.Lock()
	defer p.mux.Unlock()

	metrics := pmetric.NewMetrics()
	for _, resource := range p.counter.Resources() {
		resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
		err := resourceMetrics.Resource().Attributes().FromRaw(resource.Values())
		if err != nil {
			p.logger.Error("Failed to set resource attributes", zap.Error(err))
		}

		scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
		scopeMetrics.Scope().SetName(typeStr)
		for _, attributes := range resource.Attributes() {
			metrics := scopeMetrics.Metrics().AppendEmpty()
			metrics.SetName(p.config.MetricName)
			metrics.SetUnit(p.config.MetricUnit)
			metrics.SetEmptyGauge()

			gauge := metrics.Gauge().DataPoints().AppendEmpty()
			gauge.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			gauge.SetIntValue(int64(attributes.Count()))
			err = gauge.Attributes().FromRaw(attributes.Values())
			if err != nil {
				p.logger.Error("Failed to set metric attributes", zap.Error(err))
			}

		}
	}

	p.counter.Reset()

	return metrics
}

// eachDatapoint calls the callback function f with each datapoint in the metric
func eachDatapoint(metric pmetric.Metric, f func(dp any)) {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			f(dp)
		}
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			f(dp)
		}
	case pmetric.MetricTypeHistogram:
		dps := metric.Histogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			f(dp)
		}
	case pmetric.MetricTypeExponentialHistogram:
		dps := metric.ExponentialHistogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			f(dp)
		}
	case pmetric.MetricTypeSummary:
		dps := metric.Summary().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			dp := dps.At(i)
			f(dp)
		}
	default:
		// skip anything else
	}
}
