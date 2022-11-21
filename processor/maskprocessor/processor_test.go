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

package maskprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

var testMap = map[string]interface{}{
	"exclude":   "this is sensitive",
	"mask":      "this is sensitive",
	"unrelated": "this is unrelated",
	"embedded": map[string]interface{}{
		"mask": "this is sensitive",
	},
	"slice": []interface{}{
		"this is sensitive",
		int64(1),
	},
}

var expectedMap = map[string]interface{}{
	"exclude":   "this is sensitive",
	"mask":      "this is [masked_field]",
	"unrelated": "this is unrelated",
	"embedded": map[string]interface{}{
		"mask": "this is [masked_field]",
	},
	"slice": []interface{}{
		"this is [masked_field]",
		int64(1),
	},
}

func TestProcessTraces(t *testing.T) {
	traces := ptrace.NewTraces()
	resource := traces.ResourceSpans().AppendEmpty()
	resource.Resource().Attributes().FromRaw(testMap)
	span := resource.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.Attributes().FromRaw(testMap)

	cfg := &Config{
		Rules:   map[string]string{"field": "sensitive"},
		Exclude: []string{"resource.exclude", "attributes.exclude"},
	}
	processor := newProcessor(zap.NewNop(), cfg)
	err := processor.start(context.Background(), nil)
	require.NoError(t, err)

	result, err := processor.processTraces(context.Background(), traces)
	require.NoError(t, err)

	resourceAttrs := result.ResourceSpans().At(0).Resource().Attributes().AsRaw()
	require.Equal(t, expectedMap, resourceAttrs)

	attrs := result.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes().AsRaw()
	require.Equal(t, expectedMap, attrs)
}

func TestProcessMetrics(t *testing.T) {
	metrics := pmetric.NewMetrics()
	resource := metrics.ResourceMetrics().AppendEmpty()
	resource.Resource().Attributes().FromRaw(testMap)
	gauge := resource.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyGauge()
	gauge.DataPoints().AppendEmpty().Attributes().FromRaw(testMap)
	sum := resource.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySum()
	sum.DataPoints().AppendEmpty().Attributes().FromRaw(testMap)
	summary := resource.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptySummary()
	summary.DataPoints().AppendEmpty().Attributes().FromRaw(testMap)
	histogram := resource.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyHistogram()
	histogram.DataPoints().AppendEmpty().Attributes().FromRaw(testMap)
	exponentialHistogram := resource.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyExponentialHistogram()
	exponentialHistogram.DataPoints().AppendEmpty().Attributes().FromRaw(testMap)

	cfg := &Config{
		Rules:   map[string]string{"field": "sensitive"},
		Exclude: []string{"resource.exclude", "attributes.exclude"},
	}
	processor := newProcessor(zap.NewNop(), cfg)
	err := processor.start(context.Background(), nil)
	require.NoError(t, err)

	result, err := processor.processMetrics(context.Background(), metrics)
	require.NoError(t, err)

	resourceAttrs := result.ResourceMetrics().At(0).Resource().Attributes().AsRaw()
	require.Equal(t, expectedMap, resourceAttrs)

	scope := result.ResourceMetrics().At(0).ScopeMetrics()
	gaugeAttrs := scope.At(0).Metrics().At(0).Gauge().DataPoints().At(0).Attributes().AsRaw()
	require.Equal(t, expectedMap, gaugeAttrs)

	sumAttrs := scope.At(1).Metrics().At(0).Sum().DataPoints().At(0).Attributes().AsRaw()
	require.Equal(t, expectedMap, sumAttrs)

	summaryAttrs := scope.At(2).Metrics().At(0).Summary().DataPoints().At(0).Attributes().AsRaw()
	require.Equal(t, expectedMap, summaryAttrs)

	histogramAttrs := scope.At(3).Metrics().At(0).Histogram().DataPoints().At(0).Attributes().AsRaw()
	require.Equal(t, expectedMap, histogramAttrs)

	exponentialHistogramAttrs := scope.At(4).Metrics().At(0).ExponentialHistogram().DataPoints().At(0).Attributes().AsRaw()
	require.Equal(t, expectedMap, exponentialHistogramAttrs)
}

func TestProcessLogs(t *testing.T) {
	logs := plog.NewLogs()
	resource := logs.ResourceLogs().AppendEmpty()
	resource.Resource().Attributes().FromRaw(testMap)
	record := resource.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	record.Attributes().FromRaw(testMap)
	record.Body().FromRaw(testMap)

	cfg := &Config{
		Rules:   map[string]string{"field": "sensitive"},
		Exclude: []string{"resource.exclude", "attributes.exclude", "body.exclude"},
	}
	processor := newProcessor(zap.NewNop(), cfg)
	err := processor.start(context.Background(), nil)
	require.NoError(t, err)

	result, err := processor.processLogs(context.Background(), logs)
	require.NoError(t, err)

	resourceAttrs := result.ResourceLogs().At(0).Resource().Attributes().AsRaw()
	require.Equal(t, expectedMap, resourceAttrs)

	attrs := result.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes().AsRaw()
	require.Equal(t, expectedMap, attrs)

	body := result.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().AsRaw()
	require.Equal(t, expectedMap, body)
}

func TestFailedStart(t *testing.T) {
	cfg := &Config{
		Rules: map[string]string{"invalid": `\k`},
	}
	processor := newProcessor(zap.NewNop(), cfg)

	err := processor.start(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not compile as valid regex")
}
