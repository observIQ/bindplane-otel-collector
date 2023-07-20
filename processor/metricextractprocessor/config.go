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

// Package metricextractprocessor provides a processor that extracts metrics from logs.
package metricextractprocessor

import (
	"errors"
	"fmt"

	"github.com/observiq/bindplane-agent/expr"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

const (
	// defaultMetricName is the default metric name.
	defaultMetricName = "extracted.metric"

	// defaultMetricUnit is the default metric unit.
	defaultMetricUnit = "{units}"

	// defaultOTTLMatch is the default OTTL match expression.
	defaultOTTLMatch = "true"

	// defaultExprMatch is the default expr match expression.
	defaultExprMatch = "true"

	// defaultMetricType is the default metric type.
	defaultMetricType = gaugeDoubleType

	// gaugeDoubleType is the gauge double metric type.
	gaugeDoubleType = "gauge_double"

	// gaugeIntType is the gauge int metric type.
	gaugeIntType = "gauge_int"

	// counterDoubleType is the counter float metric type.
	counterDoubleType = "counter_double"

	// counterIntType is the counter int metric type.
	counterIntType = "counter_int"
)

var (
	// errExprExtractMissing is the error message for a missing expr extract expression.
	errExprExtractMissing = errors.New("extract expression is required")

	// errExtractMissing is the error message for a missing ottl extract expression.
	errOTTLExtractMissing = errors.New("ottl_extract expression is required")

	// errMetricTypeInvalid is the error message for an invalid metric type.
	errMetricTypeInvalid = errors.New("invalid metric type")
)

// Config is the config of the processor.
type Config struct {
	Route          string            `mapstructure:"route"`
	Match          *string           `mapstructure:"match"`
	OTTLMatch      *string           `mapstructure:"ottl_match"`
	Extract        string            `mapstructure:"extract"`
	OTTLExtract    string            `mapstructure:"ottl_extract"`
	MetricName     string            `mapstructure:"metric_name"`
	MetricUnit     string            `mapstructure:"metric_unit"`
	MetricType     string            `mapstructure:"metric_type"`
	Attributes     map[string]string `mapstructure:"attributes"`
	OTTLAttributes map[string]string `mapstructure:"ottl_attributes"`
}

// Validate validates the config.
func (c Config) Validate() error {
	usesExprFields := c.Extract != "" || c.Match != nil || c.Attributes != nil
	usesOTTLFields := c.OTTLExtract != "" || c.OTTLMatch != nil || c.OTTLAttributes != nil

	if usesExprFields && usesOTTLFields {
		return errors.New("cannot use ottl fields (ottl_match, ottl_extract, ottl_attributes) and expr fields (match, extract, attributes)")
	}

	switch c.MetricType {
	case gaugeDoubleType, gaugeIntType, counterDoubleType, counterIntType: // OK
	default:
		return errMetricTypeInvalid
	}

	if c.isOTTL() {
		return c.validateOTTL()
	}

	return c.validateExpr()
}

func (c Config) validateExpr() error {
	if c.Extract == "" {
		return errExprExtractMissing
	}

	_, err := expr.CreateBoolExpression(c.exprMatchExpression())
	if err != nil {
		return fmt.Errorf("invalid match: %w", err)
	}

	_, err = expr.CreateExpression(c.Extract)
	if err != nil {
		return fmt.Errorf("invalid extract: %w", err)
	}

	_, err = expr.CreateExpressionMap(c.Attributes)
	if err != nil {
		return fmt.Errorf("invalid attributes: %w", err)
	}
	return nil
}

func (c Config) validateOTTL() error {
	if c.OTTLExtract == "" {
		return errOTTLExtractMissing
	}

	_, err := expr.NewOTTLLogRecordCondition(c.ottlMatchExpression(), component.TelemetrySettings{
		Logger: zap.NewNop(),
	})
	if err != nil {
		return fmt.Errorf("invalid ottl_match: %w", err)
	}

	_, err = expr.NewOTTLLogRecordExpression(c.OTTLExtract, component.TelemetrySettings{
		Logger: zap.NewNop(),
	})
	if err != nil {
		return fmt.Errorf("invalid ottl_extract: %w", err)
	}

	_, err = expr.MakeOTTLAttributeMap(c.OTTLAttributes, component.TelemetrySettings{
		Logger: zap.NewNop(),
	}, expr.NewOTTLLogRecordExpression)
	if err != nil {
		return fmt.Errorf("invalid ottl_attributes: %w", err)
	}

	return nil
}

func (c Config) exprMatchExpression() string {
	if c.Match != nil {
		return *c.Match
	}

	return defaultExprMatch
}

func (c Config) ottlMatchExpression() string {
	if c.OTTLMatch != nil {
		return *c.OTTLMatch
	}
	return defaultOTTLMatch
}

func (c Config) isOTTL() bool {
	// Use OTTL if none of the expr fields are set.
	return c.Match == nil && c.Attributes == nil && c.Extract == ""
}

// createDefaultConfig returns the default config for the processor.
func createDefaultConfig() component.Config {
	return &Config{
		MetricName: defaultMetricName,
		MetricUnit: defaultMetricUnit,
		MetricType: defaultMetricType,
	}
}
