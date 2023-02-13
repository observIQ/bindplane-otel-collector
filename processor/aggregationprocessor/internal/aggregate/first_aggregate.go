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
	"time"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

type firstAggregate struct {
	firstValInt    int64
	firstValDouble float64
	firstTimestamp time.Time
	isInt          bool
}

func newFirstAggregate(initialVal pmetric.NumberDataPoint) (Aggregate, error) {
	switch initialVal.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		return &firstAggregate{
			firstValInt:    initialVal.IntValue(),
			isInt:          true,
			firstTimestamp: initialVal.Timestamp().AsTime(),
		}, nil
	case pmetric.NumberDataPointValueTypeDouble:
		return &firstAggregate{
			firstValDouble: initialVal.DoubleValue(),
			isInt:          false,
			firstTimestamp: initialVal.Timestamp().AsTime(),
		}, nil
	}

	return nil, errors.New("cannot create first aggregation from empty datapoint")
}

func (m *firstAggregate) AddDatapoint(ndp pmetric.NumberDataPoint) {
	if ndp.Timestamp() == 0 {
		// Ignore uninitialized timestamp
		return
	}

	ndpTimestamp := ndp.Timestamp().AsTime()
	if m.firstTimestamp.After(ndpTimestamp) {
		// ndp is before this metric, so we should use it first
		if m.isInt {
			m.firstValInt = getDatapointValueInt(ndp)
		} else {
			m.firstValDouble = getDatapointValueDouble(ndp)
		}
	}
}

func (m *firstAggregate) SetDatapointValue(dp pmetric.NumberDataPoint) {
	if m.isInt {
		dp.SetIntValue(m.firstValInt)
	} else {
		dp.SetDoubleValue(m.firstValDouble)
	}
}
