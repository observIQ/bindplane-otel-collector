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

package logcountprocessor

import (
	"context"
	"sync"
	"time"

	"github.com/observiq/observiq-otel-collector/receiver/routereceiver"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// processor is a processor that counts logs.
type processor struct {
	config    *Config
	matchExpr *Expression
	attrExprs map[string]*Expression
	counter   *LogCounter
	consumer  consumer.Logs
	logger    *zap.Logger
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mux       sync.Mutex
}

// newProcessor returns a new processor.
func newProcessor(config *Config, consumer consumer.Logs, matchExpr *Expression, attrExprs map[string]*Expression, logger *zap.Logger) *processor {
	return &processor{
		config:    config,
		matchExpr: matchExpr,
		attrExprs: attrExprs,
		counter:   NewLogCounter(),
		consumer:  consumer,
		logger:    logger,
	}
}

// Start starts the processor.
func (p *processor) Start(_ context.Context, _ component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	p.wg.Add(1)
	go p.handleMetricInterval(ctx)

	return nil
}

// Capabilities returns the consumer's capabilities.
func (p *processor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// Shutdown stops the processor.
func (p *processor) Shutdown(_ context.Context) error {
	p.cancel()
	p.wg.Wait()
	return nil
}

// ConsumeLogs processes the logs.
func (p *processor) ConsumeLogs(ctx context.Context, pl plog.Logs) error {
	p.mux.Lock()
	defer p.mux.Unlock()

	records := convertToRecords(pl)
	for _, record := range records {
		if p.matchRecord(record) {
			resource := p.extractResource(record)
			attrs := p.extractAttributes(record)
			p.counter.Add(resource, attrs)
		}
	}

	return p.consumer.ConsumeLogs(ctx, pl)
}

// handleMetricInterval sends metrics at the configured interval.
func (p *processor) handleMetricInterval(ctx context.Context) {
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
func (p *processor) sendMetrics(ctx context.Context) {
	p.mux.Lock()
	defer p.mux.Unlock()

	metrics := p.createMetrics()
	if metrics.ResourceMetrics().Len() == 0 {
		return
	}

	p.counter.Reset()

	if err := routereceiver.RouteMetrics(ctx, p.config.Route, metrics); err != nil {
		p.logger.Error("Failed to send metrics", zap.Error(err))
	}
}

// matchRecord returns true if the record matches the configured expression.
func (p *processor) matchRecord(record Record) bool {
	matches, err := p.matchExpr.Match(record)
	if err != nil {
		p.logger.Debug("Failed to evaluate match expression", zap.Error(err))
		return false
	}

	return matches
}

// extractAttributes extracts attributes from the record.
func (p *processor) extractAttributes(record Record) map[string]any {
	attrs := map[string]any{}
	for key, expression := range p.attrExprs {
		value, err := expression.Evaluate(record)
		if err != nil {
			p.logger.Debug("Failed to evaluate attribute expression", zap.Error(err))
			continue
		}
		attrs[key] = value
	}
	return attrs
}

// extractResource extracts the resource from the record.
func (p *processor) extractResource(record Record) map[string]any {
	value, ok := record[resourceField].(map[string]any)
	if !ok {
		return map[string]any{}
	}

	return value
}

// createMetrics creates metrics from the counter.
func (p *processor) createMetrics() pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	for _, resource := range p.counter.resources {
		resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
		err := resourceMetrics.Resource().Attributes().FromRaw(resource.values)
		if err != nil {
			p.logger.Error("Failed to set resource attributes", zap.Error(err))
		}

		scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
		scopeMetrics.Scope().SetName(typeStr)
		for _, attributes := range resource.attributes {
			metrics := scopeMetrics.Metrics().AppendEmpty()
			metrics.SetName(p.config.MetricName)
			metrics.SetUnit(p.config.MetricUnit)
			metrics.SetEmptyGauge()

			gauge := metrics.Gauge().DataPoints().AppendEmpty()
			gauge.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			gauge.SetIntValue(int64(attributes.count))
			err = gauge.Attributes().FromRaw(attributes.values)
			if err != nil {
				p.logger.Error("Failed to set metric attributes", zap.Error(err))
			}

		}
	}

	return metrics
}
