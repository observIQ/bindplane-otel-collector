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

package samplingprocessor

import (
	"context"
	"math/rand"

	"github.com/observiq/bindplane-agent/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type logsSamplingProcessor struct {
	logger          *zap.Logger
	dropCutOffRatio float64
	condition       *expr.OTTLCondition[ottllog.TransformContext]
}

type metricsSamplingProcessor struct {
	logger          *zap.Logger
	dropCutOffRatio float64
	condition       *expr.OTTLCondition[ottldatapoint.TransformContext]
}

type tracesSamplingProcessor struct {
	logger          *zap.Logger
	dropCutOffRatio float64
	condition       *expr.OTTLCondition[ottlspan.TransformContext]
}

func newLogsSamplingProcessor(logger *zap.Logger, cfg *Config, condition *expr.OTTLCondition[ottllog.TransformContext]) *logsSamplingProcessor {
	return &logsSamplingProcessor{
		logger:          logger,
		dropCutOffRatio: cfg.DropRatio,
		condition:       condition,
	}
}

func newMetricsSamplingProcessor(logger *zap.Logger, cfg *Config, condition *expr.OTTLCondition[ottldatapoint.TransformContext]) *metricsSamplingProcessor {
	return &metricsSamplingProcessor{
		logger:          logger,
		dropCutOffRatio: cfg.DropRatio,
		condition:       condition,
	}
}

func newTracesSamplingProcessor(logger *zap.Logger, cfg *Config, condition *expr.OTTLCondition[ottlspan.TransformContext]) *tracesSamplingProcessor {
	return &tracesSamplingProcessor{
		logger:          logger,
		dropCutOffRatio: cfg.DropRatio,
		condition:       condition,
	}
}

func sampleFunc(dropCutOffRatio float64) bool {
	//#nosec G404 -- randomly generated number is not used for security purposes. It's ok if it's weak
	return rand.Float64() <= dropCutOffRatio
}

func (sp *tracesSamplingProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	// Drop nothing
	if sp.dropCutOffRatio == 0.0 {
		return td, nil
	}

	// Drop based on ratio and condition
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		for j := 0; j < td.ResourceSpans().At(i).ScopeSpans().Len(); j++ {
			td.ResourceSpans().At(i).ScopeSpans().At(j).Spans().RemoveIf(func(span ptrace.Span) bool {
				logCtx := ottlspan.NewTransformContext(
					span,
					td.ResourceSpans().At(i).ScopeSpans().At(j).Scope(),
					td.ResourceSpans().At(i).Resource(),
					td.ResourceSpans().At(i).ScopeSpans().At(j),
					td.ResourceSpans().At(i),
				)
				match, err := sp.condition.Match(ctx, logCtx)
				return err == nil && match && sampleFunc(sp.dropCutOffRatio)
			})
		}
	}
	return td, nil
}

func (sp *logsSamplingProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	// Drop nothing
	if sp.dropCutOffRatio == 0.0 {
		return ld, nil
	}
	// Drop based on ratio and condition
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		for j := 0; j < ld.ResourceLogs().At(i).ScopeLogs().Len(); j++ {
			ld.ResourceLogs().At(i).ScopeLogs().At(j).LogRecords().RemoveIf(func(logRecord plog.LogRecord) bool {
				logCtx := ottllog.NewTransformContext(
					logRecord,
					ld.ResourceLogs().At(i).ScopeLogs().At(j).Scope(),
					ld.ResourceLogs().At(i).Resource(),
					ld.ResourceLogs().At(i).ScopeLogs().At(j),
					ld.ResourceLogs().At(i),
				)
				match, err := sp.condition.Match(ctx, logCtx)
				return err == nil && match && sampleFunc(sp.dropCutOffRatio)
			})
		}
	}
	return ld, nil
}

func (sp *metricsSamplingProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	// Drop nothing
	if sp.dropCutOffRatio == 0.0 {
		return md, nil
	}
	// Drop based on ratio and condition
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		for j := 0; j < md.ResourceMetrics().At(i).ScopeMetrics().Len(); j++ {
			md.ResourceMetrics().At(i).ScopeMetrics().At(j).Metrics().RemoveIf(func(metric pmetric.Metric) bool {
				anyMatch := false
				eachDatapoint(metric, func(dp any) {
					datapointCtx := ottldatapoint.NewTransformContext(
						dp,
						metric,
						md.ResourceMetrics().At(i).ScopeMetrics().At(j).Metrics(),
						md.ResourceMetrics().At(i).ScopeMetrics().At(j).Scope(),
						md.ResourceMetrics().At(i).Resource(),
						md.ResourceMetrics().At(i).ScopeMetrics().At(j),
						md.ResourceMetrics().At(i),
					)
					curDatapointMatched, err := sp.condition.Match(ctx, datapointCtx)
					anyMatch = anyMatch || (err == nil && curDatapointMatched)
				})

				// if no datapoints matched, check if metric matches without a datapoint
				if !anyMatch {
					metricCtx := ottldatapoint.NewTransformContext(
						nil,
						metric,
						md.ResourceMetrics().At(i).ScopeMetrics().At(j).Metrics(),
						md.ResourceMetrics().At(i).ScopeMetrics().At(j).Scope(),
						md.ResourceMetrics().At(i).Resource(),
						md.ResourceMetrics().At(i).ScopeMetrics().At(j),
						md.ResourceMetrics().At(i),
					)
					metricMatch, err := sp.condition.Match(ctx, metricCtx)
					anyMatch = err == nil && metricMatch
				}

				return anyMatch && sampleFunc(sp.dropCutOffRatio)
			})
		}
	}
	return md, nil
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
