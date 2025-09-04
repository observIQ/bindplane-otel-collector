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

package chronicleexporter

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	grpcgzip "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"
)

const grpcScope = "https://www.googleapis.com/auth/malachite-ingestion"

type grpcExporter struct {
	cfg        *Config
	set        component.TelemetrySettings
	exporterID string
	marshaler  *protoMarshaler

	client  api.IngestionServiceV2Client
	conn    *grpc.ClientConn
	metrics *hostMetricsReporter

	telemetry        *metadata.TelemetryBuilder
	metricAttributes attribute.Set
}

func newGRPCExporter(cfg *Config, params exporter.Settings, telemetry *metadata.TelemetryBuilder) (*grpcExporter, error) {
	marshaler, err := newProtoMarshaler(*cfg, params.TelemetrySettings, telemetry, params.Logger)
	if err != nil {
		return nil, fmt.Errorf("create proto marshaler: %w", err)
	}
	return &grpcExporter{
		cfg:        cfg,
		set:        params.TelemetrySettings,
		exporterID: params.ID.String(),
		marshaler:  marshaler,
		telemetry:  telemetry,
		metricAttributes: attribute.NewSet(attribute.KeyValue{
			Key:   "exporter",
			Value: attribute.StringValue(params.ID.String()),
		}),
	}, nil
}

func (exp *grpcExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (exp *grpcExporter) Start(ctx context.Context, _ component.Host) error {
	ts, err := tokenSource(ctx, exp.cfg)
	if err != nil {
		return fmt.Errorf("load Google credentials: %w", err)
	}
	endpoint, dialOpts := grpcClientParams(exp.cfg.Endpoint, ts)
	conn, err := grpc.NewClient(endpoint, dialOpts...)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	exp.conn = conn
	exp.client = api.NewIngestionServiceV2Client(conn)

	if exp.cfg.CollectAgentMetrics {
		f := func(ctx context.Context, request *api.BatchCreateEventsRequest) error {
			_, err := exp.client.BatchCreateEvents(ctx, request, exp.buildOptions()...)
			return err
		}
		metrics, err := newHostMetricsReporter(exp.cfg, exp.set, exp.exporterID, f)
		if err != nil {
			return fmt.Errorf("create metrics reporter: %w", err)
		}
		exp.metrics = metrics
		exp.metrics.start()
	}

	return nil
}

func (exp *grpcExporter) Shutdown(context.Context) error {
	defer http.DefaultTransport.(*http.Transport).CloseIdleConnections()
	if exp.metrics != nil {
		exp.metrics.shutdown()
	}
	if exp.conn != nil {
		if err := exp.conn.Close(); err != nil {
			return fmt.Errorf("connection close: %s", err)
		}
	}
	return nil
}

func (exp *grpcExporter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	payloads, err := exp.marshaler.MarshalRawLogs(ctx, ld)
	if err != nil {
		return fmt.Errorf("marshal logs: %w", err)
	}
	for _, payload := range payloads {
		if err := exp.uploadToChronicle(ctx, payload); err != nil {
			return err
		}
		batchBytes := exp.countBatchBytes(payload.Batch)
		exp.telemetry.ExporterRawBytes.Add(
			ctx,
			int64(batchBytes),
			metric.WithAttributeSet(exp.metricAttributes),
		)
	}
	return nil
}

func (exp *grpcExporter) countBatchBytes(batch *api.LogEntryBatch) uint {
	batchBytes := uint(0)
	for _, entries := range batch.Entries {
		batchBytes += uint(len(entries.Data))
	}
	return batchBytes
}

func (exp *grpcExporter) uploadToChronicle(ctx context.Context, request *api.BatchCreateLogsRequest) error {
	if exp.metrics != nil {
		totalLogs := int64(len(request.GetBatch().GetEntries()))
		defer exp.metrics.recordSent(totalLogs)
	}

	// Track request latency
	start := time.Now()

	_, err := exp.client.BatchCreateLogs(ctx, request, exp.buildOptions()...)
	if err != nil {
		errCode := status.Code(err)
		switch errCode {
		// These errors are potentially transient
		// TODO interpret with https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/internal/coreinternal/errorutil/grpc.go
		case codes.Canceled,
			codes.Unavailable,
			codes.DeadlineExceeded,
			codes.ResourceExhausted,
			codes.Aborted:

			errAttr := attribute.String(attrError, errCode.String())
			exp.telemetry.ExporterRequestLatency.Record(
				ctx, time.Since(start).Milliseconds(),
				metric.WithAttributeSet(attribute.NewSet(errAttr)),
			)
			exp.telemetry.ExporterRequestCount.Add(ctx, 1,
				metric.WithAttributeSet(attribute.NewSet(errAttr)))

			return fmt.Errorf("upload logs to chronicle: %w", err)
		default:
			exp.telemetry.ExporterRequestCount.Add(ctx, 1,
				metric.WithAttributeSet(attribute.NewSet(attrErrorUnknown)))

			return consumererror.NewPermanent(fmt.Errorf("upload logs to chronicle: %w", err))
		}
	}

	exp.telemetry.ExporterRequestLatency.Record(ctx, time.Since(start).Milliseconds())
	exp.telemetry.ExporterRequestCount.Add(ctx, 1,
		metric.WithAttributeSet(attribute.NewSet(attrErrorNone)))

	if exp.metrics != nil {
		totalLogs := int64(len(request.GetBatch().GetEntries()))
		exp.metrics.recordSent(totalLogs)
	}

	return nil
}

func (exp *grpcExporter) buildOptions() []grpc.CallOption {
	opts := make([]grpc.CallOption, 0)
	if exp.cfg.Compression == grpcgzip.Name {
		opts = append(opts, grpc.UseCompressor(grpcgzip.Name))
	}
	return opts
}

// Override for testing
var grpcClientParams = func(cfgEndpoint string, ts oauth2.TokenSource) (string, []grpc.DialOption) {
	return cfgEndpoint + ":443", []grpc.DialOption{
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: ts}),
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
	}
}
