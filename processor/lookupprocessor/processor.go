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

package lookupprocessor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// lookupProcessor is a lookupProcessor that looks up values and adds them to telemetry
type lookupProcessor struct {
	logger      *zap.Logger
	source      LookupSource
	context     string
	field       string
	cancel      context.CancelFunc
	wg          *sync.WaitGroup
	cfg         *Config
	componentID component.ID
}

// newLookupProcessor creates a new lookupProcessor
func newLookupProcessor(cfg *Config, componentID component.ID, logger *zap.Logger) (*lookupProcessor, error) {
	return &lookupProcessor{
		logger:      logger,
		context:     cfg.Context,
		field:       cfg.Field,
		wg:          &sync.WaitGroup{},
		cfg:         cfg,
		componentID: componentID,
	}, nil
}

// start starts the processor
func (p *lookupProcessor) start(ctx context.Context, host component.Host) error {
	var source LookupSource
	var err error

	// Determine which source to use
	switch {
	case p.cfg.Redis != nil && (p.cfg.SourceType == "" || p.cfg.SourceType == "redis"):
		source, err = NewRedisSource(p.cfg.Redis, p.logger)
	case p.cfg.API != nil && (p.cfg.SourceType == "api"):
		source, err = NewAPISource(p.cfg.API, p.logger)
	case p.cfg.CSV != "" || p.cfg.SourceType == "csv":
		// Default to CSV if no other source specified or explicitly set to csv
		source = NewCSVFile(p.cfg.CSV, p.cfg.Field)
	default:
		// Should have been caught by validation, but safe fallback
		if p.cfg.CSV != "" {
			source = NewCSVFile(p.cfg.CSV, p.cfg.Field)
		} else {
			return errMissingSource
		}
	}

	if err != nil {
		return fmt.Errorf("failed to create lookup source: %w", err)
	}

	// Wrap source with cache
	cacheTTL := 5 * time.Minute // default
	if p.cfg.CacheTTL > 0 {
		cacheTTL = p.cfg.CacheTTL
	}

	cachedSource, err := NewLookupCache(
		ctx,
		source,
		cacheTTL,
		p.cfg.CacheEnabled,
		p.cfg.StorageID,
		host,
		p.componentID,
		p.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create lookup cache: %w", err)
	}

	p.source = cachedSource

	backgroundCtx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	p.wg.Add(1)
	go p.loadSource(backgroundCtx)

	return nil
}

// shutdown stops the processor
func (p *lookupProcessor) shutdown(context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	
	if p.source != nil {
		return p.source.Close()
	}
	
	return nil
}

// loadSource loads the source into memory every minute until the context is canceled
func (p *lookupProcessor) loadSource(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	defer p.wg.Done()

	for {
		err := p.source.Load()
		if err != nil {
			p.logger.Error("failed to load source", zap.Error(err))
		} else {
			p.logger.Debug("source loaded/refreshed")
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

// processLogs processes incoming logs
func (p *lookupProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	switch p.context {
	case bodyContext:
		return p.processLogsWithBodyContext(ld)
	case attributesContext:
		return p.processLogsWithAttributesContext(ld)
	case resourceContext:
		return p.processLogsWithResourceContext(ld)
	default:
		return ld, errInvalidContext
	}
}

// processLogsWithResourceContext processes incoming logs with resource context
func (p *lookupProcessor) processLogsWithResourceContext(ld plog.Logs) (plog.Logs, error) {
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resource := ld.ResourceLogs().At(i)
		attrs := resource.Resource().Attributes()
		p.addLookupValues(attrs)
	}

	return ld, nil
}

// processLogsWithAttributesContext processes incoming logs with attributes context
func (p *lookupProcessor) processLogsWithAttributesContext(ld plog.Logs) (plog.Logs, error) {
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resource := ld.ResourceLogs().At(i)
		for j := 0; j < resource.ScopeLogs().Len(); j++ {
			scope := resource.ScopeLogs().At(j)
			for k := 0; k < scope.LogRecords().Len(); k++ {
				logs := scope.LogRecords().At(k)
				attrs := logs.Attributes()
				p.addLookupValues(attrs)
			}
		}
	}

	return ld, nil
}

// processLogsWithBodyContext processes incoming logs with body context
func (p *lookupProcessor) processLogsWithBodyContext(ld plog.Logs) (plog.Logs, error) {
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resource := ld.ResourceLogs().At(i)
		for j := 0; j < resource.ScopeLogs().Len(); j++ {
			scope := resource.ScopeLogs().At(j)
			for k := 0; k < scope.LogRecords().Len(); k++ {
				logs := scope.LogRecords().At(k)
				if logs.Body().Type() != pcommon.ValueTypeMap {
					continue
				}

				body := logs.Body().Map()
				p.addLookupValues(body)
			}
		}
	}

	return ld, nil
}

// processTraces processes incoming traces
func (p *lookupProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	switch p.context {
	case attributesContext:
		return p.processTracesWithAttributesContext(td)
	case resourceContext:
		return p.processTracesWithResourceContext(td)
	default:
		return td, errInvalidContext
	}
}

