package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type googleCloudStorageExporter struct {
	cfg *Config
	storageClient storageClient
	logger     *zap.Logger
	marshaler  marshaler
}

func newExporter(cfg *Config, params exporter.Settings) (*googleCloudStorageExporter, error) {
	storageClient, err := newGoogleCloudStorageClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}
	// should close the client?

	return &googleCloudStorageExporter{
		cfg: cfg,
		storageClient: storageClient,
		logger:     params.Logger,
		marshaler:  newMarshaler(),
	}, nil
}

func (g *googleCloudStorageExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (g *googleCloudStorageExporter) metricsDataPusher(ctx context.Context, md pmetric.Metrics) error {
	buf, err := g.marshaler.MarshalMetrics(md)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	blobName := g.getBlobName("metrics")

	return g.uploadBuffer(ctx, blobName, buf)
}

func (g *googleCloudStorageExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	buf, err := g.marshaler.MarshalLogs(ld)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	blobName := g.getBlobName("logs")

	return g.uploadBuffer(ctx, blobName, buf)
}

func (g *googleCloudStorageExporter) traceDataPusher(ctx context.Context, td ptrace.Traces) error {
	buf, err := g.marshaler.MarshalTraces(td)
	if err != nil {
		return fmt.Errorf("failed to marshal traces: %w", err)
	}

	blobName := g.getBlobName("traces")

	return g.uploadBuffer(ctx, blobName, buf)
}

// func (g *googleCloudStorageExporter) getBlobName(telemetryType string) string {
// 	now := time.Now()
// 	year, month, day := now.Date()
// 	hour, minute, _ := now.Clock()

// 	blobNameBuilder := strings.Builder{}

// 	if g.cfg.FolderName != "" {
// 		blobNameBuilder.WriteString(g.cfg.FolderName)
// 		blobNameBuilder.WriteString("/")
// 	}

// 	blobNameBuilder.WriteString(fmt.Sprintf("%s-%d-%d-%d-%d-%d", telemetryType, year, month, day, hour, minute))

// 	if g.cfg.ObjectPrefix != "" {
// 		blobNameBuilder.WriteString(fmt.Sprintf("-%s", g.cfg.ObjectPrefix))
// 	}

// 	randomID := randomInRange(100000000, 999999999)

// 	blobNameBuilder.WriteString(fmt.Sprintf("-%d", randomID))

// 	return blobNameBuilder.String()
// }

// func (g *googleCloudStorageExporter) uploadBuffer(ctx context.Context, blobName string, buf []byte) error {
// 	bucket, err := g.storageClient.GetBucket(ctx, g.cfg.BucketName)
// 	if err != nil {
// 		return fmt.Errorf("failed to create bucket: %w", err)
// 	}

// 	object := bucket.Object(blobName)
// 	if err != nil {
// 		return fmt.Errorf("failed to create object: %w", err)
// 	}

// 	writer := object.NewWriter(ctx)
// 	writer.Write(buf)
// 	if err != nil {
// 		return fmt.Errorf("failed to write to object: %w", err)
// 	}

// 	err = writer.Close()
// 	if err != nil {
// 		return fmt.Errorf("failed to close writer: %w", err)
// 	}

// 	return nil
// }

// func randomInRange(low, hi int) int {
// 	return low + rand.Intn(hi-low)
// }
