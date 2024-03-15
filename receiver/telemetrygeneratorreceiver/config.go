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

// Package telemetrygeneratorreceiver generates telemetry for testing purposes
package telemetrygeneratorreceiver //import "github.com/observiq/bindplane-agent/receiver/telemetrygeneratorreceiver"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Config is the configuration for the telemetry generator receiver
type Config struct {
	PayloadsPerSecond int               `mapstructure:"payloads_per_second"`
	Generators        []GeneratorConfig `mapstructure:"generators"`
}

// GeneratorConfig is the configuration for a single generator
type GeneratorConfig struct {
	// Type of generator to use, either "logs", "host_metrics", or "windows_events"
	Type generatorType `mapstructure:"type"`

	// ResourceAttributes are additional key-value pairs to add to the resource attributes of telemetry.
	ResourceAttributes map[string]any `mapstructure:"resource_attributes"`

	// Attributes are Additional key-value pairs to add to the telemetry attributes
	Attributes map[string]any `mapstructure:"attributes"`

	// AdditionalConfig are any additional config that a generator might need.
	AdditionalConfig map[string]any `mapstructure:"additional_config"`
}

// Validate validates the config
func (c *Config) Validate() error {

	if c.PayloadsPerSecond < 1 {
		return errors.New("payloads_per_second must be at least 1")
	}

	for _, generator := range c.Generators {
		if err := generator.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the generator config
func (g *GeneratorConfig) Validate() error {

	switch g.Type {
	case generatorTypeLogs:
		return validateLogGeneratorConfig(g)
	case generatorTypeMetrics:
		return validateMetricsGeneratorConfig(g)
	case generatorTypeHostMetrics:
		return validateHostMetricsGeneratorConfig(g)
	case generatorTypeWindowsEvents:
		return validateWindowsEventsGeneratorConfig(g)
	case generatorTypeOTLP:
		return validateOTLPGenerator(g)
	default:
		return fmt.Errorf("invalid generator type: %s", g.Type)
	}
}

func validateLogGeneratorConfig(g *GeneratorConfig) error {

	err := pcommon.NewMap().FromRaw(g.Attributes)
	if err != nil {
		return fmt.Errorf("error in attributes config: %s", err)
	}

	err = pcommon.NewMap().FromRaw(g.ResourceAttributes)
	if err != nil {
		return fmt.Errorf("error in resource_attributes config: %s", err)
	}

	// severity and body validation
	if body, ok := g.AdditionalConfig["body"]; ok {
		// check if body is a valid string, if not, return an error
		_, ok := body.(string)
		if !ok {
			return errors.New("body must be a string")
		}
	}

	// if severity is set, it must be a valid severity
	if severity, ok := g.AdditionalConfig["severity"]; ok {
		severityVal, ok := severity.(int)
		if !ok {
			return errors.New("severity must be an integer")
		}
		sn := plog.SeverityNumber(severityVal)
		if sn.String() == "" {
			return fmt.Errorf("invalid severity: %d", severityVal)
		}
	}
	return nil
}

func validateOTLPGenerator(cfg *GeneratorConfig) error {

	telemetryType, ok := cfg.AdditionalConfig["telemetry_type"]
	if !ok {
		return errors.New("telemetry_type must be set")
	}

	// validate the telemetry type
	telemetryTypeStr, ok := telemetryType.(string)
	if !ok {
		return fmt.Errorf("invalid telemetry type: %v", telemetryType)
	}
	dataType := component.DataType(telemetryTypeStr)
	switch dataType {
	case component.DataTypeLogs, component.DataTypeMetrics, component.DataTypeTraces:
	default:
		return fmt.Errorf("invalid telemetry type: %s", telemetryType)
	}

	// validate the otlp json
	otlpJSON, ok := cfg.AdditionalConfig["otlp_json"]
	if !ok {
		return errors.New("otlp_json must be set")
	}

	otlpJSONStr, ok := otlpJSON.(string)
	if !ok {
		return fmt.Errorf("otlp_json must be a string, got: %v", otlpJSON)
	}

	jsonBytes := []byte(otlpJSONStr)

	switch dataType {
	case component.DataTypeLogs:
		marshaler := plog.JSONUnmarshaler{}
		logs, err := marshaler.UnmarshalLogs(jsonBytes)
		if err != nil {
			return fmt.Errorf("error unmarshalling logs from otlp_json: %w", err)
		}
		if logs.LogRecordCount() == 0 {
			return errors.New("no log records found in otlp_json")
		}
	case component.DataTypeMetrics:
		marshaler := pmetric.JSONUnmarshaler{}
		metrics, err := marshaler.UnmarshalMetrics(jsonBytes)
		if err != nil {
			return fmt.Errorf("error unmarshalling metrics from otlp_json: %w", err)
		}
		if metrics.DataPointCount() == 0 {
			return errors.New("no metric data points found in otlp_json")
		}
	case component.DataTypeTraces:
		marshaler := ptrace.JSONUnmarshaler{}
		traces, err := marshaler.UnmarshalTraces(jsonBytes)
		if err != nil {
			return fmt.Errorf("error unmarshalling traces from otlp_json: %w", err)
		}
		if traces.SpanCount() == 0 {
			return errors.New("no trace spans found in otlp_json")
		}

	}
	return nil
}

func validateMetricsGeneratorConfig(g *GeneratorConfig) error {
	err := pcommon.NewMap().FromRaw(g.Attributes)
	if err != nil {
		return fmt.Errorf("error in attributes config: %s", err)
	}

	err = pcommon.NewMap().FromRaw(g.ResourceAttributes)
	if err != nil {
		return fmt.Errorf("error in resource_attributes config: %s", err)
	}

	// validate individual metrics
	metrics, ok := g.AdditionalConfig["metrics"]
	if !ok {
		return errors.New("metrics must be set")
	}
	// check that the metricsArray is a valid array of maps[string]any
	// Because of the way the config is unmarshaled, we have to use the `[]any` type
	// and then cast to the correct type
	metricsArray, ok := metrics.([]any)
	if !ok {
		return errors.New("metrics must be an array of maps")
	}
	for _, m := range metricsArray {
		metric, ok := m.(map[string]any)
		if !ok {
			return errors.New("each metric must be a map")
		}
		// check that the metric has a name
		name, ok := metric["name"]
		if !ok {
			return errors.New("each metric must have a name")
		}
		// check that the metric has a type
		metricType, ok := metric["type"]
		if !ok {
			return fmt.Errorf("metric %s missing type", name)
		}
		// check that the metric type is valid
		metricTypeStr, ok := metricType.(string)
		if !ok {
			return fmt.Errorf("metric %s has invalid metric type: %v", name, metricType)
		}
		switch metricTypeStr {
		case "Gauge", "Sum":
		default:
			return fmt.Errorf("metric %s has invalid metric type: %s", name, metricTypeStr)
		}
		// check that the metric has a value_min
		valMin, ok := metric["value_min"]
		if !ok {
			return fmt.Errorf("metric %s missing value_min", name)
		}
		// check that the value_min is a valid int
		if _, ok = valMin.(int); !ok {
			return fmt.Errorf("metric %s has invalid value_min: %v", name, valMin)
		}
		// check that the metric has a value_max
		valMax, ok := metric["value_max"]
		if !ok {
			return fmt.Errorf("metric %s missing value_max", name)
		}
		// check that the value_max is a valid int
		if _, ok = valMax.(int); !ok {
			return fmt.Errorf("metric %s has invalid value_max: %v", name, valMax)
		}
		// check that the metric has a unit
		unit, ok := metric["unit"]
		if !ok {
			return fmt.Errorf("metric %s missing unit", name)
		}
		// check that the unit is a valid string
		unitStr, ok := unit.(string)
		if !ok {
			return fmt.Errorf("metric %s has invalid unit: %v", name, unit)
		}
		switch unitStr {
		case "By", "by", "1", "s", "{thread}", "{errors}", "{packets}", "{entries}", "{connections}", "{faults}", "{operations}", "{processes}":
		default:
			return fmt.Errorf("metric %s has invalid unit: %s", name, unitStr)
		}

		// attributes are optional
		if attr, ok := metric["attributes"]; ok {
			attributes := attr.(map[string]any)
			err = pcommon.NewMap().FromRaw(attributes)
			if err != nil {
				return fmt.Errorf("error in attributes config for metric %s: %w", name, err)
			}
		}
	}

	return nil
}

func validateHostMetricsGeneratorConfig(g *GeneratorConfig) error {
	err := pcommon.NewMap().FromRaw(g.ResourceAttributes)
	if err != nil {
		return fmt.Errorf("error in resource_attributes config: %s", err)
	}
	return nil
}

func validateWindowsEventsGeneratorConfig(_ *GeneratorConfig) error {
	return nil
}
