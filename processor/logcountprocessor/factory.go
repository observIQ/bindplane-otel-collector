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

package logcountprocessor

import (
	"context"
	"fmt"

	"github.com/observiq/bindplane-agent/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

// componentType is the value of the "type" key in configuration.
var componentType = component.MustNewType("logcount")

const (
	// stability is the current state of the processor.
	stability = component.StabilityLevelAlpha
)

// NewFactory creates a new factory for the processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		componentType,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, stability),
	)
}

// createLogsProcessor creates a log processor.
func createLogsProcessor(_ context.Context, params processor.CreateSettings, cfg component.Config, consumer consumer.Logs) (processor.Logs, error) {
	processorCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type: %+v", cfg)
	}

	if processorCfg.isOTTL() {
		return createOTTLLogsProcessor(processorCfg, params, consumer)
	}

	return createExprLogsProcessor(processorCfg, params, consumer)
}

func createExprLogsProcessor(cfg *Config, params processor.CreateSettings, consumer consumer.Logs) (processor.Logs, error) {
	match, err := expr.CreateBoolExpression(cfg.exprMatchExpression())
	if err != nil {
		return nil, fmt.Errorf("invalid match expression: %w", err)
	}

	attrs, err := expr.CreateExpressionMap(cfg.Attributes)
	if err != nil {
		return nil, fmt.Errorf("invalid attribute expression: %w", err)
	}

	return newExprProcessor(cfg, consumer, match, attrs, params.Logger), nil
}

func createOTTLLogsProcessor(cfg *Config, params processor.CreateSettings, consumer consumer.Logs) (processor.Logs, error) {
	match, err := expr.NewOTTLLogRecordCondition(cfg.ottlMatchExpression(), params.TelemetrySettings)
	if err != nil {
		return nil, fmt.Errorf("invalid match expression: %w", err)
	}

	attrs, err := expr.MakeOTTLAttributeMap[ottllog.TransformContext](cfg.OTTLAttributes, params.TelemetrySettings, expr.NewOTTLLogRecordExpression)
	if err != nil {
		return nil, fmt.Errorf("invalid attribute expression: %w", err)
	}

	return newOTTLProcessor(cfg, consumer, match, attrs, params.Logger), nil
}
