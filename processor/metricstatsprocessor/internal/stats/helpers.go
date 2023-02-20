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

import "go.opentelemetry.io/collector/pdata/pmetric"

func getDatapointValueDouble(ndp pmetric.NumberDataPoint) float64 {
	switch ndp.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		return float64(ndp.IntValue())
	case pmetric.NumberDataPointValueTypeDouble:
		return ndp.DoubleValue()
	}

	// Empty number datapoint, we'll just return 0 in this case.
	// It's up to the caller to handle this case correctly.
	return 0
}

func getDatapointValueInt(ndp pmetric.NumberDataPoint) int64 {
	switch ndp.ValueType() {
	case pmetric.NumberDataPointValueTypeInt:
		return ndp.IntValue()
	case pmetric.NumberDataPointValueTypeDouble:
		return int64(ndp.DoubleValue())
	}

	// Empty number datapoint, we'll just return 0 in this case.
	// It's up to the caller to handle this case correctly.
	return 0
}
