// Copyright observIQ, Inc.
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
	"strconv"
	"time"

	"github.com/observiq/bindplane-agent/expr"
	"github.com/observiq/bindplane-agent/receiver/routereceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// ottlExtractProcessor is a processor that extracts metrics from logs using OTTL expressions
type ottlExtractProcessor struct {
	config    *Config
	ottlMatch *expr.OTTLCondition[ottllog.TransformContext]
	ottlValue *expr.OTTLExpression[ottllog.TransformContext]
	ottlAttrs *expr.OTTLAttributeMap[ottllog.TransformContext]
	consumer  consumer.Logs
	logger    *zap.Logger
}

// newProcessor returns a new processor for .
func newOTTLExtractProcessor(
	config *Config,
	consumer consumer.Logs,
	match *expr.OTTLCondition[ottllog.TransformContext],
	value *expr.OTTLExpression[ottllog.TransformContext],
	attrs *expr.OTTLAttributeMap[ottllog.TransformContext],
	logger *zap.Logger) *ottlExtractProcessor {
	return &ottlExtractProcessor{
		config:    config,
		ottlMatch: match,
		ottlValue: value,
		ottlAttrs: attrs,
		consumer:  consumer,
		logger:    logger,
	}
}

// Start starts the processor.
func (e *ottlExtractProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Capabilities returns the consumer's capabilities.
func (e *ottlExtractProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// Shutdown stops the processor.
func (e *ottlExtractProcessor) Shutdown(_ context.Context) error {
	return nil
}

// ConsumeLogs processes the logs.
func (e *ottlExtractProcessor) ConsumeLogs(ctx context.Context, pl plog.Logs) error {
	metrics := e.extractMetrics(ctx, pl)
	if metrics.ResourceMetrics().Len() != 0 {
		e.sendMetrics(ctx, metrics)
	}

	return e.consumer.ConsumeLogs(ctx, pl)
}

func (e *ottlExtractProcessor) extractMetrics(ctx context.Context, pl plog.Logs) pmetric.Metrics {
	metrics := pmetric.NewMetrics()

	resourceLogs := pl.ResourceLogs()
	for i := 0; i < resourceLogs.Len(); i++ {
		resourceLog := resourceLogs.At(i)
		scopeLogs := resourceLog.ScopeLogs()
		resource := resourceLog.Resource()

		resourceMetrics := pmetric.NewResourceMetrics()
		resource.Attributes().CopyTo(resourceMetrics.Resource().Attributes())

		scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
		scopeMetrics.Scope().SetName(typeStr)

		metric := scopeMetrics.Metrics().AppendEmpty()
		metric.SetName(e.config.MetricName)
		metric.SetUnit(e.config.MetricUnit)

		var dpSlice pmetric.NumberDataPointSlice = pmetric.NewNumberDataPointSlice()
		switch e.config.MetricType {
		case gaugeDoubleType, gaugeIntType:
			dpSlice = metric.SetEmptyGauge().DataPoints()
		case counterDoubleType, counterIntType:
			dpSlice = metric.SetEmptySum().DataPoints()
		}

		for j := 0; j < scopeLogs.Len(); j++ {
			scopeLog := scopeLogs.At(j)
			logRecords := scopeLog.LogRecords()
			for k := 0; k < logRecords.Len(); k++ {
				lr := logRecords.At(k)
				logCtx := ottllog.NewTransformContext(lr, scopeLog.Scope(), resource)

				matches, err := e.ottlMatch.Match(ctx, logCtx)
				if err != nil {
					e.logger.Error("Failed when executing ottl match statement.", zap.Error(err))
					continue
				}

				if !matches {
					continue
				}

				e.addDatapointOTTL(ctx, lr, logCtx, dpSlice)
			}
		}

		if dpSlice.Len() != 0 {
			// Add the resource metric to the slice if we had any datapoints.
			resourceMetrics.MoveTo(metrics.ResourceMetrics().AppendEmpty())
		}
	}

	return metrics
}

func (e *ottlExtractProcessor) addDatapointOTTL(ctx context.Context, lr plog.LogRecord, logCtx ottllog.TransformContext, dpSlice pmetric.NumberDataPointSlice) {
	val, err := e.ottlValue.Execute(ctx, logCtx)
	if err != nil {
		e.logger.Error("Failed when extracting value.", zap.Error(err))
		return
	}

	if val == nil {
		return
	}

	attrs := e.ottlAttrs.ExtractAttributes(ctx, logCtx)

	dp := pmetric.NewNumberDataPoint()
	err = dp.Attributes().FromRaw(attrs)
	if err != nil {
		e.logger.Error("Failed when setting attributes.", zap.Error(err))
		return
	}

	dp.SetTimestamp(extractTimestampFromLogRecord(lr))
	switch e.config.MetricType {
	case gaugeDoubleType, counterDoubleType:
		floatVal, err := convertAnyToFloat(val)
		if err != nil {
			e.logger.Error("Failed when parsing float.", zap.Error(err))
			return
		}

		dp.SetDoubleValue(floatVal)
	case gaugeIntType, counterIntType:
		intVal, err := convertAnyToInt(val)
		if err != nil {
			e.logger.Error("Failed when parsing integer.", zap.Error(err))
			return
		}

		dp.SetIntValue(intVal)
	}

	// Successfully constructed dp, we can add it to the slice
	dp.MoveTo(dpSlice.AppendEmpty())
}

// sendMetrics sends metrics to the configured route.
func (e *ottlExtractProcessor) sendMetrics(ctx context.Context, metrics pmetric.Metrics) {
	err := routereceiver.RouteMetrics(ctx, e.config.Route, metrics)
	if err != nil {
		e.logger.Error("Failed to send metrics", zap.Error(err))
	}
}

func extractTimestampFromLogRecord(lr plog.LogRecord) pcommon.Timestamp {
	if ts := lr.Timestamp(); ts != 0 {
		return ts
	}

	if ts := lr.ObservedTimestamp(); ts != 0 {
		return ts
	}

	return pcommon.NewTimestampFromTime(time.Now())
}

func convertAnyToInt(value any) (int64, error) {
	switch value := value.(type) {
	case int:
		return int64(value), nil
	case int32:
		return int64(value), nil
	case int64:
		return value, nil
	case float32:
		return int64(value), nil
	case float64:
		return int64(value), nil
	case string:
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i, nil
		}
		return 0, fmt.Errorf("failed to convert string to int: %s", value)
	default:
		return 0, fmt.Errorf("invalid value type: %T", value)
	}
}

func convertAnyToFloat(value any) (float64, error) {
	switch value := value.(type) {
	case int:
		return float64(value), nil
	case int32:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case float32:
		return float64(value), nil
	case float64:
		return value, nil
	case string:
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f, nil
		}
		return 0, fmt.Errorf("failed to convert string to float: %s", value)
	default:
		return 0, fmt.Errorf("invalid value type: %T", value)
	}
}
