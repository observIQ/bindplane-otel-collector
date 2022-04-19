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

package googlecloudexporter

import (
	"context"
	"testing"

	"github.com/mitchellh/mapstructure"
	gcp "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
)

func TestCreateMetricExporterSuccess(t *testing.T) {
	mockExporter := &MockExporter{}

	gcpFactory = component.NewExporterFactory(
		typeStr,
		gcpFactory.CreateDefaultConfig,
		component.WithMetricsExporter(func(_ context.Context, _ component.ExporterCreateSettings, _ config.Exporter) (component.MetricsExporter, error) {
			return mockExporter, nil
		}),
	)
	defer func() {
		gcpFactory = gcp.NewFactory()
	}()

	factory := NewFactory()
	cfg := createDefaultConfig()
	ctx := context.Background()
	set := componenttest.NewNopExporterCreateSettings()

	testExporter, err := factory.CreateMetricsExporter(ctx, set, cfg)
	require.NoError(t, err)

	googleExporter, ok := testExporter.(*exporter)
	require.True(t, ok)
	require.Equal(t, googleExporter.metricsExporter, mockExporter)
}

func TestCreateLogsExporterSuccess(t *testing.T) {
	mockExporter := &MockExporter{}

	gcpFactory = component.NewExporterFactory(
		typeStr,
		gcpFactory.CreateDefaultConfig,
		component.WithLogsExporter(func(_ context.Context, _ component.ExporterCreateSettings, _ config.Exporter) (component.LogsExporter, error) {
			return mockExporter, nil
		}),
	)
	defer func() {
		gcpFactory = gcp.NewFactory()
	}()

	factory := NewFactory()
	cfg := createDefaultConfig()
	ctx := context.Background()
	set := componenttest.NewNopExporterCreateSettings()

	testExporter, err := factory.CreateLogsExporter(ctx, set, cfg)
	require.NoError(t, err)

	googleExporter, ok := testExporter.(*exporter)
	require.True(t, ok)
	require.Equal(t, googleExporter.logsExporter, mockExporter)
}

func TestCreateTracesExporterSuccess(t *testing.T) {
	mockExporter := &MockExporter{}

	gcpFactory = component.NewExporterFactory(
		typeStr,
		gcpFactory.CreateDefaultConfig,
		component.WithTracesExporter(func(_ context.Context, _ component.ExporterCreateSettings, _ config.Exporter) (component.TracesExporter, error) {
			return mockExporter, nil
		}),
	)
	defer func() {
		gcpFactory = gcp.NewFactory()
	}()

	factory := NewFactory()
	cfg := createDefaultConfig()
	ctx := context.Background()
	set := componenttest.NewNopExporterCreateSettings()

	testExporter, err := factory.CreateTracesExporter(ctx, set, cfg)
	require.NoError(t, err)

	googleExporter, ok := testExporter.(*exporter)
	require.True(t, ok)
	require.Equal(t, googleExporter.tracesExporter, mockExporter)
}

func TestCreateExporterFailure(t *testing.T) {
	gcpFactory = component.NewExporterFactory(
		typeStr,
		gcpFactory.CreateDefaultConfig,
	)
	defer func() {
		gcpFactory = gcp.NewFactory()
	}()

	factory := NewFactory()
	cfg := createDefaultConfig()
	ctx := context.Background()
	set := componenttest.NewNopExporterCreateSettings()

	_, err := factory.CreateMetricsExporter(ctx, set, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create metrics exporter")

	_, err = factory.CreateLogsExporter(ctx, set, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create logs exporter")

	_, err = factory.CreateTracesExporter(ctx, set, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create traces exporter")
}

func TestCreateProcessorFailure(t *testing.T) {
	mockExporter := &MockExporter{}
	gcpFactory = component.NewExporterFactory(
		typeStr,
		gcpFactory.CreateDefaultConfig,
		component.WithMetricsExporter(func(_ context.Context, _ component.ExporterCreateSettings, _ config.Exporter) (component.MetricsExporter, error) {
			return mockExporter, nil
		}),
		component.WithLogsExporter(func(_ context.Context, _ component.ExporterCreateSettings, _ config.Exporter) (component.LogsExporter, error) {
			return mockExporter, nil
		}),
		component.WithTracesExporter(func(_ context.Context, _ component.ExporterCreateSettings, _ config.Exporter) (component.TracesExporter, error) {
			return mockExporter, nil
		}),
	)
	defer func() {
		gcpFactory = gcp.NewFactory()
	}()

	factory := NewFactory()
	cfg := createDefaultConfig()

	googleCfg, ok := cfg.(*Config)
	require.True(t, ok)

	invalidParams := map[string]interface{}{
		"attributes": []map[string]interface{}{
			{
				"key":    "invalid",
				"value":  "invalid",
				"action": "invalid",
			},
		},
	}
	err := mapstructure.Decode(&invalidParams, googleCfg.AttributerConfig)
	require.NoError(t, err)

	ctx := context.Background()
	set := componenttest.NewNopExporterCreateSettings()

	_, err = factory.CreateMetricsExporter(ctx, set, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create metrics processor")

	_, err = factory.CreateLogsExporter(ctx, set, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create logs processor")

	_, err = factory.CreateTracesExporter(ctx, set, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create traces processor")
}
