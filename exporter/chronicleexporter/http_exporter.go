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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	grpcgzip "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/protobuf/encoding/protojson"
)

const httpScope = "https://www.googleapis.com/auth/cloud-platform"

type exists struct{}

type httpExporter struct {
	cfg       *Config
	set       component.TelemetrySettings
	marshaler *protoMarshaler
	client    *http.Client

	telemetry        *metadata.TelemetryBuilder
	metricAttributes attribute.Set
}

func newHTTPExporter(cfg *Config, params exporter.Settings, telemetry *metadata.TelemetryBuilder) (*httpExporter, error) {
	marshaler, err := newProtoMarshaler(*cfg, params.TelemetrySettings, telemetry, params.Logger)
	if err != nil {
		return nil, fmt.Errorf("create proto marshaler: %w", err)
	}
	return &httpExporter{
		cfg:       cfg,
		set:       params.TelemetrySettings,
		marshaler: marshaler,
		telemetry: telemetry,
		metricAttributes: attribute.NewSet(
			attribute.KeyValue{
				Key:   "exporter",
				Value: attribute.StringValue(params.ID.String()),
			},
			attribute.KeyValue{
				Key:   "exporter_type",
				Value: attribute.StringValue(params.ID.Type().String()),
			},
		),
	}, nil
}

func (exp *httpExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (exp *httpExporter) Start(ctx context.Context, _ component.Host) error {
	ts, err := tokenSource(ctx, exp.cfg)
	if err != nil {
		return fmt.Errorf("load Google credentials: %w", err)
	}
	exp.client = oauth2.NewClient(context.Background(), ts)

	if exp.cfg.ValidateLogTypes {
		exp.marshaler.logTypes = exp.loadLogTypes(ctx)
	}

	return nil
}

func (exp *httpExporter) loadLogTypes(ctx context.Context) map[string]exists {

	logTypes := make(map[string]exists)
	endpoint := getLogTypesEndpoint(exp.cfg)

	request, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		exp.set.Logger.Warn("Failed to create request for loading log types", zap.Error(err))
		return nil
	}

	// time out after 10 seconds
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	for {
		if err := ctx.Err(); err != nil {
			exp.set.Logger.Warn("Context cancelled for loading log types", zap.Error(err))
			return nil
		}
		resp, err := exp.client.Do(request)
		if err != nil {
			exp.set.Logger.Warn("Failed to send request to Chronicle for loading log types", zap.Error(err))
			return nil
		}
		// https://cloud.google.com/chronicle/docs/reference/rest/v1alpha/projects.locations.instances.logTypes/list
		type logType struct {
			Name string `json:"name"`
		}

		type logTypeResponse struct {
			LogTypes      []logType `json:"logTypes"`
			NextPageToken string    `json:"nextPageToken"`
		}

		var response logTypeResponse

		respBody, err := io.ReadAll(resp.Body)
		if err := resp.Body.Close(); err != nil {
			exp.set.Logger.Warn("Failed to close response body for loading log types", zap.Error(err))
		}

		if err == nil && resp.StatusCode == http.StatusOK {
			if err := json.Unmarshal(respBody, &response); err != nil {
				exp.set.Logger.Warn("Failed to unmarshal response body", zap.Error(err))
				return nil
			}
			for _, logType := range response.LogTypes {
				logTypes[parseLogTypes(logType.Name)] = exists{}
			}
		}

		if err != nil {
			exp.set.Logger.Warn("Failed to read response body for loading log types", zap.Error(err))
			return nil
		}
		if resp.StatusCode != http.StatusOK {
			exp.set.Logger.Warn("Received non-OK response from Chronicle for loading log types", zap.String("status", resp.Status), zap.ByteString("response", respBody))
			return nil
		}
		if response.NextPageToken == "" {
			break
		}
		request, err = http.NewRequestWithContext(ctx, "GET", endpoint+"?pageToken="+response.NextPageToken, nil)
		if err != nil {
			exp.set.Logger.Warn("Failed to create request for loading log types", zap.Error(err))
			return nil
		}

	}
	return logTypes
}

// API returns a list of log types in the format:
//
//	{
//	     "name": "projects/408460088155/locations/us/instances/b536658e-469e-44a5-b764-d5ab15b72ce0/logTypes/AKAMAI_SIEM_CONNECTOR",
//	     "displayName": "Akamai SIEM Connector"
//	},
//
// we need to get the token after the last /
func parseLogTypes(logTypes string) string {

	parts := strings.Split(logTypes, "/")
	return parts[len(parts)-1]
}

func (exp *httpExporter) Shutdown(context.Context) error {
	defer http.DefaultTransport.(*http.Transport).CloseIdleConnections()
	if exp.client != nil {
		t := exp.client.Transport.(*oauth2.Transport)
		if t.Base != nil {
			t.Base.(*http.Transport).CloseIdleConnections()
		}
	}
	return nil
}

