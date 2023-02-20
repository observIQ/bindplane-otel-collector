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
	"time"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

type lastStatistic struct {
	lastValInt    int64
	lastValDouble float64
	lastTimestamp time.Time
	isInt         bool
}

func newLastStatistic(initialVal pmetric.NumberDataPoint) (Statistic, error) {
	switch initialVal.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		return &lastStatistic{
			lastValInt:    initialVal.IntValue(),
			isInt:         true,
			lastTimestamp: initialVal.Timestamp().AsTime(),
		}, nil
	case pmetric.NumberDataPointValueTypeDouble:
		return &lastStatistic{
			lastValDouble: initialVal.DoubleValue(),
			isInt:         false,
			lastTimestamp: initialVal.Timestamp().AsTime(),
		}, nil
	}

	return nil, errors.New("cannot create last statistic from empty datapoint")
}

func (m *lastStatistic) AddDatapoint(ndp pmetric.NumberDataPoint) {
	ndpTimestamp := ndp.Timestamp().AsTime()
	// Note for the Equal here: If two datapoints have the same timestamp, we consider the last
	// datapoint we receive from the pipeline to be the "last" datapoint.
	if ndpTimestamp.After(m.lastTimestamp) || ndpTimestamp.Equal(m.lastTimestamp) {
		// ndp is after this metric, so we should use it as the last value
		if m.isInt {
			m.lastValInt = getDatapointValueInt(ndp)
		} else {
			m.lastValDouble = getDatapointValueDouble(ndp)
		}

		m.lastTimestamp = ndpTimestamp
	}
}

func (m *lastStatistic) SetDatapointValue(dp pmetric.NumberDataPoint) {
	if m.isInt {
		dp.SetIntValue(m.lastValInt)
	} else {
		dp.SetDoubleValue(m.lastValDouble)
	}
}
