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

package aggregate

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

type maxAggregation struct {
	maxDouble float64
	maxInt    int64
	isInt     bool
}

func newMaxAggregate(initialVal pmetric.NumberDataPoint) (Aggregate, error) {
	switch initialVal.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		return &maxAggregation{
			maxInt: initialVal.IntValue(),
			isInt:  true,
		}, nil
	case pmetric.NumberDataPointValueTypeDouble:
		return &maxAggregation{
			maxDouble: initialVal.DoubleValue(),
			isInt:     false,
		}, nil
	}

	return nil, errors.New("cannot create max aggregation from empty datapoint")
}

func (m *maxAggregation) AddDatapoint(ndp pmetric.NumberDataPoint) {
	if m.isInt {
		i := getDatapointValueInt(ndp)
		if i > m.maxInt {
			m.maxInt = i
		}
	} else {
		f := getDatapointValueDouble(ndp)
		if f > m.maxDouble {
			m.maxDouble = f
		}
	}
}

func (m *maxAggregation) SetDatapointValue(dp pmetric.NumberDataPoint) {
	if m.isInt {
		dp.SetIntValue(m.maxInt)
	} else {
		dp.SetDoubleValue(m.maxDouble)
	}
}
