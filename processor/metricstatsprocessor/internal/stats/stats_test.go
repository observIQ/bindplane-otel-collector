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

package stats

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestStatTypesValid(t *testing.T) {
	types := []StatType{
		MinType,
		AvgType,
		MaxType,
		FirstType,
		LastType,
	}

	for _, statType := range types {
		t.Run(string(statType), func(t *testing.T) {
			require.True(t, statType.Valid(), "Invalid statistic type %s", statType)
		})
	}
}

func TestStatTypesNew(t *testing.T) {
	types := []StatType{
		MinType,
		AvgType,
		MaxType,
		FirstType,
		LastType,
	}

	for _, statType := range types {
		t.Run(string(statType), func(t *testing.T) {
			dp := pmetric.NewNumberDataPoint()
			dp.SetDoubleValue(2.0)
			a, err := statType.New(dp)
			require.NoError(t, err)
			require.NotNil(t, a)
		})
	}
}

func TestStatTypeNewInvalidType(t *testing.T) {
	dp := pmetric.NewNumberDataPoint()
	dp.SetDoubleValue(2.0)
	_, err := StatType("invalid").New(dp)
	require.ErrorContains(t, err, "invalid statistic type:")
}

func TestStatTypesNewEmptyDatapoint(t *testing.T) {
	types := []StatType{
		MinType,
		AvgType,
		MaxType,
		FirstType,
		LastType,
	}

	for _, statType := range types {
		t.Run(string(statType), func(t *testing.T) {
			dp := pmetric.NewNumberDataPoint()
			_, err := statType.New(dp)
			require.ErrorContains(t, err, fmt.Sprintf("cannot create %s statistic from empty datapoint", statType))
		})
	}
}

func TestStatisticsFloat(t *testing.T) {
	testCases := []struct {
		name       string
		statType   StatType
		values     []float64
		timestamps []int64
		finalValue float64
	}{
		{
			name:       "min",
			statType:   MinType,
			values:     []float64{45, 1, 99, 3},
			timestamps: []int64{0, 0, 0, 0},
			finalValue: 1,
		},
		{
			name:       "max",
			statType:   MaxType,
			values:     []float64{45, 1, 99, 3},
			timestamps: []int64{0, 0, 0, 0},
			finalValue: 99,
		},
		{
			name:       "avg",
			statType:   AvgType,
			values:     []float64{45, 0, 99, 3},
			timestamps: []int64{0, 0, 0, 0},
			finalValue: 36.75,
		},
		{
			name:       "first (unset timestamp)",
			statType:   FirstType,
			values:     []float64{45, 1, 99, 3},
			timestamps: []int64{0, 0, 0, 0},
			finalValue: 45,
		},
		{
			name:       "last (unset timestamp)",
			statType:   LastType,
			values:     []float64{45, 1, 99, 3},
			timestamps: []int64{0, 0, 0, 0},
			finalValue: 3,
		},
		{
			name:       "first (set timestamp)",
			statType:   FirstType,
			values:     []float64{45, 1, 99, 3},
			timestamps: []int64{10, 2, 3, 11},
			finalValue: 99,
		},
		{
			name:       "last (set timestamp)",
			statType:   LastType,
			values:     []float64{45, 1, 99, 3},
			timestamps: []int64{10, 3, 89, 11},
			finalValue: 99,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialVal := pmetric.NewNumberDataPoint()
			initialVal.SetDoubleValue(tc.values[0])
			initialVal.SetTimestamp(pcommon.Timestamp(tc.timestamps[0]))
			stat, err := tc.statType.New(initialVal)
			require.NoError(t, err)

			for i, v := range tc.values[1:] {
				dp := pmetric.NewNumberDataPoint()
				dp.SetTimestamp(pcommon.Timestamp(tc.timestamps[i+1]))
				dp.SetDoubleValue(v)
				stat.AddDatapoint(dp)
			}
			finalDp := pmetric.NewNumberDataPoint()
			stat.SetDatapointValue(finalDp)

			require.Equal(t, tc.finalValue, finalDp.DoubleValue())
		})
	}
}

func TestStatisticsInt(t *testing.T) {
	testCases := []struct {
		name       string
		statType   StatType
		values     []int64
		timestamps []int64
		finalValue int64
	}{
		{
			name:       "min",
			statType:   MinType,
			values:     []int64{45, 1, 99, 3},
			timestamps: []int64{0, 0, 0, 0},
			finalValue: 1,
		},
		{
			name:       "max",
			statType:   MaxType,
			values:     []int64{45, 1, 99, 3},
			timestamps: []int64{0, 0, 0, 0},
			finalValue: 99,
		},
		{
			name:       "avg",
			statType:   AvgType,
			values:     []int64{45, 0, 99, 3},
			timestamps: []int64{0, 0, 0, 0},
			finalValue: 36,
		},
		{
			name:       "first (unset timestamp)",
			statType:   FirstType,
			values:     []int64{45, 1, 99, 3},
			timestamps: []int64{0, 0, 0, 0},
			finalValue: 45,
		},
		{
			name:       "last (unset timestamp)",
			statType:   LastType,
			values:     []int64{45, 1, 99, 3},
			timestamps: []int64{0, 0, 0, 0},
			finalValue: 3,
		},
		{
			name:       "first (set timestamp)",
			statType:   FirstType,
			values:     []int64{45, 1, 99, 3},
			timestamps: []int64{10, 2, 3, 11},
			finalValue: 99,
		},
		{
			name:       "last (set timestamp)",
			statType:   LastType,
			values:     []int64{45, 1, 99, 3},
			timestamps: []int64{10, 3, 89, 11},
			finalValue: 99,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialVal := pmetric.NewNumberDataPoint()
			initialVal.SetIntValue(tc.values[0])
			initialVal.SetTimestamp(pcommon.Timestamp(tc.timestamps[0]))
			stat, err := tc.statType.New(initialVal)
			require.NoError(t, err)

			for i, v := range tc.values[1:] {
				dp := pmetric.NewNumberDataPoint()
				dp.SetTimestamp(pcommon.Timestamp(tc.timestamps[i+1]))
				dp.SetIntValue(v)
				stat.AddDatapoint(dp)
			}
			finalDp := pmetric.NewNumberDataPoint()
			stat.SetDatapointValue(finalDp)

			require.Equal(t, tc.finalValue, finalDp.IntValue())
		})
	}
}
