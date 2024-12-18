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
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	grpcgzip "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/protobuf/encoding/protojson"
)

const httpScope = "https://www.googleapis.com/auth/cloud-platform"

type httpExporter struct {
	cfg       *Config
	set       component.TelemetrySettings
	marshaler *protoMarshaler
	client    *http.Client
}

func newHTTPExporter(cfg *Config, params exporter.Settings) (*httpExporter, error) {
	marshaler, err := newProtoMarshaler(*cfg, params.TelemetrySettings)
	if err != nil {
		return nil, fmt.Errorf("create proto marshaler: %w", err)
	}
	return &httpExporter{
		cfg:       cfg,
		set:       params.TelemetrySettings,
		marshaler: marshaler,
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
	return nil
}

func (exp *httpExporter) Shutdown(context.Context) error {
	defer http.DefaultTransport.(*http.Transport).CloseIdleConnections()
	t := exp.client.Transport.(*oauth2.Transport)
	if t.Base != nil {
		t.Base.(*http.Transport).CloseIdleConnections()
	}
	return nil
}

func (exp *httpExporter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	payloads, err := exp.marshaler.MarshalRawLogsForHTTP(ctx, ld)
	if err != nil {
		return fmt.Errorf("marshal logs: %w", err)
	}
	for logType, logTypePayloads := range payloads {
		for _, payload := range logTypePayloads {
			if err := exp.uploadToChronicleHTTP(ctx, payload, logType); err != nil {
				return fmt.Errorf("upload to chronicle: %w", err)
			}
		}
	}
	return nil
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

	resp, err := exp.client.Do(request)
	if err != nil {
		return fmt.Errorf("send request to Chronicle: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err == nil && resp.StatusCode == http.StatusOK {
		return nil
	}

	if err != nil {
		exp.set.Logger.Warn("Failed to read response body", zap.Error(err))
	} else {
		exp.set.Logger.Warn("Received non-OK response from Chronicle", zap.String("status", resp.Status), zap.ByteString("response", respBody))
	}

	// TODO interpret with https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/internal/coreinternal/errorutil/http.go
	statusErr := errors.New(resp.Status)
	switch resp.StatusCode {
	case http.StatusInternalServerError, http.StatusServiceUnavailable: // potentially transient
		return statusErr
	default:
		return consumererror.NewPermanent(statusErr)
	}
}

// This uses the DataPlane URL for the request
// URL for the request: https://{region}-chronicle.googleapis.com/{version}/projects/{project}/location/{region}/instances/{customerID}
// Override for testing
var httpEndpoint = func(cfg *Config, logType string) string {
	formatString := "https://%s-%s/v1alpha/projects/%s/locations/%s/instances/%s/logTypes/%s/logs:import"
	return fmt.Sprintf(formatString, cfg.Location, cfg.Endpoint, cfg.Project, cfg.Location, cfg.CustomerID, logType)
}
