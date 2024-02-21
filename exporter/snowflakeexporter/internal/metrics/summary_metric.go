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
	// CreateSummaryMetricTableTemplate is SQL to create a table for summary metrics in Snowflake
	CreateSummaryMetricTableTemplate = `
	CREATE TABLE IF NOT EXISTS "%s"."%s"."%s_summary" (
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
		"Attributes" VARCHAR,
		"StartTimestamp" TIMESTAMP_NTZ,
		"Timestamp" TIMESTAMP_NTZ,
		"Count" INT,
		"Sum" NUMBER,
		"Flags" INT,
		"Quantiles" VARCHAR,
		"Values" VARCHAR
	);`

	// InsertIntoSummaryMetricTableTemplate is SQL to insert a data point into the summary table
	InsertIntoSummaryMetricTableTemplate = `
	INSERT INTO "%s"."%s"."%s_summary" (
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
		"Attributes",
		"StartTimestamp",
		"Timestamp",
		"Count",
		"Sum",
		"Flags",
		"Quantiles",
		"Values"
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
		:attributes,
		:startTimestamp,
		:timestamp,
		:count,
		:sum,
		:flags,
		:quantiles,
		:values
	);`
)

// SummaryModel implements MetricModel
type SummaryModel struct {
	logger    *zap.Logger
	summaries []*summaryData
	insertSQL string
}

type summaryData struct {
	resource pmetric.ResourceMetrics
	scope    pmetric.ScopeMetrics
	metric   pmetric.Metric
	summary  pmetric.Summary
}

// NewSummaryModel returns a newly created SummaryModel
func NewSummaryModel(logger *zap.Logger, sql string) *SummaryModel {
	return &SummaryModel{
		logger:    logger,
		insertSQL: sql,
	}
}

// AddMetric will add a new summary metric to this model
func (sm *SummaryModel) AddMetric(r pmetric.ResourceMetrics, s pmetric.ScopeMetrics, m pmetric.Metric) {
	sm.summaries = append(sm.summaries, &summaryData{
		resource: r,
		scope:    s,
		metric:   m,
		summary:  m.Summary(),
	})
}

// BatchInsert will insert the available summary metrics and their data points into Snowflake
func (sm *SummaryModel) BatchInsert(ctx context.Context, db database.Database) error {
	sm.logger.Debug("starting SumModel BatchInsert")
	if len(sm.summaries) == 0 {
		sm.logger.Debug("end SumModel BatchInsert: no sum metrics to insert")
		return nil
	}

	summaryMaps := []map[string]any{}

	for _, s := range sm.summaries {
		for i := 0; i < s.summary.DataPoints().Len(); i++ {
			dp := s.summary.DataPoints().At(i)

			quantiles, values := flattenQuantileValues(dp.QuantileValues())

			summaryMaps = append(summaryMaps, map[string]any{
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
				"attributes":     utility.ConvertAttributesToString(dp.Attributes(), sm.logger),
				"startTimestamp": dp.StartTimestamp().AsTime(),
				"timestamp":      dp.Timestamp().AsTime(),
				"count":          dp.Count(),
				"sum":            dp.Sum(),
				"flags":          dp.Flags(),
				"quantiles":      quantiles,
				"values":         values,
			})
		}
	}

	sm.logger.Debug("SummaryModel calling utility.batchInsert")
	err := db.BatchInsert(ctx, summaryMaps, sm.insertSQL)
	if err != nil {
		return fmt.Errorf("failed to insert summary metric data: %w", err)
	}
	sm.logger.Debug("end SummaryModel BatchInsert: successful insert")
	return nil
}

func flattenQuantileValues(qv pmetric.SummaryDataPointValueAtQuantileSlice) (utility.Array, utility.Array) {
	quantiles := utility.Array{}
	values := utility.Array{}

	for i := 0; i < qv.Len(); i++ {
		quantiles = append(quantiles, qv.At(i).Quantile())
		values = append(values, qv.At(i).Value())
	}

	return quantiles, values
}
