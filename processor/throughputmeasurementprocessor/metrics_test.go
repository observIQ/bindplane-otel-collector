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

package throughputmeasurementprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricViews(t *testing.T) {
	expectedViewNames := []string{
		"processor_throughputmeasurement_log_data_size",
		"processor_throughputmeasurement_metric_data_size",
		"processor_throughputmeasurement_trace_data_size",
		"processor_throughputmeasurement_log_count",
		"processor_throughputmeasurement_metric_count",
		"processor_throughputmeasurement_trace_count",
	}

	views := metricViews()
	for i, viewName := range expectedViewNames {
		assert.Equal(t, viewName, views[i].Name)
	}
}
