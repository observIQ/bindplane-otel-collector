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
	// CreateHistogramMetricTableTemplate is SQL to create a table for histogram metrics in Snowflake
	CreateHistogramMetricTableTemplate = `
	CREATE TABLE IF NOT EXISTS "%s"."%s"."%s_histogram" (
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
		"Flags" INT,
		"Min" INT,
		"Max" INT,
		"BucketCounts" VARCHAR,
		"ExplicitBounds" VARCHAR,
		"ExemplarAttributes" VARCHAR,
		"ExemplarTimestamps" VARCHAR,
		"ExemplarTraceIDs" VARCHAR,
		"ExemplarSpanIDs" VARCHAR,
		"ExemplarValues" VARCHAR
	);`

	// InsertIntoHistogramMetricTableTemplate is SQL to insert a data point into the histogram table
	InsertIntoHistogramMetricTableTemplate = `
	INSERT INTO "%s"."%s"."%s_histogram" (
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
		"Flags",
		"Min",
		"Max",
		"BucketCounts",
		"ExplicitBounds",
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
		:flags,
		:min,
		:max,
		:bucketCounts,
		:explicitBounds,
		:eAttributes,
		:eTimestamps,
		:eTraceIDs,
		:eSpanIDs,
		:eValues
	);`
)

// HistogramModel implements MetricModel
type HistogramModel struct {
	logger     *zap.Logger
	histograms []*histogramData
	insertSQL  string
}

type histogramData struct {
	resource  pmetric.ResourceMetrics
	scope     pmetric.ScopeMetrics
	metric    pmetric.Metric
	histogram pmetric.Histogram
}

// NewHistogramModel returns a newly created HistogramModel
func NewHistogramModel(logger *zap.Logger, sql string) *HistogramModel {
	return &HistogramModel{
		logger:    logger,
		insertSQL: sql,
	}
}

// AddMetric will add a new histogram metric to this model
func (hm *HistogramModel) AddMetric(r pmetric.ResourceMetrics, s pmetric.ScopeMetrics, m pmetric.Metric) {
	hm.histograms = append(hm.histograms, &histogramData{
		resource:  r,
		scope:     s,
		metric:    m,
		histogram: m.Histogram(),
	})
}

// BatchInsert will insert the available histogram metrics and their data points into Snowflake
func (hm *HistogramModel) BatchInsert(ctx context.Context, db database.Database) error {
	hm.logger.Debug("starting HistogramModel BatchInsert")
	if len(hm.histograms) == 0 {
		hm.logger.Debug("end HistogramModel BatchInsert: no histogram metrics to insert")
		return nil
	}

	histogramMaps := []map[string]any{}
	for _, h := range hm.histograms {
		for i := 0; i < h.histogram.DataPoints().Len(); i++ {
			dp := h.histogram.DataPoints().At(i)

			eAttributes, eTimestamps, eTraceIDs, eSpanIDs, eValues := utility.FlattenExemplars(dp.Exemplars())

			histogramMaps = append(histogramMaps, map[string]any{
				"rSchema":        h.resource.SchemaUrl(),
				"rDroppedCount":  h.resource.Resource().DroppedAttributesCount(),
				"rAttributes":    utility.ConvertAttributesToString(h.resource.Resource().Attributes(), hm.logger),
				"sSchema":        h.scope.SchemaUrl(),
				"sName":          h.scope.Scope().Name(),
				"sVersion":       h.scope.Scope().Version(),
				"sDroppedCount":  h.scope.Scope().DroppedAttributesCount(),
				"sAttributes":    utility.ConvertAttributesToString(h.scope.Scope().Attributes(), hm.logger),
				"mName":          h.metric.Name(),
				"mDescription":   h.metric.Description(),
				"mUnit":          h.metric.Unit(),
				"aggTemp":        h.histogram.AggregationTemporality().String(),
				"attributes":     utility.ConvertAttributesToString(dp.Attributes(), hm.logger),
				"startTimestamp": dp.Timestamp().AsTime(),
				"timestamp":      dp.Timestamp().AsTime(),
				"count":          dp.Count(),
				"sum":            dp.Sum(),
				"flags":          dp.Flags(),
				"min":            dp.Min(),
				"max":            dp.Max(),
				"bucketCounts":   dp.BucketCounts().AsRaw(),
				"explicitBounds": dp.ExplicitBounds().AsRaw(),
				"eAttributes":    eAttributes,
				"eTimestamps":    eTimestamps,
				"eTraceIDs":      eTraceIDs,
				"eSpanIDs":       eSpanIDs,
				"eValues":        eValues,
			})
		}
	}

	hm.logger.Debug("HistogramModel calling utility.batchInsert")
	err := db.BatchInsert(ctx, histogramMaps, hm.insertSQL)
	if err != nil {
		return fmt.Errorf("failed to insert histogram metric data: %w", err)
	}
	hm.logger.Debug("end HistogramModel BatchInsert: successful insert")
	return nil
}
