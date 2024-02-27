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

package snowflakeexporter

import (
	"context"
	"fmt"

	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/database"
	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/utility"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

const (
	createTracesTableSnowflakeTemplate = `
	CREATE TABLE IF NOT EXISTS "%s"."%s"."%s" (
		"ResourceSchemaURL" VARCHAR,
		"ResourceDroppedAttributesCount" INT,
		"ResourceAttributes" VARCHAR,
		"ScopeSchemaURL" VARCHAR,
		"ScopeName" VARCHAR,
		"ScopeVersion" VARCHAR,
		"ScopeDroppedAttributesCount" INT,
		"ScopeAttributes" VARCHAR,
		"TraceID" VARCHAR,
		"SpanID" VARCHAR,
		"TraceState" VARCHAR,
		"ParentSpanID" BINARY,
		"Name" VARCHAR,
		"Kind" VARCHAR,
		"StartTime" TIMESTAMP_NTZ,
		"EndTime" TIMESTAMP_NTZ,
		"DroppedAttributesCount" INT,
		"Attributes" VARCHAR,
		"StatusMessage" VARCHAR,
		"StatusCode" VARCHAR,
		"EventTimes" VARCHAR,
		"EventNames" VARCHAR,
		"EventDroppedAttributesCount" VARCHAR,
		"EventAttributes" VARCHAR,
		"LinkTraceID" VARCHAR,
		"LinkSpanID" VARCHAR,
		"LinkTraceState" VARCHAR,
		"LinkDroppedAttributesCount" VARCHAR,
		"LinkAttributes" VARCHAR
	);`

	insertIntoTracesTableSnowflakeTemplate = `
	INSERT INTO "%s"."%s"."%s" (
		"ResourceSchemaURL",
		"ResourceDroppedAttributesCount",
		"ResourceAttributes",
		"ScopeSchemaURL",
		"ScopeName",
		"ScopeVersion",
		"ScopeDroppedAttributesCount",
		"ScopeAttributes",
		"TraceID",
		"SpanID",
		"TraceState",
		"ParentSpanID",
		"Name",
		"Kind",
		"StartTime",
		"EndTime",
		"DroppedAttributesCount",
		"Attributes",
		"StatusMessage",
		"StatusCode",
		"EventTimes",
		"EventNames",
		"EventDroppedAttributesCount",
		"EventAttributes",
		"LinkTraceID",
		"LinkSpanID",
		"LinkTraceState",
		"LinkDroppedAttributesCount",
		"LinkAttributes"
	) VALUES (
		:rSchema,
		:rDroppedCount,
		:rAttributes,
		:sSchema,
		:sName,
		:sVersion,
		:sDroppedCount,
		:sAttributes,
		:traceID,
		:spanID,
		:traceState,
		:parentSpanID,
		:name,
		:kind,
		:startTime,
		:endTime,
		:droppedCount,
		:attributes,
		:statusMessage,
		:statusCode,
		:eventTimes,
		:eventNames,
		:eventDroppedCount,
		:eventAttributes,
		:linkTraceIDs,
		:linkSpanIDs,
		:linkTraceStates,
		:linkDroppedCount,
		:linkAttributes
	);`
)

type tracesExporter struct {
	cfg       *Config
	logger    *zap.Logger
	db        database.Database
	insertSQL string
}

func newTracesExporter(
	_ context.Context,
	cfg *Config,
	params exporter.CreateSettings,
	newDatabase func(dsn, wh, db string) (database.Database, error),
) (*tracesExporter, error) {
	db, err := newDatabase(cfg.dsn, cfg.Warehouse, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create new database connection for traces: %w", err)
	}

	return &tracesExporter{
		cfg:       cfg,
		logger:    params.Logger,
		db:        db,
		insertSQL: fmt.Sprintf(insertIntoTracesTableSnowflakeTemplate, cfg.Database, cfg.Traces.Schema, cfg.Traces.Table),
	}, nil
}

func (te *tracesExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (te *tracesExporter) start(ctx context.Context, _ component.Host) error {
	err := te.db.InitDatabaseConn(ctx, te.cfg.Role)
	if err != nil {
		return fmt.Errorf("failed to initialize database connection for traces: %w", err)
	}

	err = te.db.CreateSchema(ctx, te.cfg.Traces.Schema)
	if err != nil {
		return fmt.Errorf("failed to create traces schema: %w", err)
	}

	err = te.db.CreateTable(ctx, fmt.Sprintf(createTracesTableSnowflakeTemplate, te.cfg.Database, te.cfg.Traces.Schema, te.cfg.Traces.Table))
	if err != nil {
		return fmt.Errorf("failed to create traces table: %w", err)
	}

	return nil
}

func (te *tracesExporter) shutdown(_ context.Context) error {
	if te.db != nil {
		return te.db.Close()
	}
	return nil
}

func (te *tracesExporter) tracesDataPusher(ctx context.Context, td ptrace.Traces) error {
	te.logger.Debug("begin tracesDataPusher")
	traceMaps := make([]map[string]any, 0, td.SpanCount())
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resourceSpan := td.ResourceSpans().At(i)

		for j := 0; j < resourceSpan.ScopeSpans().Len(); j++ {
			scopeSpan := resourceSpan.ScopeSpans().At(j)

			for k := 0; k < scopeSpan.Spans().Len(); k++ {
				span := scopeSpan.Spans().At(k)
				eTimes, eNames, eAttributes, eDroppedCount := flattenEvents(span.Events())
				lTraceIDs, lSpanIDs, lTraceStates, lAttributes, lDroppedCount := flattenLinks(span.Links())

				traceMaps = append(traceMaps, map[string]any{
					"rSchema":           resourceSpan.SchemaUrl(),
					"rDroppedCount":     resourceSpan.Resource().DroppedAttributesCount(),
					"rAttributes":       utility.ConvertAttributesToString(resourceSpan.Resource().Attributes(), te.logger),
					"sSchema":           scopeSpan.SchemaUrl(),
					"sName":             scopeSpan.Scope().Name(),
					"sVersion":          scopeSpan.Scope().Version(),
					"sDroppedCount":     scopeSpan.Scope().DroppedAttributesCount(),
					"sAttributes":       utility.ConvertAttributesToString(scopeSpan.Scope().Attributes(), te.logger),
					"traceID":           utility.TraceIDToHexOrEmptyString(span.TraceID()),
					"spanID":            utility.SpanIDToHexOrEmptyString(span.SpanID()),
					"traceState":        span.TraceState().AsRaw(),
					"parentSpanID":      utility.SpanIDToHexOrEmptyString(span.ParentSpanID()),
					"name":              span.Name(),
					"kind":              span.Kind().String(),
					"startTime":         span.StartTimestamp().AsTime(),
					"endTime":           span.EndTimestamp().AsTime(),
					"droppedCount":      span.DroppedAttributesCount(),
					"attributes":        utility.ConvertAttributesToString(span.Attributes(), te.logger),
					"statusMessage":     span.Status().Message(),
					"statusCode":        span.Status().Code().String(),
					"eventTimes":        eTimes,
					"eventNames":        eNames,
					"eventDroppedCount": eDroppedCount,
					"eventAttributes":   eAttributes,
					"linkTraceIDs":      lTraceIDs,
					"linkSpanIDs":       lSpanIDs,
					"linkTraceStates":   lTraceStates,
					"linkDroppedCount":  lDroppedCount,
					"linkAttributes":    lAttributes,
				})
			}
		}
	}

	err := te.db.BatchInsert(ctx, traceMaps, te.insertSQL)
	if err != nil {
		return fmt.Errorf("failed to insert trace data: %w", err)
	}
	te.logger.Debug("end tracesDataPusher")
	return nil
}

func flattenEvents(e ptrace.SpanEventSlice) (utility.Array, utility.Array, utility.Array, utility.Array) {
	times := utility.Array{}
	names := utility.Array{}
	attrs := utility.Array{}
	dropped := utility.Array{}

	for i := 0; i < e.Len(); i++ {
		times = append(times, e.At(i).Timestamp().AsTime())
		names = append(names, e.At(i).Name())
		attrs = append(attrs, e.At(i).Attributes().AsRaw())
		dropped = append(dropped, e.At(i).DroppedAttributesCount())
	}

	return times, names, attrs, dropped
}

func flattenLinks(li ptrace.SpanLinkSlice) (utility.Array, utility.Array, utility.Array, utility.Array, utility.Array) {
	traceIDs := utility.Array{}
	spanIDs := utility.Array{}
	traceStates := utility.Array{}
	attrs := utility.Array{}
	dropped := utility.Array{}

	for i := 0; i < li.Len(); i++ {
		traceIDs = append(traceIDs, utility.TraceIDToHexOrEmptyString(li.At(i).TraceID()))
		spanIDs = append(spanIDs, utility.SpanIDToHexOrEmptyString(li.At(i).SpanID()))
		traceStates = append(traceStates, li.At(i).TraceState().AsRaw())
		attrs = append(attrs, li.At(i).Attributes().AsRaw())
		dropped = append(dropped, li.At(i).DroppedAttributesCount())
	}

	return traceIDs, spanIDs, traceStates, attrs, dropped
}