// ConsumeLogs sends logs to Chronicle via HTTP.
//
// Retry behavior: When this function returns an error, the OTel collector's
// exporterhelper will retry the entire batch (ld plog.Logs) from the beginning.
// This means all payloads will be retried, including any that succeeded before
// the error occurred. Chronicle is expected to handle duplicate requests
// idempotently to prevent duplicate log entries.
//
// Metrics: When retry is enabled, raw bytes are only counted on success to prevent
// double-counting across retry attempts. When retry is disabled, bytes are counted
// regardless of success/failure since this is the only attempt to send the data.
func (exp *httpExporter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	payloads, totalBytes, err := exp.marshaler.MarshalRawLogsForHTTP(ctx, ld)
	if err != nil {
		return fmt.Errorf("marshal logs: %w", err)
	}
	successfulPayloads := []*api.ImportLogsRequest{}
	for logType, logTypePayloads := range payloads {
		for _, payload := range logTypePayloads {
			if err := exp.uploadToChronicleHTTP(ctx, payload, logType); err != nil {
				// Track the failure for observability
				exp.telemetry.ExporterLogsSendFailed.Add(ctx, 1, metric.WithAttributeSet(exp.metricAttributes))

				// If retry is disabled, count bytes for payloads that succeeded before this failure
				if !exp.cfg.BackOffConfig.Enabled {
					exp.countAndReportBatchBytes(ctx, successfulPayloads)
				}
				return fmt.Errorf("upload to chronicle: %w", err)
			}
			successfulPayloads = append(successfulPayloads, payload)
		}
	}
	// Count bytes on success (for both retry enabled and disabled cases)
	exp.telemetry.ExporterRawBytes.Add(
		ctx,
		int64(totalBytes),
		metric.WithAttributeSet(exp.metricAttributes),
	)
	return nil
}

func (exp *httpExporter) countAndReportBatchBytes(ctx context.Context, payloads []*api.ImportLogsRequest) {
	totalBytes := uint(0)
	for _, payload := range payloads {
		inlineSource := payload.GetInlineSource()
		if inlineSource == nil {
			exp.set.Logger.Warn("Payload source is not InlineSource, skipping bytes calculation")
			continue
		}
		for _, entry := range inlineSource.Logs {
			totalBytes += uint(len(entry.Data))
		}
	}
	if totalBytes > 0 {
		exp.telemetry.ExporterRawBytes.Add(
			ctx,
			int64(totalBytes),
			metric.WithAttributeSet(exp.metricAttributes),
		)
	}
}

func (exp *httpExporter) uploadToChronicleHTTP(ctx context.Context, logs *api.ImportLogsRequest, logType string) error {
	data, err := protojson.Marshal(logs)
	if err != nil {
		return fmt.Errorf("marshal protobuf logs to JSON: %w", err)
	}

	var body io.Reader
	if exp.cfg.Compression == grpcgzip.Name {
		var b bytes.Buffer
		gz := gzip.NewWriter(&b)
		if _, err := gz.Write(data); err != nil {
			return fmt.Errorf("gzip write: %w", err)
		}
		if err := gz.Close(); err != nil {
			return fmt.Errorf("gzip close: %w", err)
		}
		body = &b
	} else {
		body = bytes.NewBuffer(data)
	}

	request, err := http.NewRequestWithContext(ctx, "POST", httpEndpoint(exp.cfg, logType), body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if exp.cfg.Compression == grpcgzip.Name {
		request.Header.Set("Content-Encoding", "gzip")
	}

	request.Header.Set("Content-Type", "application/json")

	// Track request latency
	start := time.Now()

	resp, err := exp.client.Do(request)
	if err != nil {
		logTypeAttr := attribute.String("logType", logType)
		errAttr := attribute.String(attrError, "unknown")
		if errors.Is(err, context.DeadlineExceeded) {
			errAttr = attribute.String(attrError, "timeout")
		}
		exp.telemetry.ExporterRequestLatency.Record(
			ctx, time.Since(start).Milliseconds(),
			metric.WithAttributeSet(attribute.NewSet(errAttr, logTypeAttr)),
		)
		exp.telemetry.ExporterRequestCount.Add(ctx, 1,
			metric.WithAttributeSet(attribute.NewSet(errAttr, logTypeAttr)))
		return fmt.Errorf("send request to Chronicle: %w", err)
	}
	defer resp.Body.Close()

	statusAttr := attribute.String("status", resp.Status)
	exp.telemetry.ExporterRequestLatency.Record(
		ctx, time.Since(start).Milliseconds(),
		metric.WithAttributeSet(attribute.NewSet(statusAttr)),
	)
	exp.telemetry.ExporterRequestCount.Add(ctx, 1,
		metric.WithAttributeSet(attribute.NewSet(attrErrorNone)))

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		exp.set.Logger.Warn("Failed to read response body", zap.Error(err), zap.String("logType", logType))
	}

	exp.set.Logger.Warn("Received non-OK response from Chronicle", zap.String("status", resp.Status), zap.ByteString("response", respBody), zap.String("logType", logType))

	// TODO interpret with https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/internal/coreinternal/errorutil/http.go
	statusErr := errors.New(resp.Status)
	switch resp.StatusCode {
	case http.StatusInternalServerError, http.StatusServiceUnavailable: // potentially transient
		return statusErr
	default:
		if exp.cfg.LogErroredPayloads {
			exp.set.Logger.Warn("Import request rejected", zap.String("logType", logType), zap.String("rejectedRequest", string(data)))
		}
		return consumererror.NewPermanent(statusErr)
	}
}

// This uses the DataPlane URL for the request
// URL for the request: https://{region}-chronicle.googleapis.com/{version}/projects/{project}/location/{region}/instances/{customerID}
// Override for testing
var httpEndpoint = func(cfg *Config, logType string) string {
	formatString := "%s/logTypes/%s/logs:import"
	return fmt.Sprintf(formatString, baseEndpoint(cfg), logType)
}

var getLogTypesEndpoint = func(cfg *Config) string {
	formatString := "%s/logTypes"
	return fmt.Sprintf(formatString, baseEndpoint(cfg))
}

func baseEndpoint(cfg *Config) string {
	formatString := "https://%s-%s/v1alpha/projects/%s/locations/%s/instances/%s"
	return fmt.Sprintf(formatString, cfg.Location, cfg.Endpoint, cfg.Project, cfg.Location, cfg.CustomerID)
}
