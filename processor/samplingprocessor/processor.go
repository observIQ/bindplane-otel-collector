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

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type samplingProcessor struct {
	logger          *zap.Logger
	dropCutOffRatio float64
}

func newSamplingProcessor(logger *zap.Logger, cfg *Config) *samplingProcessor {
	return &samplingProcessor{
		logger:          logger,
		dropCutOffRatio: cfg.DropRatio,
	}
}

func (sp *samplingProcessor) sampleFunc() bool {
	//#nosec G404 -- randomly generated number is not used for security purposes. It's ok if it's weak
	return rand.Float64() <= sp.dropCutOffRatio
}

func (sp *samplingProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	switch {
	case sp.dropCutOffRatio == 1.0: // Drop everything
		return ptrace.NewTraces(), nil
	case sp.dropCutOffRatio == 0.0: // Drop nothing
		return td, nil
	default: // Drop based on ratio
		for i := 0; i < td.ResourceSpans().Len(); i++ {
			for j := 0; j < td.ResourceSpans().At(i).ScopeSpans().Len(); j++ {
				td.ResourceSpans().At(i).ScopeSpans().At(j).Spans().RemoveIf(func(_ ptrace.Span) bool {
					return sp.sampleFunc()
				})
			}
		}
		return td, nil
	}
}

func (sp *samplingProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	switch {
	case sp.dropCutOffRatio == 1.0: // Drop everything
		return plog.NewLogs(), nil
	case sp.dropCutOffRatio == 0.0: // Drop nothing
		return ld, nil
	default: // Drop based on ratio
		for i := 0; i < ld.ResourceLogs().Len(); i++ {
			for j := 0; j < ld.ResourceLogs().At(i).ScopeLogs().Len(); j++ {
				ld.ResourceLogs().At(i).ScopeLogs().At(j).LogRecords().RemoveIf(func(_ plog.LogRecord) bool {
					return sp.sampleFunc()
				})
			}
		}
		return ld, nil
	}
}

func (sp *samplingProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	switch {
	case sp.dropCutOffRatio == 1.0: // Drop everything
		return pmetric.NewMetrics(), nil
	case sp.dropCutOffRatio == 0.0: // Drop nothing
		return md, nil
	default: // Drop based on ratio
		for i := 0; i < md.ResourceMetrics().Len(); i++ {
			for j := 0; j < md.ResourceMetrics().At(i).ScopeMetrics().Len(); j++ {
				md.ResourceMetrics().At(i).ScopeMetrics().At(j).Metrics().RemoveIf(func(_ pmetric.Metric) bool {
					return sp.sampleFunc()
				})
			}
		}
		return md, nil
	}
}
