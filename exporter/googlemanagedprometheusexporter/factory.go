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

// Package googlemanagedprometheusexporter provides a wrapper around the official googlemanagedprometheusexporter component that exposes some quality of life improvements in configuration
package googlemanagedprometheusexporter

import (
	"context"
	"fmt"

	gmp "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlemanagedprometheusexporter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.uber.org/zap"
)

// gmpFactory is the factory used to create the underlying google managed prometheus exporter
var gmpFactory = gmp.NewFactory()

const (
	// typeStr is the type of the google managed prometheus exporter
	typeStr = "googlemanagedprometheus"

	// The stability level of the exporter. Matches the current exporter in contrib
	stability = component.StabilityLevelBeta
)

// NewFactory creates a factory for the googlecloud exporter
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithMetrics(createMetricsExporter, stability),
	)
}

// createMetricsExporter creates a metrics exporter based on this config.
func createMetricsExporter(ctx context.Context, set exporter.CreateSettings, cfg component.Config) (exporter.Metrics, error) {
	exporterConfig := cfg.(*Config)
	exporterConfig.setClientOptions()

	if err := exporterConfig.setProject(); err != nil {
		set.Logger.Error("Failed to set project automatically", zap.Error(err))
	}

	gmpExporter, err := gmpFactory.CreateMetricsExporter(ctx, set, exporterConfig.GMPConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	return gmpExporter, nil
}
