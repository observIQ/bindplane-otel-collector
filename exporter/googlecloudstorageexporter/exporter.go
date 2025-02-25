package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter"

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type googleCloudStorageExporter struct {
	cfg *Config
	storageClient *storage.Client
	logger     *zap.Logger
	marshaler  marshaler
}

func newExporter(cfg *Config, params exporter.CreateSettings) (*googleCloudStorageExporter, error) {
	storageClient, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &googleCloudStorageExporter{
		cfg: cfg,
		storageClient: storageClient,
		logger:     params.Logger,
		marshaler:  newMarshaler(),
	}, nil
}

func (a *googleCloudStorageExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (a *googleCloudStorageExporter) metricsDataPusher(_ context.Context, md pmetric.Metrics) error {
	buf, err := a.marshaler.MarshalMetrics(md)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	blobName := a.getBlobName("metrics")

	return a.uploadBuffer(ctx, blobName, buf)
}

func (a *googleCloudStorageExporter) logsDataPusher(_ context.Context, ld plog.Logs) error {
	buf, err := a.marshaler.MarshalLogs(ld)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	blobName := a.getBlobName("logs")

	return a.uploadBuffer(ctx, blobName, buf)
}

func (a *googleCloudStorageExporter) traceDataPusher(_ context.Context, td ptrace.Traces) error {
	buf, err := a.marshaler.MarshalTraces(td)
	if err != nil {
		return fmt.Errorf("failed to marshal traces: %w", err)
	}

	blobName := a.getBlobName("traces")

	return a.uploadBuffer(ctx, blobName, buf)
}

func (a *googleCloudStorageExporter) getBlobName(telemetryType string) string {
	now := time.Now()
	year, month, day := now.Date()
	hour, minute, _ := now.Clock()

	blobNameBuilder := strings.Builder{}

	if a.cfg.FolderName != "" {
		blobNameBuilder.WriteString(a.cfg.FolderName)
		blobNameBuilder.WriteString("/")
	}

	blobNameBuilder.WriteString(fmt.Sprintf("%s-%d-%d-%d-%d-%d", telemetryType, year, month, day, hour, minute))

	if a.cfg.BlobPrefix != "" {
		blobNameBuilder.WriteString(fmt.Sprintf("-%s", a.cfg.BlobPrefix))
	}

	randomID := randomInRange(100000000, 999999999)

	blobNameBuilder.WriteString(fmt.Sprintf("-%d", randomID))

	return blobNameBuilder.String()
}

func (a *googleCloudStorageExporter) uploadBuffer(ctx context.Context, blobName string, buf []byte) error {
	bucket := a.storageClient.Bucket(a.cfg.BucketName)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	object := bucket.Object(blobName)
	if err != nil {
		return fmt.Errorf("failed to create object: %w", err)
	}

	writer := object.NewWriter(ctx)
	writer.Write(buf)
	if err != nil {
		return fmt.Errorf("failed to write to object: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}

func randomInRange(low, hi int) int {
	return low + rand.Intn(hi-low)
}
