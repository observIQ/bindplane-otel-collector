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
	"errors"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

type minStatistic struct {
	minDouble float64
	minInt    int64
	isInt     bool
}

func newMinStatistic(initialVal pmetric.NumberDataPoint) (Statistic, error) {
	switch initialVal.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		return &minStatistic{
			minInt: initialVal.IntValue(),
			isInt:  true,
		}, nil
	case pmetric.NumberDataPointValueTypeDouble:
		return &minStatistic{
			minDouble: initialVal.DoubleValue(),
			isInt:     false,
		}, nil
	}

	return nil, errors.New("cannot create min statistic from empty datapoint")
}

func (m *minStatistic) AddDatapoint(ndp pmetric.NumberDataPoint) {
	if m.isInt {
		i := getDatapointValueInt(ndp)
		if i < m.minInt {
			m.minInt = i
		}
	} else {
		f := getDatapointValueDouble(ndp)
		if f < m.minDouble {
			m.minDouble = f
		}
	}
}

func (m *minStatistic) SetDatapointValue(dp pmetric.NumberDataPoint) {
	if m.isInt {
		dp.SetIntValue(m.minInt)
	} else {
		dp.SetDoubleValue(m.minDouble)
	}
}
