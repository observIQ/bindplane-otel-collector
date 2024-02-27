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

func TestExponentialHistogramBatchInsert(t *testing.T) {
	testCases := []struct {
		desc        string
		ctx         context.Context
		ehmGen      func() []*exponentialHistogramData
		mockGen     func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase
		expectedErr error
	}{
		{
			desc:   "pass",
			ctx:    context.Background(),
			ehmGen: generateEHMMetrics,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("BatchInsert", ctx, expectedEHMMaps(), sql).Return(nil)
				return m
			},
		},
		{
			desc:    "no data",
			ctx:     context.Background(),
			ehmGen:  func() []*exponentialHistogramData { return []*exponentialHistogramData{} },
			mockGen: func(_ *testing.T, _ context.Context, _ string) *mocks.MockDatabase { return nil },
		},
		{
			desc:   "fail BatchInsert",
			ctx:    context.Background(),
			ehmGen: generateEHMMetrics,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("BatchInsert", ctx, expectedEHMMaps(), sql).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to insert exponential histogram metric data: fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ehm := NewExponentialHistogramModel(zap.NewNop(), "insert")
			ehm.exponentialHistograms = tc.ehmGen()

			mockDB := tc.mockGen(t, tc.ctx, ehm.insertSQL)

			err := ehm.BatchInsert(tc.ctx, mockDB)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func generateEHMMetrics() []*exponentialHistogramData {
	md := generateTestMetrics() // from common_test.go
	resource := md.ResourceMetrics().At(0)
	scope := resource.ScopeMetrics().At(0)
	metric := scope.Metrics().At(0) // ehm index
	return []*exponentialHistogramData{
		{
			resource:             resource,
			scope:                scope,
			metric:               metric,
			exponentialHistogram: metric.ExponentialHistogram(),
		},
	}
}

func expectedEHMMaps() []map[string]any {
	return []map[string]any{
		{
			"rSchema":              "resource_test_metrics",
			"rDroppedCount":        uint32(1),
			"rAttributes":          "{\"a1\":\"resource_attributes\"}",
			"sSchema":              "scope_test_metrics",
			"sName":                "unit_test_scope_metrics",
			"sVersion":             "v0",
			"sDroppedCount":        uint32(1),
			"sAttributes":          "{\"a1\":\"scope_attributes\",\"parent\":\"resource\"}",
			"mName":                "exponential histogram metrics",
			"mDescription":         "eh metrics for unit tests",
			"mUnit":                "m/s",
			"aggTemp":              "Unspecified",
			"attributes":           "{}",
			"startTimestamp":       time.Unix(0, int64(0)).UTC(),
			"timestamp":            time.Unix(0, int64(0)).UTC(),
			"count":                uint64(0),
			"sum":                  float64(0.01),
			"scale":                int32(1),
			"zeroCount":            uint64(2),
			"zeroThreshold":        float64(2.1),
			"flags":                pmetric.DataPointFlags(0),
			"min":                  float64(2.3),
			"max":                  float64(3.2),
			"positiveOffset":       int32(0),
			"positiveBucketCounts": []uint64{0, 1, 2, 3, 4},
			"negativeOffset":       int32(1),
			"negativeBucketCounts": []uint64{5, 6, 7, 8, 0},
			"eAttributes":          utility.Array{map[string]any{"a1": "exemplar attribute", "a2": int64(0)}, map[string]any{"a1": "exemplar attribute", "a2": int64(1)}, map[string]any{}},
			"eTimestamps":          utility.Array{time.Unix(0, int64(0)).UTC(), time.Unix(0, int64(0)).UTC(), time.Unix(0, int64(0)).UTC()},
			"eTraceIDs":            utility.Array{"", "", ""},
			"eSpanIDs":             utility.Array{"", "", ""},
			"eValues":              utility.Array{2.1, 3.1, int64(3)},
		},
		{
			"rSchema":              "resource_test_metrics",
			"rDroppedCount":        uint32(1),
			"rAttributes":          "{\"a1\":\"resource_attributes\"}",
			"sSchema":              "scope_test_metrics",
			"sName":                "unit_test_scope_metrics",
			"sVersion":             "v0",
			"sDroppedCount":        uint32(1),
			"sAttributes":          "{\"a1\":\"scope_attributes\",\"parent\":\"resource\"}",
			"mName":                "exponential histogram metrics",
			"mDescription":         "eh metrics for unit tests",
			"mUnit":                "m/s",
			"aggTemp":              "Unspecified",
			"attributes":           "{}",
			"startTimestamp":       time.Unix(0, int64(0)).UTC(),
			"timestamp":            time.Unix(0, int64(0)).UTC(),
			"count":                uint64(1),
			"sum":                  float64(1.01),
			"scale":                int32(2),
			"zeroCount":            uint64(3),
			"zeroThreshold":        float64(3.1),
			"flags":                pmetric.DataPointFlags(1),
			"min":                  float64(3.3),
			"max":                  float64(4.2),
			"positiveOffset":       int32(1),
			"positiveBucketCounts": []uint64{1, 1, 2, 3, 4},
			"negativeOffset":       int32(2),
			"negativeBucketCounts": []uint64{5, 6, 7, 8, 1},
			"eAttributes":          utility.Array{map[string]any{"a1": "exemplar attribute", "a2": int64(0)}, map[string]any{"a1": "exemplar attribute", "a2": int64(1)}, map[string]any{}},
			"eTimestamps":          utility.Array{time.Unix(0, int64(0)).UTC(), time.Unix(0, int64(0)).UTC(), time.Unix(0, int64(0)).UTC()},
			"eTraceIDs":            utility.Array{"", "", ""},
			"eSpanIDs":             utility.Array{"", "", ""},
			"eValues":              utility.Array{2.1, 3.1, int64(3)},
		},
		{
			"rSchema":              "resource_test_metrics",
			"rDroppedCount":        uint32(1),
			"rAttributes":          "{\"a1\":\"resource_attributes\"}",
			"sSchema":              "scope_test_metrics",
			"sName":                "unit_test_scope_metrics",
			"sVersion":             "v0",
			"sDroppedCount":        uint32(1),
			"sAttributes":          "{\"a1\":\"scope_attributes\",\"parent\":\"resource\"}",
			"mName":                "exponential histogram metrics",
			"mDescription":         "eh metrics for unit tests",
			"mUnit":                "m/s",
			"aggTemp":              "Unspecified",
			"attributes":           "{}",
			"startTimestamp":       time.Unix(0, int64(0)).UTC(),
			"timestamp":            time.Unix(0, int64(0)).UTC(),
			"count":                uint64(2),
			"sum":                  float64(2.01),
			"scale":                int32(3),
			"zeroCount":            uint64(4),
			"zeroThreshold":        float64(4.1),
			"flags":                pmetric.DataPointFlags(2),
			"min":                  float64(4.3),
			"max":                  float64(5.2),
			"positiveOffset":       int32(2),
			"positiveBucketCounts": []uint64{2, 1, 2, 3, 4},
			"negativeOffset":       int32(3),
			"negativeBucketCounts": []uint64{5, 6, 7, 8, 2},
			"eAttributes":          utility.Array{map[string]any{"a1": "exemplar attribute", "a2": int64(0)}, map[string]any{"a1": "exemplar attribute", "a2": int64(1)}, map[string]any{}},
			"eTimestamps":          utility.Array{time.Unix(0, int64(0)).UTC(), time.Unix(0, int64(0)).UTC(), time.Unix(0, int64(0)).UTC()},
			"eTraceIDs":            utility.Array{"", "", ""},
			"eSpanIDs":             utility.Array{"", "", ""},
			"eValues":              utility.Array{2.1, 3.1, int64(3)},
		},
	}
}
