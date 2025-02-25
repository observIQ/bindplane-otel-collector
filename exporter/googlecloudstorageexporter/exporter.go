package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter"

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type googleCloudStorageExporter struct {
	storageClient *storage.Client
	logger     *zap.Logger
}

func newExporter(config *Config, params exporter.CreateSettings) (*googleCloudStorageExporter, error) {
	storageClient, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &googleCloudStorageExporter{
		storageClient: storageClient,
		logger:     params.Logger,
	}, nil
}

func (a *googleCloudStorageExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (a *googleCloudStorageExporter) metricsDataPusher(_ context.Context, md pmetric.Metrics) error {
	return nil
}

func (a *googleCloudStorageExporter) logsDataPusher(_ context.Context, ld plog.Logs) error {
	return nil
}

func (a *googleCloudStorageExporter) traceDataPusher(_ context.Context, td ptrace.Traces) error {
	return nil
}