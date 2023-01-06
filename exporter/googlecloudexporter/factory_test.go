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

	gcp "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

func TestCreateMetricExporterSuccess(t *testing.T) {
	mockExporter := &MockExporter{}

	gcpFactory = exporter.NewFactory(
		typeStr,
		gcpFactory.CreateDefaultConfig,
		exporter.WithMetrics(func(_ context.Context, _ component.ExporterCreateSettings, _ component.Config) (exporter.Metrics, error) {
			return mockExporter, nil
		}, stability),
	)
	defer func() {
		gcpFactory = gcp.NewFactory()
	}()

	factory := NewFactory()
	cfg := createDefaultConfig()
	ctx := context.Background()
	set := exportertest.NewNopCreateSettings()

	testExporter, err := factory.CreateMetricsExporter(ctx, set, cfg)
	require.NoError(t, err)

	googleExporter, ok := testExporter.(*googlecloudExporter)
	require.True(t, ok)
	require.Equal(t, googleExporter.metricsExporter, mockExporter)
}

func TestCreateLogsExporterSuccess(t *testing.T) {
	mockExporter := &MockExporter{}

	gcpFactory = exporter.NewFactory(
		typeStr,
		gcpFactory.CreateDefaultConfig,
		exporter.WithLogs(func(_ context.Context, _ component.ExporterCreateSettings, _ component.Config) (exporter.Logs, error) {
			return mockExporter, nil
		}, stability),
	)
	defer func() {
		gcpFactory = gcp.NewFactory()
	}()

	factory := NewFactory()
	cfg := createDefaultConfig()
	ctx := context.Background()
	set := exportertest.NewNopCreateSettings()

	testExporter, err := factory.CreateLogsExporter(ctx, set, cfg)
	require.NoError(t, err)

	googleExporter, ok := testExporter.(*googlecloudExporter)
	require.True(t, ok)
	require.Equal(t, googleExporter.logsExporter, mockExporter)
}

func TestCreateTracesExporterSuccess(t *testing.T) {
	mockExporter := &MockExporter{}

	gcpFactory = exporter.NewFactory(
		typeStr,
		gcpFactory.CreateDefaultConfig,
		exporter.WithTraces(func(_ context.Context, _ component.ExporterCreateSettings, _ component.Config) (exporter.Traces, error) {
			return mockExporter, nil
		}, component.StabilityLevelUndefined),
	)
	defer func() {
		gcpFactory = gcp.NewFactory()
	}()

	factory := NewFactory()
	cfg := createDefaultConfig()
	ctx := context.Background()
	set := exportertest.NewNopCreateSettings()

	testExporter, err := factory.CreateTracesExporter(ctx, set, cfg)
	require.NoError(t, err)

	googleExporter, ok := testExporter.(*googlecloudExporter)
	require.True(t, ok)
	require.Equal(t, googleExporter.tracesExporter, mockExporter)
}

func TestCreateExporterFailure(t *testing.T) {
	gcpFactory = exporter.NewFactory(
		typeStr,
		gcpFactory.CreateDefaultConfig,
	)
	defer func() {
		gcpFactory = gcp.NewFactory()
	}()

	factory := NewFactory()
	cfg := createDefaultConfig()
	ctx := context.Background()
	set := exportertest.NewNopCreateSettings()

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
