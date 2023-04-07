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
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// metricCountProcessor is a processor that counts logs.
type metricCountProcessor struct {
	config   *Config
	match    *expr.Expression
	attrs    *expr.ExpressionMap
	counter  *counter.TelemetryCounter
	consumer consumer.Metrics
	logger   *zap.Logger
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mux      sync.Mutex
}

// newProcessor returns a new processor.
func newProcessor(config *Config, consumer consumer.Metrics, match *expr.Expression, attrs *expr.ExpressionMap, logger *zap.Logger) *metricCountProcessor {
	return &metricCountProcessor{
		config:   config,
		match:    match,
		attrs:    attrs,
		counter:  counter.NewTelemetryCounter(),
		consumer: consumer,
		logger:   logger,
	}
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

	return p.consumer.ConsumeMetrics(ctx, m)
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
