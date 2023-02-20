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

package metricstatsprocessor

import (
	"github.com/observiq/observiq-otel-collector/processor/metricstatsprocessor/internal/stats"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type resourceMetadata struct {
	resource pcommon.Map
	// metric name -> metric metadata
	metrics map[string]*metricMetadata
}

type metricMetadata struct {
	name       string
	desc       string
	unit       string
	metricType pmetric.MetricType
	// Only relevant to sum metrics
	monotonic bool
	// Map of attributes hash to datapointMetadata
	datapoints map[uint64]*datapointMetadata
}

type datapointMetadata struct {
	attributes pcommon.Map
	statistics map[stats.StatType]stats.Statistic
}
