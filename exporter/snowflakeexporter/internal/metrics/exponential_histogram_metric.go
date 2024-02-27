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

	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/database"
	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/utility"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

const (
	// CreateExponentialHistogramMetricTableTemplate is SQL to create a table for exponential histogram metrics in Snowflake
	CreateExponentialHistogramMetricTableTemplate = `
	CREATE TABLE IF NOT EXISTS "%s"."%s"."%s_exponential_histogram" (
		"ResourceSchemaURL" VARCHAR,
		"ResourceDroppedAttributesCount" INT,
		"ResourceAttributes" VARCHAR,
		"ScopeSchemaURL" VARCHAR,
		"ScopeName" VARCHAR,
		"ScopeVersion" VARCHAR,
		"ScopeDroppedAttributesCount" INT,
		"ScopeAttributes" VARCHAR,
		"MetricName" VARCHAR,
		"MetricDescription" VARCHAR,
		"MetricUnit" VARCHAR,
		"AggregationTemporality" VARCHAR,
		"Attributes" VARCHAR,
		"StartTimestamp" TIMESTAMP_NTZ,
		"Timestamp" TIMESTAMP_NTZ,
		"Count" INT,
		"Sum" NUMBER,
		"Scale" INT,
		"ZeroCount" INT,
		"ZeroThreshold" NUMBER,
		"Flags" INT,
		"Min" INT,
		"Max" INT,
		"PositiveOffset" INT,
		"PositiveBucketCounts" VARCHAR,
		"NegativeOffset" INT,
		"NegativeBucketCounts" VARCHAR,
		"ExemplarAttributes" VARCHAR,
		"ExemplarTimestamps" VARCHAR,
		"ExemplarTraceIDs" VARCHAR,
		"ExemplarSpanIDs" VARCHAR,
		"ExemplarValues" VARCHAR
	);`

	// InsertIntoExponentialHistogramMetricTableTemplate is SQL to insert a data point into the exponential histogram table
	InsertIntoExponentialHistogramMetricTableTemplate = `
	INSERT INTO "%s"."%s"."%s_exponential_histogram" (
		"ResourceSchemaURL",
		"ResourceDroppedAttributesCount",
		"ResourceAttributes",
		"ScopeSchemaURL",
		"ScopeName",
		"ScopeVersion",
		"ScopeDroppedAttributesCount",
		"ScopeAttributes",
		"MetricName",
		"MetricDescription",
		"MetricUnit",
		"AggregationTemporality",
		"Attributes",
		"StartTimestamp",
		"Timestamp",
		"Count",
		"Sum",
		"Scale",
		"ZeroCount",
		"ZeroThreshold",
		"Flags",
		"Min",
		"Max",
		"PositiveOffset",
		"PositiveBucketCounts",
		"NegativeOffset",
		"NegativeBucketCounts",
		"ExemplarAttributes",
		"ExemplarTimestamps",
		"ExemplarTraceIDs",
		"ExemplarSpanIDs",
		"ExemplarValues"
	) VALUES (
		:rSchema,
		:rDroppedCount,
		:rAttributes,
		:sSchema,
		:sName,
		:sVersion,
		:sDroppedCount,
		:sAttributes,
		:mName,
		:mDescription,
		:mUnit,
		:aggTemp,
		:attributes,
		:startTimestamp,
		:timestamp,
		:count,
		:sum,
		:scale,
		:zeroCount,
		:zeroThreshold,
		:flags,
		:min,
		:max,
		:positiveOffset,
		:positiveBucketCounts,
		:negativeOffset,
		:negativeBucketCounts,
		:eAttributes,
		:eTimestamps,
		:eTraceIDs,
		:eSpanIDs,
		:eValues
	);`
)

// ExponentialHistogramModel implements MetricModel
type ExponentialHistogramModel struct {
	logger                *zap.Logger
	exponentialHistograms []*exponentialHistogramData
	insertSQL             string
}

type exponentialHistogramData struct {
	resource             pmetric.ResourceMetrics
	scope                pmetric.ScopeMetrics
	metric               pmetric.Metric
	exponentialHistogram pmetric.ExponentialHistogram
}

