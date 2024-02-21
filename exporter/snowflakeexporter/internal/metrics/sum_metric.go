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
	// CreateSumMetricTableTemplate is SQL to create a table for sum metrics in Snowflake
	CreateSumMetricTableTemplate = `
	CREATE TABLE IF NOT EXISTS "%s"."%s"."%s_sum" (
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
		"IsMonotonic" BOOLEAN,
		"Attributes" VARCHAR,
		"StartTimestamp" TIMESTAMP_NTZ,
		"Timestamp" TIMESTAMP_NTZ,
		"Value" NUMBER,
		"Flags" INT,
		"ExemplarAttributes" VARCHAR,
		"ExemplarTimestamps" VARCHAR,
		"ExemplarTraceIDs" VARCHAR,
		"ExemplarSpanIDs" VARCHAR,
		"ExemplarValues" VARCHAR
	);`

	// InsertIntoSumMetricTableTemplate is SQL to insert a data point into the sum table
	InsertIntoSumMetricTableTemplate = `
	INSERT INTO "%s"."%s"."%s_sum" (
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
		"IsMonotonic",
		"Attributes",
		"StartTimestamp",
		"Timestamp",
		"Value",
		"Flags",
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
		:monotonic,
		:attributes,
		:startTimestamp,
		:timestamp,
		:value,
		:flags,
		:eAttributes,
		:eTimestamps,
		:eTraceIDs,
		:eSpanIDs,
		:eValues
	);`
)

// SumModel implements MetricModel
type SumModel struct {
	logger    *zap.Logger
	sums      []*sumData
	insertSQL string
}

type sumData struct {
	resource pmetric.ResourceMetrics
	scope    pmetric.ScopeMetrics
	metric   pmetric.Metric
	sum      pmetric.Sum
}

// NewSumModel returns a newly created SumModel
func NewSumModel(logger *zap.Logger, sql string) *SumModel {
	return &SumModel{
		logger:    logger,
		sums:      []*sumData{},
		insertSQL: sql,
	}
}

// AddMetric will add a new sum metric to this model
func (sm *SumModel) AddMetric(r pmetric.ResourceMetrics, s pmetric.ScopeMetrics, m pmetric.Metric) {
	sm.sums = append(sm.sums, &sumData{
		resource: r,
		scope:    s,
		metric:   m,
		sum:      m.Sum(),
	})
}

// BatchInsert will insert the available sum metrics and their data points into Snowflake
func (sm *SumModel) BatchInsert(ctx context.Context, db database.Database) error {
	sm.logger.Debug("starting SumModel BatchInsert")
	if len(sm.sums) == 0 {
		sm.logger.Debug("end SumModel BatchInsert: no sum metrics to insert")
		return nil
	}

	sumMaps := []map[string]any{}

	for _, s := range sm.sums {
		for i := 0; i < s.sum.DataPoints().Len(); i++ {
			dp := s.sum.DataPoints().At(i)

			var value any
			if dp.ValueType() == pmetric.NumberDataPointValueTypeInt {
				value = dp.IntValue()
			} else {
				value = dp.DoubleValue()
			}

			eAttributes, eTimestamps, eTraceIDs, eSpanIDs, eValues := utility.FlattenExemplars(dp.Exemplars())

			sumMaps = append(sumMaps, map[string]any{
				"rSchema":        s.resource.SchemaUrl(),
				"rDroppedCount":  s.resource.Resource().DroppedAttributesCount(),
				"rAttributes":    utility.ConvertAttributesToString(s.resource.Resource().Attributes(), sm.logger),
				"sSchema":        s.scope.SchemaUrl(),
				"sName":          s.scope.Scope().Name(),
				"sVersion":       s.scope.Scope().Version(),
				"sDroppedCount":  s.scope.Scope().DroppedAttributesCount(),
				"sAttributes":    utility.ConvertAttributesToString(s.scope.Scope().Attributes(), sm.logger),
				"mName":          s.metric.Name(),
				"mDescription":   s.metric.Description(),
				"mUnit":          s.metric.Unit(),
				"aggTemp":        s.sum.AggregationTemporality().String(),
				"monotonic":      s.sum.IsMonotonic(),
				"attributes":     utility.ConvertAttributesToString(dp.Attributes(), sm.logger),
				"startTimestamp": dp.StartTimestamp().AsTime(),
				"timestamp":      dp.Timestamp().AsTime(),
				"value":          value,
				"flags":          dp.Flags(),
				"eAttributes":    eAttributes,
				"eTimestamps":    eTimestamps,
				"eTraceIDs":      eTraceIDs,
				"eSpanIDs":       eSpanIDs,
				"eValues":        eValues,
			})
		}
	}

	sm.logger.Debug("SumModel calling utility.batchInsert")
	err := db.BatchInsert(ctx, sumMaps, sm.insertSQL)
	if err != nil {
		return fmt.Errorf("failed to insert sum metric data: %w", err)
	}
	sm.logger.Debug("end SumModel BatchInsert: successful insert")
	return nil
}
