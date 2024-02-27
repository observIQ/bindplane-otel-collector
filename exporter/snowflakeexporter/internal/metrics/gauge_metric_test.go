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

func TestGaugeMetricBatchInsert(t *testing.T) {
	testCases := []struct {
		desc        string
		ctx         context.Context
		gaugeGen    func() []*gaugeData
		mockGen     func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase
		expectedErr error
	}{
		{
			desc:     "pass",
			ctx:      context.Background(),
			gaugeGen: generateGaugeMetrics,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("BatchInsert", ctx, expectedGaugeMaps(), sql).Return(nil)
				return m
			},
		},
		{
			desc:     "no data",
			ctx:      context.Background(),
			gaugeGen: func() []*gaugeData { return []*gaugeData{} },
			mockGen:  func(_ *testing.T, _ context.Context, _ string) *mocks.MockDatabase { return nil },
		},
		{
			desc:     "fail BatchInsert",
			ctx:      context.Background(),
			gaugeGen: generateGaugeMetrics,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("BatchInsert", ctx, expectedGaugeMaps(), sql).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to insert gauge metric data: fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			g := NewGaugeModel(zap.NewNop(), "insert")
			g.gauges = tc.gaugeGen()

			mockDB := tc.mockGen(t, tc.ctx, g.insertSQL)

			err := g.BatchInsert(tc.ctx, mockDB)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func generateGaugeMetrics() []*gaugeData {
	md := generateTestMetrics() // from common_test.go
	resource := md.ResourceMetrics().At(0)
	scope := resource.ScopeMetrics().At(0)
	metric := scope.Metrics().At(1) // gauge index
	return []*gaugeData{
		{
			resource: resource,
			scope:    scope,
			metric:   metric,
			gauge:    metric.Gauge(),
		},
	}
}

func expectedGaugeMaps() []map[string]any {
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
			"mName":          "gauge metrics",
			"mDescription":   "gauge metrics for unit tests",
			"mUnit":          "N",
			"attributes":     "{\"a1\":0,\"a2\":\"gauge attributes\"}",
			"startTimestamp": time.Unix(0, int64(0)).UTC(),
			"timestamp":      time.Unix(0, int64(0)).UTC(),
			"value":          float64(1.23),
			"flags":          pmetric.DataPointFlags(0),
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
			"mName":          "gauge metrics",
			"mDescription":   "gauge metrics for unit tests",
			"mUnit":          "N",
			"attributes":     "{\"a1\":1,\"a2\":\"gauge attributes\"}",
			"startTimestamp": time.Unix(0, int64(0)).UTC(),
			"timestamp":      time.Unix(0, int64(0)).UTC(),
			"value":          float64(2.23),
			"flags":          pmetric.DataPointFlags(1),
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
			"mName":          "gauge metrics",
			"mDescription":   "gauge metrics for unit tests",
			"mUnit":          "N",
			"attributes":     "{\"a1\":2,\"a2\":\"gauge attributes\"}",
			"startTimestamp": time.Unix(0, int64(0)).UTC(),
			"timestamp":      time.Unix(0, int64(0)).UTC(),
			"value":          float64(3.23),
			"flags":          pmetric.DataPointFlags(2),
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
			"mName":          "gauge metrics",
			"mDescription":   "gauge metrics for unit tests",
			"mUnit":          "N",
			"attributes":     "{\"a1\":3,\"a2\":\"gauge attributes\"}",
			"startTimestamp": time.Unix(0, int64(0)).UTC(),
			"timestamp":      time.Unix(0, int64(0)).UTC(),
			"value":          float64(4.23),
			"flags":          pmetric.DataPointFlags(3),
			"eAttributes":    utility.Array{map[string]any{"a1": "exemplar attribute", "a2": int64(0)}, map[string]any{"a1": "exemplar attribute", "a2": int64(1)}, map[string]any{}},
			"eTimestamps":    utility.Array{time.Unix(0, int64(0)).UTC(), time.Unix(0, int64(0)).UTC(), time.Unix(0, int64(0)).UTC()},
			"eTraceIDs":      utility.Array{"", "", ""},
			"eSpanIDs":       utility.Array{"", "", ""},
			"eValues":        utility.Array{2.1, 3.1, int64(3)},
		},
	}
}
