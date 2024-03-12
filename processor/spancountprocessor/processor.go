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

package spancountprocessor

import (
	"context"
	"sync"
	"time"

	"github.com/observiq/bindplane-agent/counter"
	"github.com/observiq/bindplane-agent/expr"
	"github.com/observiq/bindplane-agent/receiver/routereceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// spanCountProcessor is a processor that counts spans.
type spanCountProcessor struct {
	config    *Config
	match     *expr.Expression
	attrs     *expr.ExpressionMap
	OTTLmatch *expr.OTTLCondition[ottlspan.TransformContext]
	OTTLattrs *expr.OTTLAttributeMap[ottlspan.TransformContext]
	counter   *counter.TelemetryCounter
	consumer  consumer.Traces
	logger    *zap.Logger
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mux       sync.Mutex
}

// newProcessor returns a new processor.
func newExprProcessor(config *Config, consumer consumer.Traces, match *expr.Expression, attrs *expr.ExpressionMap, logger *zap.Logger) *spanCountProcessor {
	return &spanCountProcessor{
		config:   config,
		match:    match,
		attrs:    attrs,
		counter:  counter.NewTelemetryCounter(),
		consumer: consumer,
		logger:   logger,
	}
}

// newProcessor returns a new processor.
func newOTTLProcessor(
	config *Config,
	consumer consumer.Traces,
	match *expr.OTTLCondition[ottlspan.TransformContext],
	attrs *expr.OTTLAttributeMap[ottlspan.TransformContext],
	logger *zap.Logger) *spanCountProcessor {
	return &spanCountProcessor{
		config:    config,
		OTTLmatch: match,
		OTTLattrs: attrs,
		counter:   counter.NewTelemetryCounter(),
		consumer:  consumer,
		logger:    logger,
	}
}

func (p *spanCountProcessor) isOTTL() bool {
	return p.OTTLmatch != nil
}

// Start starts the processor.
func (p *spanCountProcessor) Start(_ context.Context, _ component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	p.wg.Add(1)
	go p.handleMetricInterval(ctx)

	return nil
}

// Capabilities returns the consumer's capabilities.
func (p *spanCountProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// Shutdown stops the processor.
func (p *spanCountProcessor) Shutdown(_ context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	return nil
}

// ConsumeMetrics processes the metrics.
func (p *spanCountProcessor) ConsumeTraces(ctx context.Context, t ptrace.Traces) error {
	p.mux.Lock()
	defer p.mux.Unlock()

	if p.isOTTL() {
		p.consumeTracesOTTL(ctx, t)
	} else {
		p.consumeTracesExpr(t)
	}

	return p.consumer.ConsumeTraces(ctx, t)
}

// consumeTracesOTTL processes the traces using configured OTTL expressions
func (p *spanCountProcessor) consumeTracesOTTL(ctx context.Context, t ptrace.Traces) {
	resourceSpans := t.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		resourceSpan := resourceSpans.At(i)
		resource := resourceSpan.Resource()
		scopeSpans := resourceSpan.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			scopeSpan := scopeSpans.At(j)
			spans := scopeSpan.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				spanCtx := ottlspan.NewTransformContext(span, scopeSpan.Scope(), resource)
				match, err := p.OTTLmatch.Match(ctx, spanCtx)
				if err != nil {
					p.logger.Error("Error while matching OTTL span", zap.Error(err))
					continue
				}

				if match {
					attrs := p.OTTLattrs.ExtractAttributes(ctx, spanCtx)
					p.counter.Add(resource.Attributes().AsRaw(), attrs)
				}
			}
		}
	}
}

// consumeTracesExpr processes the traces using configured Expr expressions
func (p *spanCountProcessor) consumeTracesExpr(t ptrace.Traces) {
	resourceGroups := expr.ConvertToSpanResourceGroups(t)
	for _, group := range resourceGroups {
		resource := group.Resource
		for _, span := range group.Spans {
			match, err := p.match.Match(span)
			if err != nil {
				p.logger.Error("Error while matching span", zap.Error(err))
				continue
			}

			if match {
				attrs := p.attrs.Extract(span)
				p.counter.Add(resource, attrs)
			}
		}
	}
}

// handleMetricInterval sends metrics at the configured interval.
func (p *spanCountProcessor) handleMetricInterval(ctx context.Context) {
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
func (p *spanCountProcessor) sendMetrics(ctx context.Context) {
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

// createMetrics creates metrics from the counter.
func (p *spanCountProcessor) createMetrics() pmetric.Metrics {
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

	return metrics
}
