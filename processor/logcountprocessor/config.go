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

// Package logcountprocessor provides a processor that counts logs as metrics.
package logcountprocessor

import (
	"fmt"
	"time"

	"github.com/antonmedv/expr"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

const (
	// defaultMetricName is the default metric name.
	defaultMetricName = "log.count"

	// defaultMetricUnit is the default metric unit.
	defaultMetricUnit = "{logs}"

	// defaultInterval is the default metric interval.
	defaultInterval = time.Minute

	// defaultMatch is the default match expression.
	defaultMatch = "true"
)

// Config is the config of the processor.
type Config struct {
	config.ProcessorSettings `mapstructure:",squash"`
	Route                    string            `mapstructure:"route"`
	MetricName               string            `mapstructure:"metric_name"`
	MetricUnit               string            `mapstructure:"metric_unit"`
	Interval                 time.Duration     `mapstructure:"interval"`
	Match                    string            `mapstructure:"match"`
	Attributes               map[string]string `mapstructure:"attributes"`
}

// createMatchExpr returns the match expression defined by the config.
func (c *Config) createMatchExpr() (*Expression, error) {
	matchExpr, err := NewExpression(c.Match, expr.AsBool(), expr.AllowUndefinedVariables())
	if err != nil {
		return nil, fmt.Errorf("failed to create match expression: %w", err)
	}

	return matchExpr, nil
}

// createAttrExprs returns the attribute expressions defined by the config.
func (c *Config) createAttrExprs() (map[string]*Expression, error) {
	attrExprs := make(map[string]*Expression, len(c.Attributes))
	for key, attr := range c.Attributes {
		expr, err := NewExpression(attr, expr.AllowUndefinedVariables())
		if err != nil {
			return nil, fmt.Errorf("failed to create attribute expression for %s: %w", key, err)
		}
		attrExprs[key] = expr
	}

	return attrExprs, nil
}

// createDefaultConfig returns the default config for the processor.
func createDefaultConfig() component.Config {
	return &Config{
		ProcessorSettings: config.NewProcessorSettings(component.NewID(typeStr)),
		MetricName:        defaultMetricName,
		MetricUnit:        defaultMetricUnit,
		Interval:          defaultInterval,
		Match:             defaultMatch,
	}
}
