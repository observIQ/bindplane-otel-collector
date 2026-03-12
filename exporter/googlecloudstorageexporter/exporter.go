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

package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-contrib/exporter/googlecloudstorageexporter"

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/observiq/bindplane-otel-contrib/exporter/googlecloudstorageexporter/internal/metadata"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// googleCloudStorageExporter exports OTLP data as Google Cloud Storage objects
type googleCloudStorageExporter struct {
	cfg           *Config
	storageClient storageClient
	logger        *zap.Logger
	marshaler     marshaler
	telemetry     *metadata.TelemetryBuilder
}

// newExporter creates a new Google Cloud Storage exporter
func newExporter(cfg *Config, params exporter.Settings) (*googleCloudStorageExporter, error) {
	storageClient, err := newGoogleCloudStorageClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}

	return &googleCloudStorageExporter{
		cfg:           cfg,
		storageClient: storageClient,
		logger:        params.Logger,
		marshaler:     newMarshaler(cfg.Compression),
	}, nil
}

// Capabilities lists the exporter's capabilities
func (g *googleCloudStorageExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// metricsDataPusher pushes metrics data to Google Cloud Storage
func (g *googleCloudStorageExporter) metricsDataPusher(ctx context.Context, md pmetric.Metrics) error {
	buf, err := g.marshaler.MarshalMetrics(md)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}

	objectName := g.getObjectName("metrics")

	return g.uploadAndRecord(ctx, objectName, buf)
}

// logsDataPusher pushes logs data to Google Cloud Storage
func (g *googleCloudStorageExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	buf, err := g.marshaler.MarshalLogs(ld)
	if err != nil {
		return fmt.Errorf("marshal logs: %w", err)
	}

	objectName := g.getObjectName("logs")

	return g.uploadAndRecord(ctx, objectName, buf)
}

// tracesDataPusher pushes trace data to Google Cloud Storage
func (g *googleCloudStorageExporter) tracesDataPusher(ctx context.Context, td ptrace.Traces) error {
	buf, err := g.marshaler.MarshalTraces(td)
	if err != nil {
		return fmt.Errorf("marshal traces: %w", err)
	}

	objectName := g.getObjectName("traces")

	return g.uploadAndRecord(ctx, objectName, buf)
}

// uploadAndRecord uploads the payload to GCS and records telemetry metrics.
func (g *googleCloudStorageExporter) uploadAndRecord(ctx context.Context, objectName string, buf []byte) error {
	bucketAttr := attribute.String("bucket", g.cfg.BucketName)
	payloadSize := int64(len(buf))

	g.telemetry.ExporterPayloadSize.Record(ctx, payloadSize,
		metric.WithAttributes(
			attribute.String("encoding", g.marshaler.Format()),
			bucketAttr,
		),
	)

	g.telemetry.ExporterUploadInflight.Add(ctx, 1,
		metric.WithAttributes(bucketAttr),
	)

	start := time.Now()
	uploadErr := g.storageClient.UploadObject(ctx, objectName, buf)

	g.telemetry.ExporterUploadInflight.Add(ctx, -1,
		metric.WithAttributes(bucketAttr),
	)

	g.telemetry.ExporterRequestDuration.Record(ctx, time.Since(start).Milliseconds(),
		metric.WithAttributes(
			attribute.String("error", classifyError(uploadErr)),
			bucketAttr,
			attribute.String("location", g.cfg.BucketLocation),
		),
	)

	if uploadErr == nil {
		g.telemetry.ExporterUploadBytesTotal.Add(ctx, payloadSize,
			metric.WithAttributes(bucketAttr),
		)
	} else if errors.Is(uploadErr, context.DeadlineExceeded) {
		g.telemetry.ExporterTimeoutTotal.Add(ctx, 1,
			metric.WithAttributes(bucketAttr),
		)
	}

	return uploadErr
}

// classifyError returns a string classification for the given error.
func classifyError(err error) string {
	if err == nil {
		return "none"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "unknown"
}

// getObjectName formats the object name based on the configuration and current time stamp
func (g *googleCloudStorageExporter) getObjectName(telemetryType string) string {
	now := time.Now().UTC()
	year, month, day := now.Date()
	hour, minute, _ := now.Clock()

	objectNameBuilder := strings.Builder{}

	// Add folder name if specified
	if g.cfg.FolderName != "" {
		objectNameBuilder.WriteString(fmt.Sprintf("%s/", g.cfg.FolderName))
	}

	// Add hierarchical time-based folders
	objectNameBuilder.WriteString(fmt.Sprintf("year=%d/month=%02d/day=%02d/hour=%02d", year, month, day, hour))

	// Add minute folder if using minute partitioning
	if g.cfg.Partition == minutePartition {
		objectNameBuilder.WriteString(fmt.Sprintf("/minute=%02d", minute))
	}

	objectNameBuilder.WriteString("/")

	// Add object prefix if specified
	if g.cfg.ObjectPrefix != "" {
		objectNameBuilder.WriteString(g.cfg.ObjectPrefix)
	}

	// Generate a random ID for the name
	randomID := randomInRange(100000000, 999999999)

	// Write base file name with telemetry type and random ID
	objectNameBuilder.WriteString(fmt.Sprintf("%s_%d.%s", telemetryType, randomID, g.marshaler.Format()))

	return objectNameBuilder.String()
}

// #nosec G404 -- randomly generated number is not used for security purposes. It's ok if it's weak
func randomInRange(low, hi int) int {
	return low + rand.Intn(hi-low)
}
