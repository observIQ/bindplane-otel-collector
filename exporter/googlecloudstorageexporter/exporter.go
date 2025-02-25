package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter"

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

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
	storageClient, err := newGoogleCloudStorageClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}
	// should close the client?

	return &googleCloudStorageExporter{
		cfg: cfg,
		storageClient: storageClient,
		logger:     params.Logger,
		marshaler:  newMarshaler(cfg.Compression),
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

	objectName := g.getObjectName("metrics")

	return g.storageClient.UploadObject(ctx, g.cfg.BucketName, objectName, buf)
}

func (g *googleCloudStorageExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	buf, err := g.marshaler.MarshalLogs(ld)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	objectName := g.getObjectName("logs")

	return g.storageClient.UploadObject(ctx, g.cfg.BucketName, objectName, buf)
}

func (g *googleCloudStorageExporter) traceDataPusher(ctx context.Context, td ptrace.Traces) error {
	buf, err := g.marshaler.MarshalTraces(td)
	if err != nil {
		return fmt.Errorf("failed to marshal traces: %w", err)
	}

	objectName := g.getObjectName("traces")

	return g.storageClient.UploadObject(ctx, g.cfg.BucketName, objectName, buf)
}

func (g *googleCloudStorageExporter) getObjectName(telemetryType string) string {
	now := time.Now()
	year, month, day := now.Date()
	hour, minute, _ := now.Clock()

	objectNameBuilder := strings.Builder{}

	if g.cfg.FolderName != "" {
		objectNameBuilder.WriteString(g.cfg.FolderName)
		objectNameBuilder.WriteString("/")
	}

	objectNameBuilder.WriteString(fmt.Sprintf("%s-%d-%d-%d-%d-%d", telemetryType, year, month, day, hour, minute))

	if g.cfg.ObjectPrefix != "" {
		objectNameBuilder.WriteString(fmt.Sprintf("-%s", g.cfg.ObjectPrefix))
	}

	randomID := randomInRange(100000000, 999999999)

	objectNameBuilder.WriteString(fmt.Sprintf("-%d", randomID))

	return objectNameBuilder.String()
}

func randomInRange(low, hi int) int {
	return low + rand.Intn(hi-low)
}
