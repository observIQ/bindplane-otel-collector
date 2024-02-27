// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/database/mocks"
	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/utility"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

func TestHistogramBatchInsert(t *testing.T) {
	testCases := []struct {
		desc         string
		ctx          context.Context
		histogramGen func() []*histogramData
		mockGen      func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase
		expectedErr  error
	}{
		{
			desc:         "pass",
			ctx:          context.Background(),
			histogramGen: generateHistogramMetrics,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("BatchInsert", ctx, expectedHistogramMaps(), sql).Return(nil)
				return m
			},
		},
		{
			desc:         "no data",
			ctx:          context.Background(),
			histogramGen: func() []*histogramData { return []*histogramData{} },
			mockGen:      func(_ *testing.T, _ context.Context, _ string) *mocks.MockDatabase { return nil },
		},
		{
			desc:         "fail batchInsert",
			ctx:          context.Background(),
			histogramGen: generateHistogramMetrics,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("BatchInsert", ctx, expectedHistogramMaps(), sql).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to insert histogram metric data: fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			histogram := NewHistogramModel(zap.NewNop(), "insert")
			histogram.histograms = tc.histogramGen()

			mockDB := tc.mockGen(t, tc.ctx, histogram.insertSQL)

			err := histogram.BatchInsert(tc.ctx, mockDB)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func generateHistogramMetrics() []*histogramData {
	md := generateTestMetrics() // from common_test.go
	resource := md.ResourceMetrics().At(0)
	scope := resource.ScopeMetrics().At(0)
	metric := scope.Metrics().At(2) // histogram index
	return []*histogramData{
		{
			resource:  resource,
			scope:     scope,
			metric:    metric,
			histogram: metric.Histogram(),
		},
	}
}

func expectedHistogramMaps() []map[string]any {
	return []map[string]any{
		{
			"rSchema":        "resource_test_metrics",
			"rDroppedCount":  uint32(1),
			"rAttributes":    "{\"a1\":\"resource_attributes\"}",
			"sSchema":        "scope_test_metrics",
			"sName":          "unit_test_scope_metrics",
			"sVersion":       "v0",
			"sDroppedCount":  uint32(1),
			"sAttributes":    "{\"a1\":\"scope_attributes\",\"parent\":\"resource\"}",
			"mName":          "histogram metrics",
			"mDescription":   "histogram metrics for unit tests",
			"mUnit":          "mi/h",
			"aggTemp":        "Cumulative",
			"attributes":     "{}",
			"startTimestamp": time.Unix(0, int64(0)).UTC(),
			"timestamp":      time.Unix(0, int64(0)).UTC(),
			"count":          uint64(3),
			"sum":            float64(0.2),
			"flags":          pmetric.DataPointFlags(0),
			"min":            float64(2.1),
			"max":            float64(3.4),
			"bucketCounts":   []uint64{1, 3, 0, 4},
			"explicitBounds": []float64{0.3, 4.1, 2.01, 1.1},
			"eAttributes":    utility.Array{map[string]any{"a1": "exemplar attribute", "a2": int64(0)}, map[string]any{"a1": "exemplar attribute", "a2": int64(1)}, map[string]any{}},
			"eTimestamps":    utility.Array{time.Unix(0, int64(0)).UTC(), time.Unix(0, int64(0)).UTC(), time.Unix(0, int64(0)).UTC()},
			"eTraceIDs":      utility.Array{"", "", ""},
			"eSpanIDs":       utility.Array{"", "", ""},
			"eValues":        utility.Array{2.1, 3.1, int64(3)},
		},
		{
			"rSchema":        "resource_test_metrics",
			"rDroppedCount":  uint32(1),
			"rAttributes":    "{\"a1\":\"resource_attributes\"}",
			"sSchema":        "scope_test_metrics",
			"sName":          "unit_test_scope_metrics",
			"sVersion":       "v0",
			"sDroppedCount":  uint32(1),
			"sAttributes":    "{\"a1\":\"scope_attributes\",\"parent\":\"resource\"}",
			"mName":          "histogram metrics",
			"mDescription":   "histogram metrics for unit tests",
			"mUnit":          "mi/h",
			"aggTemp":        "Cumulative",
			"attributes":     "{}",
			"startTimestamp": time.Unix(0, int64(0)).UTC(),
			"timestamp":      time.Unix(0, int64(0)).UTC(),
			"count":          uint64(4),
			"sum":            float64(1.2),
			"flags":          pmetric.DataPointFlags(1),
			"min":            float64(3.1),
			"max":            float64(4.4),
			"bucketCounts":   []uint64{1, 3, 0, 4},
			"explicitBounds": []float64{0.3, 4.1, 2.01, 2.1},
			"eAttributes":    utility.Array{map[string]any{"a1": "exemplar attribute", "a2": int64(0)}, map[string]any{"a1": "exemplar attribute", "a2": int64(1)}, map[string]any{}},
			"eTimestamps":    utility.Array{time.Unix(0, int64(0)).UTC(), time.Unix(0, int64(0)).UTC(), time.Unix(0, int64(0)).UTC()},
			"eTraceIDs":      utility.Array{"", "", ""},
			"eSpanIDs":       utility.Array{"", "", ""},
			"eValues":        utility.Array{2.1, 3.1, int64(3)},
		},
	}
}
