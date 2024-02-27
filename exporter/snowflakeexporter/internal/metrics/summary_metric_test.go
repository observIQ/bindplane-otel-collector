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

func TestSummaryBatchInsert(t *testing.T) {
	testCases := []struct {
		desc        string
		ctx         context.Context
		summaryGen  func() []*summaryData
		mockGen     func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase
		expectedErr error
	}{
		{
			desc:       "pass",
			ctx:        context.Background(),
			summaryGen: generateSummaryMetrics,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("BatchInsert", ctx, expectedSummaryMaps(), sql).Return(nil)
				return m
			},
		},
		{
			desc:       "no data",
			ctx:        context.Background(),
			summaryGen: func() []*summaryData { return []*summaryData{} },
			mockGen:    func(_ *testing.T, _ context.Context, _ string) *mocks.MockDatabase { return nil },
		},
		{
			desc:       "fail BatchInsert",
			ctx:        context.Background(),
			summaryGen: generateSummaryMetrics,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("BatchInsert", ctx, expectedSummaryMaps(), sql).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to insert summary metric data: fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			summary := NewSummaryModel(zap.NewNop(), "insert")
			summary.summaries = tc.summaryGen()

			mockDB := tc.mockGen(t, tc.ctx, summary.insertSQL)

			err := summary.BatchInsert(tc.ctx, mockDB)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func generateSummaryMetrics() []*summaryData {
	md := generateTestMetrics() // from common_test.go
	resource := md.ResourceMetrics().At(0)
	scope := resource.ScopeMetrics().At(0)
	metric := scope.Metrics().At(4) // summary index
	return []*summaryData{
		{
			resource: resource,
			scope:    scope,
			metric:   metric,
			summary:  metric.Summary(),
		},
	}
}

func expectedSummaryMaps() []map[string]any {
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
			"mName":          "summary metrics",
			"mDescription":   "summary metrics for unit tests",
			"mUnit":          "m^2",
			"attributes":     "{}",
			"startTimestamp": time.Unix(0, int64(0)).UTC(),
			"timestamp":      time.Unix(0, int64(0)).UTC(),
			"count":          uint64(1),
			"sum":            float64(2.03),
			"flags":          pmetric.DataPointFlags(0),
			"quantiles":      utility.Array{float64(0)},
			"values":         utility.Array{1.7},
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
			"mName":          "summary metrics",
			"mDescription":   "summary metrics for unit tests",
			"mUnit":          "m^2",
			"attributes":     "{}",
			"startTimestamp": time.Unix(0, int64(0)).UTC(),
			"timestamp":      time.Unix(0, int64(0)).UTC(),
			"count":          uint64(2),
			"sum":            float64(3.03),
			"flags":          pmetric.DataPointFlags(1),
			"quantiles":      utility.Array{float64(1)},
			"values":         utility.Array{2.7},
		},
	}
}
