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
	// CreateGaugeMetricTableTemplate is SQL to create a table for gauge metrics in Snowflake
	CreateGaugeMetricTableTemplate = `
	CREATE TABLE IF NOT EXISTS "%s"."%s"."%s_gauge" (
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
		"Value" NUMBER,
		"Flags" INT,
		"ExemplarAttributes" VARCHAR,
		"ExemplarTimestamps" VARCHAR,
		"ExemplarTraceIDs" VARCHAR,
		"ExemplarSpanIDs" VARCHAR,
		"ExemplarValues" VARCHAR
	);`

	// InsertIntoGaugeMetricTableTemplate is SQL to insert a data point into the gauge table
	InsertIntoGaugeMetricTableTemplate = `
	INSERT INTO "%s"."%s"."%s_gauge" (
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

// GaugeModel implements MetricModel
type GaugeModel struct {
	logger    *zap.Logger
	gauges    []*gaugeData
	insertSQL string
}

type gaugeData struct {
	resource pmetric.ResourceMetrics
	scope    pmetric.ScopeMetrics
	metric   pmetric.Metric
	gauge    pmetric.Gauge
}

// NewGaugeModel returns a newly created GaugeModel
func NewGaugeModel(logger *zap.Logger, sql string) *GaugeModel {
	return &GaugeModel{
		logger:    logger,
		insertSQL: sql,
	}
}

// AddMetric will add a new gauge metric to this model
func (gm *GaugeModel) AddMetric(r pmetric.ResourceMetrics, s pmetric.ScopeMetrics, m pmetric.Metric) {
	gm.gauges = append(gm.gauges, &gaugeData{
		resource: r,
		scope:    s,
		metric:   m,
		gauge:    m.Gauge(),
	})
}

// BatchInsert will insert the available gauge metrics nad their data points into Snowflake
func (gm *GaugeModel) BatchInsert(ctx context.Context, db database.Database) error {
	gm.logger.Debug("starting GaugeModel BatchInsert")
	if len(gm.gauges) == 0 {
		gm.logger.Debug("end GaugeModel BatchInsert: no gauge metrics to insert")
		return nil
	}

	gaugeMaps := []map[string]any{}
	for _, g := range gm.gauges {
		for i := 0; i < g.gauge.DataPoints().Len(); i++ {
			dp := g.gauge.DataPoints().At(i)

			var value any
			if dp.ValueType() == pmetric.NumberDataPointValueTypeInt {
				value = dp.IntValue()
			} else {
				value = dp.DoubleValue()
			}

			eAttributes, eTimestamps, eTraceIDs, eSpanIDs, eValues := utility.FlattenExemplars(dp.Exemplars())

			gaugeMaps = append(gaugeMaps, map[string]any{
				"rSchema":        g.resource.SchemaUrl(),
				"rDroppedCount":  g.resource.Resource().DroppedAttributesCount(),
				"rAttributes":    utility.ConvertAttributesToString(g.resource.Resource().Attributes(), gm.logger),
				"sSchema":        g.scope.SchemaUrl(),
				"sName":          g.scope.Scope().Name(),
				"sVersion":       g.scope.Scope().Version(),
				"sDroppedCount":  g.scope.Scope().DroppedAttributesCount(),
				"sAttributes":    utility.ConvertAttributesToString(g.scope.Scope().Attributes(), gm.logger),
				"mName":          g.metric.Name(),
				"mDescription":   g.metric.Description(),
				"mUnit":          g.metric.Unit(),
				"attributes":     utility.ConvertAttributesToString(dp.Attributes(), gm.logger),
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

	gm.logger.Debug("GaugeModel calling utility.batchInsert")
	err := db.BatchInsert(ctx, gaugeMaps, gm.insertSQL)
	if err != nil {
		return fmt.Errorf("failed to insert gauge metric data: %w", err)
	}
	gm.logger.Debug("end GaugeModel BatchInsert: successful insert")
	return nil
}
