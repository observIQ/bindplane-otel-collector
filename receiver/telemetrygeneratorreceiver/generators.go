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

package telemetrygeneratorreceiver //import "github.com/observiq/bindplane-agent/receiver/telemetrygeneratorreceiver"

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// generatorType is the type of generator to use, either "logs", "host_metrics", or "windows_events"
type generatorType string

const (
	// generatorTypeLogs is the generator type for logs
	generatorTypeLogs generatorType = "logs"

	// generatorTypeHostMetrics is the generator type for host metrics
	generatorTypeHostMetrics generatorType = "host_metrics"

	// generatorTypeWindowsEvents is the generator type for windows events
	generatorTypeWindowsEvents generatorType = "windows_events"

	// generatorTypeOTLP is the generator type for OTLP
	generatorTypeOTLP generatorType = "otlp"
)

type metricGenerator interface {
	// generateMetrics returns a set of generated metrics
	generateMetrics() pmetric.Metrics
}

type logGenerator interface {
	// generateLogs returns a set of generated logs
	generateLogs() plog.Logs
}

type traceGenerator interface {
	// generateTraces returns a set of generated traces
	generateTraces() ptrace.Traces
}

// newLogsGenerators creates and returns a slice of logGenerator instances based on the provided configuration and logger.
func newLogsGenerators(cfg *Config, logger *zap.Logger) []logGenerator {
	var generators []logGenerator
	for _, gen := range cfg.Generators {
		switch gen.Type {
		case generatorTypeLogs:
			generators = append(generators, newLogsGenerator(gen, logger))
		case generatorTypeWindowsEvents:
			generators = append(generators, newWindowsEventsGenerator(gen, logger))
		case generatorTypeOTLP:
			generators = append(generators, newOtlpGenerator(gen, logger))

		}
	}
	return generators
}

// newMetricsGenerators creates a slice of metricGenerator based on the provided configuration and logger.
func newMetricsGenerators(cfg *Config, logger *zap.Logger) []metricGenerator {
	var generators []metricGenerator
	for _, gen := range cfg.Generators {
		switch gen.Type {
		case generatorTypeHostMetrics:
			generators = append(generators, newHostMetricsGenerator(gen, logger))
		case generatorTypeOTLP:
			generators = append(generators, newOtlpGenerator(gen, logger))
		}

	}
	return generators
}

func newTraceGenerators(cfg *Config, logger *zap.Logger) []traceGenerator {
	var generators []traceGenerator
	for _, gen := range cfg.Generators {
		switch gen.Type {
		case generatorTypeOTLP:
			generators = append(generators, newOtlpGenerator(gen, logger))
		}
	}
	return generators
}
