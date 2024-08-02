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

	"github.com/observiq/bindplane-otel-collector/internal/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type logsSamplingProcessor struct {
	logger          *zap.Logger
	dropCutOffRatio float64
	conditionString string
	condition       *expr.OTTLCondition[ottllog.TransformContext]
}

type metricsSamplingProcessor struct {
	logger          *zap.Logger
	dropCutOffRatio float64
	conditionString string
	condition       *expr.OTTLCondition[ottlmetric.TransformContext]
}

type tracesSamplingProcessor struct {
	logger          *zap.Logger
	dropCutOffRatio float64
	conditionString string
	condition       *expr.OTTLCondition[ottlspan.TransformContext]
}

func newLogsSamplingProcessor(logger *zap.Logger, cfg *Config, condition *expr.OTTLCondition[ottllog.TransformContext]) *logsSamplingProcessor {
	return &logsSamplingProcessor{
		logger:          logger,
		dropCutOffRatio: cfg.DropRatio,
		condition:       condition,
		conditionString: cfg.Condition,
	}
}

func newMetricsSamplingProcessor(logger *zap.Logger, cfg *Config, condition *expr.OTTLCondition[ottlmetric.TransformContext]) *metricsSamplingProcessor {
	return &metricsSamplingProcessor{
		logger:          logger,
		dropCutOffRatio: cfg.DropRatio,
		condition:       condition,
		conditionString: cfg.Condition,
	}
}

func newTracesSamplingProcessor(logger *zap.Logger, cfg *Config, condition *expr.OTTLCondition[ottlspan.TransformContext]) *tracesSamplingProcessor {
	return &tracesSamplingProcessor{
		logger:          logger,
		dropCutOffRatio: cfg.DropRatio,
		condition:       condition,
		conditionString: cfg.Condition,
	}
}

func sampleFunc(dropCutOffRatio float64) bool {
	//#nosec G404 -- randomly generated number is not used for security purposes. It's ok if it's weak
	return rand.Float64() <= dropCutOffRatio
}

func (sp *tracesSamplingProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	// Drop everything
	if sp.dropCutOffRatio == 1.0 && sp.conditionString == "true" {
		return ptrace.NewTraces(), nil
	}

	// Drop nothing
	if sp.dropCutOffRatio == 0.0 || sp.conditionString == "false" {
		return td, nil
	}

	// Drop based on ratio and condition
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		for j := 0; j < td.ResourceSpans().At(i).ScopeSpans().Len(); j++ {
			td.ResourceSpans().At(i).ScopeSpans().At(j).Spans().RemoveIf(func(span ptrace.Span) bool {
				if sp.conditionString == "true" {
					return sampleFunc(sp.dropCutOffRatio)
				}

				spanCtx := ottlspan.NewTransformContext(
					span,
					td.ResourceSpans().At(i).ScopeSpans().At(j).Scope(),
					td.ResourceSpans().At(i).Resource(),
					td.ResourceSpans().At(i).ScopeSpans().At(j),
					td.ResourceSpans().At(i),
				)
				match, err := sp.condition.Match(ctx, spanCtx)
				return err == nil && match && sampleFunc(sp.dropCutOffRatio)
			})
		}
		td.ResourceSpans().At(i).ScopeSpans().RemoveIf(func(ss ptrace.ScopeSpans) bool {
			return ss.Spans().Len() == 0
		})
	}
	td.ResourceSpans().RemoveIf(func(rs ptrace.ResourceSpans) bool {
		return rs.ScopeSpans().Len() == 0
	})
	return td, nil
}

func (sp *logsSamplingProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	// Drop everything
	if sp.dropCutOffRatio == 1.0 && sp.conditionString == "true" {
		return plog.NewLogs(), nil
	}

	// Drop nothing
	if sp.dropCutOffRatio == 0.0 || sp.conditionString == "false" {
		return ld, nil
	}

	// Drop based on ratio and condition
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		for j := 0; j < ld.ResourceLogs().At(i).ScopeLogs().Len(); j++ {
			ld.ResourceLogs().At(i).ScopeLogs().At(j).LogRecords().RemoveIf(func(logRecord plog.LogRecord) bool {
				if sp.conditionString == "true" {
					return sampleFunc(sp.dropCutOffRatio)
				}

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
		ld.ResourceLogs().At(i).ScopeLogs().RemoveIf(func(sl plog.ScopeLogs) bool {
			return sl.LogRecords().Len() == 0
		})
	}
	ld.ResourceLogs().RemoveIf(func(rl plog.ResourceLogs) bool {
		return rl.ScopeLogs().Len() == 0
	})
	return ld, nil
}

func (sp *metricsSamplingProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	// Drop everything
	if sp.dropCutOffRatio == 1.0 && sp.conditionString == "true" {
		return pmetric.NewMetrics(), nil
	}

	// Drop nothing
	if sp.dropCutOffRatio == 0.0 || sp.conditionString == "false" {
		return md, nil
	}

	// Drop based on ratio and condition
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		for j := 0; j < md.ResourceMetrics().At(i).ScopeMetrics().Len(); j++ {
			md.ResourceMetrics().At(i).ScopeMetrics().At(j).Metrics().RemoveIf(func(metric pmetric.Metric) bool {
				if sp.conditionString == "true" {
					return sampleFunc(sp.dropCutOffRatio)
				}

				metricCtx := ottlmetric.NewTransformContext(
					metric,
					md.ResourceMetrics().At(i).ScopeMetrics().At(j).Metrics(),
					md.ResourceMetrics().At(i).ScopeMetrics().At(j).Scope(),
					md.ResourceMetrics().At(i).Resource(),
					md.ResourceMetrics().At(i).ScopeMetrics().At(j),
					md.ResourceMetrics().At(i),
				)
				match, err := sp.condition.Match(ctx, metricCtx)
				return err == nil && match && sampleFunc(sp.dropCutOffRatio)
			})
		}
		md.ResourceMetrics().At(i).ScopeMetrics().RemoveIf(func(sm pmetric.ScopeMetrics) bool {
			return sm.Metrics().Len() == 0
		})
	}
	md.ResourceMetrics().RemoveIf(func(rm pmetric.ResourceMetrics) bool {
		return rm.ScopeMetrics().Len() == 0
	})
	return md, nil
}
