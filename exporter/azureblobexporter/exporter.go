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

package azureblobexporter // import "github.com/observiq/bindplane-agent/exporter/azureblobexporter"

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

// azureBlobExporter exports OTLP data as Azure blobs
type azureBlobExporter struct {
	cfg        *Config
	blobClient blobClient
	logger     *zap.Logger
	marshaler  marshaler
}

// newExporter creates a new Azure Blob exporter
func newExporter(cfg *Config, params exporter.CreateSettings) (*azureBlobExporter, error) {
	blobClient, err := newAzureBlobClient(cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob client: %w", err)
	}

	return &azureBlobExporter{
		cfg:        cfg,
		blobClient: blobClient,
		logger:     params.Logger,
		marshaler:  newMarshaler(cfg.Compression),
	}, nil
}

// Capabilities lists the exporter's capabilities
func (a *azureBlobExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// metricsDataPusher pushes metrics data to Azure Blob
func (a *azureBlobExporter) metricsDataPusher(ctx context.Context, md pmetric.Metrics) error {
	buf, err := a.marshaler.MarshalMetrics(md)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	blobName := a.getBlobName("metrics")

	return a.uploadBuffer(ctx, blobName, buf)
}

// logsDataPusher pushes logs data to Azure Blob
func (a *azureBlobExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	buf, err := a.marshaler.MarshalLogs(ld)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	blobName := a.getBlobName("logs")

	return a.uploadBuffer(ctx, blobName, buf)
}

// tracesDataPusher pushes trace data to Azure Blob
func (a *azureBlobExporter) tracesDataPusher(ctx context.Context, td ptrace.Traces) error {
	buf, err := a.marshaler.MarshalTraces(td)
	if err != nil {
		return fmt.Errorf("failed to marshal traces: %w", err)
	}

	blobName := a.getBlobName("traces")

	return a.uploadBuffer(ctx, blobName, buf)
}

// getBlobName formats the blob name based on the configuration and current time stamp
func (a *azureBlobExporter) getBlobName(telemetryType string) string {
	now := time.Now().UTC()
	year, month, day := now.Date()
	hour, minute, _ := now.Clock()

	blobNameBuilder := strings.Builder{}

	if a.cfg.RootFolder != "" {
		blobNameBuilder.WriteString(fmt.Sprintf("%s/", a.cfg.RootFolder))
	}

	blobNameBuilder.WriteString(fmt.Sprintf("year=%d/month=%02d/day=%02d/hour=%02d", year, month, day, hour))

	if a.cfg.Partition == minutePartition {
		blobNameBuilder.WriteString(fmt.Sprintf("/minute=%02d", minute))
	}

	blobNameBuilder.WriteString("/")

	if a.cfg.BlobPrefix != "" {
		blobNameBuilder.WriteString(a.cfg.BlobPrefix)
	}

	// Generate a random ID for the name
	randomID := randomInRange(100000000, 999999999)

	// Write base file name
	blobNameBuilder.WriteString(fmt.Sprintf("%s_%d.%s", telemetryType, randomID, a.marshaler.Format()))

	return blobNameBuilder.String()
}

// uploadBuffer uploads the supplied buffer to the configured container with the given blobName
func (a *azureBlobExporter) uploadBuffer(ctx context.Context, blobName string, buf []byte) error {
	if err := a.blobClient.UploadBuffer(ctx, a.cfg.Container, blobName, buf); err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	return nil
}

// #nosec G404 -- randomly generated number is not used for security purposes. It's ok if it's weak
func randomInRange(low, hi int) int {
	return low + rand.Intn(hi-low)
}