// processTracesWithResourceContext processes incoming traces with resource context
func (p *lookupProcessor) processTracesWithResourceContext(td ptrace.Traces) (ptrace.Traces, error) {
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resource := td.ResourceSpans().At(i)
		attrs := resource.Resource().Attributes()
		p.addLookupValues(attrs)
	}

	return td, nil
}

// processTracesWithAttributesContext processes incoming traces with attributes context
func (p *lookupProcessor) processTracesWithAttributesContext(td ptrace.Traces) (ptrace.Traces, error) {
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resource := td.ResourceSpans().At(i)
		for j := 0; j < resource.ScopeSpans().Len(); j++ {
			scope := resource.ScopeSpans().At(j)
			for k := 0; k < scope.Spans().Len(); k++ {
				spans := scope.Spans().At(k)
				attrs := spans.Attributes()
				p.addLookupValues(attrs)
			}
		}
	}

	return td, nil
}

// processMetrics processes incoming metrics
func (p *lookupProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	switch p.context {
	case attributesContext:
		return p.processMetricsWithAttributesContext(md)
	case resourceContext:
		return p.processMetricsWithResourceContext(md)
	default:
		return md, errInvalidContext
	}
}

// processMetricsWithResourceContext processes incoming metrics with resource context
func (p *lookupProcessor) processMetricsWithResourceContext(md pmetric.Metrics) (pmetric.Metrics, error) {
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		resource := md.ResourceMetrics().At(i)
		attrs := resource.Resource().Attributes()
		p.addLookupValues(attrs)
	}

	return md, nil
}

// processMetricsWithAttributesContext processes incoming metrics with attributes context
func (p *lookupProcessor) processMetricsWithAttributesContext(md pmetric.Metrics) (pmetric.Metrics, error) {
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		resource := md.ResourceMetrics().At(i)
		for j := 0; j < resource.ScopeMetrics().Len(); j++ {
			scope := resource.ScopeMetrics().At(j)
			for k := 0; k < scope.Metrics().Len(); k++ {
				metrics := scope.Metrics().At(k)

				switch metrics.Type() {
				case pmetric.MetricTypeSum:
					p.processSumMetrics(metrics)
				case pmetric.MetricTypeGauge:
					p.processGaugeMetrics(metrics)
				case pmetric.MetricTypeSummary:
					p.processSummaryMetrics(metrics)
				case pmetric.MetricTypeHistogram:
					p.processHistogramMetrics(metrics)
				case pmetric.MetricTypeExponentialHistogram:
					p.processExponentialHistogramMetrics(metrics)
				}
			}
		}
	}

	return md, nil
}

// processSumMetrics processes incoming sum metrics
func (p *lookupProcessor) processSumMetrics(metrics pmetric.Metric) {
	sum := metrics.Sum()
	for i := 0; i < sum.DataPoints().Len(); i++ {
		attrs := sum.DataPoints().At(i).Attributes()
		p.addLookupValues(attrs)
	}
}

// processGaugeMetrics processes incoming gauge metrics
func (p *lookupProcessor) processGaugeMetrics(metrics pmetric.Metric) {
	gauge := metrics.Gauge()
	for i := 0; i < gauge.DataPoints().Len(); i++ {
		attrs := gauge.DataPoints().At(i).Attributes()
		p.addLookupValues(attrs)
	}
}

// processSummaryMetrics processes incoming summary metrics
func (p *lookupProcessor) processSummaryMetrics(metrics pmetric.Metric) {
	summary := metrics.Summary()
	for i := 0; i < summary.DataPoints().Len(); i++ {
		attrs := summary.DataPoints().At(i).Attributes()
		p.addLookupValues(attrs)
	}
}

// processHistogramMetrics processes incoming histogram metrics
func (p *lookupProcessor) processHistogramMetrics(metrics pmetric.Metric) {
	histogram := metrics.Histogram()
	for i := 0; i < histogram.DataPoints().Len(); i++ {
		attrs := histogram.DataPoints().At(i).Attributes()
		p.addLookupValues(attrs)
	}
}

// processExponentialHistogramMetrics processes incoming exponential histogram metrics
func (p *lookupProcessor) processExponentialHistogramMetrics(metrics pmetric.Metric) {
	exponentialHistogram := metrics.ExponentialHistogram()
	for i := 0; i < exponentialHistogram.DataPoints().Len(); i++ {
		attrs := exponentialHistogram.DataPoints().At(i).Attributes()
		p.addLookupValues(attrs)
	}
}

// addLookupValues adds lookup values to the source map
func (p *lookupProcessor) addLookupValues(source pcommon.Map) {
	lookupValue, ok := source.Get(p.field)
	if !ok {
		return
	}

	if lookupValue.Type() != pcommon.ValueTypeStr {
		return
	}

	mappedValues, err := p.source.Lookup(lookupValue.AsString())
	if err != nil {
		p.logger.Debug("Could not find value in source", zap.String("value", lookupValue.AsString()), zap.Error(err))
		return
	}

	for k, v := range mappedValues {
		source.PutStr(k, v)
	}
}