// NewExponentialHistogramModel returns a newly created ExponentialHistogramModel
func NewExponentialHistogramModel(logger *zap.Logger, sql string) *ExponentialHistogramModel {
	return &ExponentialHistogramModel{
		logger:    logger,
		insertSQL: sql,
	}
}

// AddMetric will add a new exponential histogram metric to this model
func (ehm *ExponentialHistogramModel) AddMetric(r pmetric.ResourceMetrics, s pmetric.ScopeMetrics, m pmetric.Metric) {
	ehm.exponentialHistograms = append(ehm.exponentialHistograms, &exponentialHistogramData{
		resource:             r,
		scope:                s,
		metric:               m,
		exponentialHistogram: m.ExponentialHistogram(),
	})
}

// BatchInsert will insert the available exponential histogram metrics and their data points into Snowflake
func (ehm *ExponentialHistogramModel) BatchInsert(ctx context.Context, db database.Database) error {
	ehm.logger.Debug("starting ExponentialHistogramModel BatchInsert")
	if len(ehm.exponentialHistograms) == 0 {
		ehm.logger.Debug("end ExponentialHistogramModel BatchInsert: no exponential histogram metrics to insert")
		return nil
	}

	exponentialHistogramMaps := []map[string]any{}
	for _, eh := range ehm.exponentialHistograms {
		for i := 0; i < eh.exponentialHistogram.DataPoints().Len(); i++ {
			dp := eh.exponentialHistogram.DataPoints().At(i)

			eAttributes, eTimestamps, eTraceIDs, eSpanIDs, eValues := utility.FlattenExemplars(dp.Exemplars())

			exponentialHistogramMaps = append(exponentialHistogramMaps, map[string]any{
				"rSchema":              eh.resource.SchemaUrl(),
				"rDroppedCount":        eh.resource.Resource().DroppedAttributesCount(),
				"rAttributes":          utility.ConvertAttributesToString(eh.resource.Resource().Attributes(), ehm.logger),
				"sSchema":              eh.scope.SchemaUrl(),
				"sName":                eh.scope.Scope().Name(),
				"sVersion":             eh.scope.Scope().Version(),
				"sDroppedCount":        eh.scope.Scope().DroppedAttributesCount(),
				"sAttributes":          utility.ConvertAttributesToString(eh.scope.Scope().Attributes(), ehm.logger),
				"mName":                eh.metric.Name(),
				"mDescription":         eh.metric.Description(),
				"mUnit":                eh.metric.Unit(),
				"aggTemp":              eh.exponentialHistogram.AggregationTemporality().String(),
				"attributes":           utility.ConvertAttributesToString(dp.Attributes(), ehm.logger),
				"startTimestamp":       dp.Timestamp().AsTime(),
				"timestamp":            dp.Timestamp().AsTime(),
				"count":                dp.Count(),
				"sum":                  dp.Sum(),
				"scale":                dp.Scale(),
				"zeroCount":            dp.ZeroCount(),
				"zeroThreshold":        dp.ZeroThreshold(),
				"flags":                dp.Flags(),
				"min":                  dp.Min(),
				"max":                  dp.Max(),
				"positiveOffset":       dp.Positive().Offset(),
				"positiveBucketCounts": dp.Positive().BucketCounts().AsRaw(),
				"negativeOffset":       dp.Negative().Offset(),
				"negativeBucketCounts": dp.Negative().BucketCounts().AsRaw(),
				"eAttributes":          eAttributes,
				"eTimestamps":          eTimestamps,
				"eTraceIDs":            eTraceIDs,
				"eSpanIDs":             eSpanIDs,
				"eValues":              eValues,
			})
		}
	}

	ehm.logger.Debug("ExponentialHistogramModel calling utility.batchInsert")
	err := db.BatchInsert(ctx, exponentialHistogramMaps, ehm.insertSQL)
	if err != nil {
		return fmt.Errorf("failed to insert exponential histogram metric data: %w", err)
	}
	ehm.logger.Debug("end ExponentialHistogramModel BatchInsert: successful insert")
	return nil
}
