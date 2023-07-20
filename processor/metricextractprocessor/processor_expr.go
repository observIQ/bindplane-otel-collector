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

package metricextractprocessor

import (
	"context"
	"fmt"
	"time"

	"github.com/observiq/bindplane-agent/expr"
	"github.com/observiq/bindplane-agent/receiver/routereceiver"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// exprExtractProcessor is a processor that extracts metrics from logs.
type exprExtractProcessor struct {
	config   *Config
	match    *expr.Expression
	value    *expr.Expression
	attrs    *expr.ExpressionMap
	consumer consumer.Logs
	logger   *zap.Logger
}

// newExprExtractProcessor returns a new processor for expr expressions.
func newExprExtractProcessor(config *Config, consumer consumer.Logs, match, value *expr.Expression, attrs *expr.ExpressionMap, logger *zap.Logger) *exprExtractProcessor {
	return &exprExtractProcessor{
		config:   config,
		match:    match,
		value:    value,
		attrs:    attrs,
		consumer: consumer,
		logger:   logger,
	}
}

// Start starts the processor.
func (e *exprExtractProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Capabilities returns the consumer's capabilities.
func (e *exprExtractProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// Shutdown stops the processor.
func (e *exprExtractProcessor) Shutdown(_ context.Context) error {
	return nil
}

// ConsumeLogs processes the logs.
func (e *exprExtractProcessor) ConsumeLogs(ctx context.Context, pl plog.Logs) error {
	metrics := e.extractMetrics(pl)
	if metrics.ResourceMetrics().Len() != 0 {
		e.sendMetrics(ctx, metrics)
	}

	return e.consumer.ConsumeLogs(ctx, pl)
}

// extractMetrics extracts metrics from logs.
func (e *exprExtractProcessor) extractMetrics(pl plog.Logs) pmetric.Metrics {
	recordGroups := expr.ConvertToResourceGroups(pl)

	metrics := pmetric.NewMetrics()
	for _, group := range recordGroups {
		dataPoints := e.extractDataPoints(group.Records)
		if dataPoints.Len() == 0 {
			continue
		}

		metricResource := pmetric.NewResourceMetrics()
		err := metricResource.Resource().Attributes().FromRaw(group.Resource)
		if err != nil {
			e.logger.Error("Failed to convert resource attributes", zap.Error(err))
			continue
		}

		scopeMetrics := metricResource.ScopeMetrics().AppendEmpty()
		scopeMetrics.Scope().SetName(typeStr)

		metric := scopeMetrics.Metrics().AppendEmpty()
		metric.SetName(e.config.MetricName)
		metric.SetUnit(e.config.MetricUnit)

		switch e.config.MetricType {
		case gaugeDoubleType, gaugeIntType:
			dataPoints.CopyTo(metric.SetEmptyGauge().DataPoints())
		case counterDoubleType, counterIntType:
			dataPoints.CopyTo(metric.SetEmptySum().DataPoints())
		}

		metricResource.CopyTo(metrics.ResourceMetrics().AppendEmpty())
	}

	return metrics
}

// extractDataPoints extracts data points from the records.
func (e *exprExtractProcessor) extractDataPoints(records []expr.Record) pmetric.NumberDataPointSlice {
	dataPoints := pmetric.NewNumberDataPointSlice()

	for _, record := range records {
		if !e.match.MatchRecord(record) {
			continue
		}

		dataPoint, err := e.extractDataPoint(record)
		if err != nil {
			e.logger.Error("Failed to extract data point", zap.Error(err))
			continue
		}

		dataPoint.CopyTo(dataPoints.AppendEmpty())
	}

	return dataPoints
}

// extractDataPoint extracts a data point from the record.
func (e *exprExtractProcessor) extractDataPoint(record expr.Record) (pmetric.NumberDataPoint, error) {
	switch e.config.MetricType {
	case gaugeDoubleType, counterDoubleType:
		return e.extractFloatDataPoint(record)
	case gaugeIntType, counterIntType:
		return e.extractIntDataPoint(record)
	default:
		return pmetric.NumberDataPoint{}, fmt.Errorf("invalid metric type: %s", e.config.MetricType)
	}
}

// extractIntDataPoint extracts an int data point from the record.
func (e *exprExtractProcessor) extractIntDataPoint(record expr.Record) (pmetric.NumberDataPoint, error) {
	value, err := e.value.ExtractInt(record)
	if err != nil {
		return pmetric.NumberDataPoint{}, err
	}

	timestamp := extractTimestamp(record)
	attrs := e.attrs.Extract(record)
	dataPoint := pmetric.NewNumberDataPoint()
	dataPoint.SetIntValue(value)
	dataPoint.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
	err = dataPoint.Attributes().FromRaw(attrs)

	return dataPoint, err
}

// extractFloatDataPoint extracts a float data point from the record.
func (e *exprExtractProcessor) extractFloatDataPoint(record expr.Record) (pmetric.NumberDataPoint, error) {
	value, err := e.value.ExtractFloat(record)
	if err != nil {
		return pmetric.NumberDataPoint{}, err
	}

	timestamp := extractTimestamp(record)
	attrs := e.attrs.Extract(record)
	dataPoint := pmetric.NewNumberDataPoint()
	dataPoint.SetDoubleValue(value)
	dataPoint.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
	err = dataPoint.Attributes().FromRaw(attrs)
	return dataPoint, err
}

// extractTimestamp extracts a timestamp from the record.
func extractTimestamp(record expr.Record) time.Time {
	timestamp, ok := record[expr.TimestampField].(time.Time)
	if !ok {
		return time.Now()
	}
	return timestamp
}

// sendMetrics sends metrics to the configured route.
func (e *exprExtractProcessor) sendMetrics(ctx context.Context, metrics pmetric.Metrics) {
	err := routereceiver.RouteMetrics(ctx, e.config.Route, metrics)
	if err != nil {
		e.logger.Error("Failed to send metrics", zap.Error(err))
	}
}
