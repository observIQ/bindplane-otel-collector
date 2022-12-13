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

	"github.com/observiq/observiq-otel-collector/internal/expr"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

const (
	// defaultMetricName is the default metric name.
	defaultMetricName = "extracted.metric"

	// defaultMetricUnit is the default metric unit.
	defaultMetricUnit = "{units}"

	// defaultMatch is the default match expression.
	defaultMatch = "true"

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
	// errExtractMissing is the error message for a missing extract expression.
	errExtractMissing = errors.New("extract expression is required")

	// errMetricTypeInvalid is the error message for an invalid metric type.
	errMetricTypeInvalid = errors.New("invalid metric type")
)

// Config is the config of the processor.
type Config struct {
	config.ProcessorSettings `mapstructure:",squash"`
	Route                    string            `mapstructure:"route"`
	Match                    string            `mapstructure:"match"`
	Extract                  string            `mapstructure:"extract"`
	MetricName               string            `mapstructure:"metric_name"`
	MetricUnit               string            `mapstructure:"metric_unit"`
	MetricType               string            `mapstructure:"metric_type"`
	Attributes               map[string]string `mapstructure:"attributes"`
}

// Validate validates the config.
func (c Config) Validate() error {
	if c.Extract == "" {
		return errExtractMissing
	}

	_, err := expr.CreateBoolExpression(c.Match)
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

	switch c.MetricType {
	case gaugeDoubleType, gaugeIntType, counterDoubleType, counterIntType:
		return nil
	default:
		return errMetricTypeInvalid
	}
}

// createDefaultConfig returns the default config for the processor.
func createDefaultConfig() component.ProcessorConfig {
	return &Config{
		ProcessorSettings: config.NewProcessorSettings(component.NewID(typeStr)),
		MetricName:        defaultMetricName,
		MetricUnit:        defaultMetricUnit,
		MetricType:        defaultMetricType,
		Match:             defaultMatch,
	}
}
