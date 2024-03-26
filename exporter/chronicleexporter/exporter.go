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
	"os"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const scope = "https://www.googleapis.com/auth/malachite-ingestion"

const baseEndpoint = "https://malachiteingestion-pa.googleapis.com"

const apiTarget = "/v2/unstructuredlogentries:batchCreate"

type chronicleExporter struct {
	cfg        *Config
	logger     *zap.Logger
	httpClient *http.Client
	marshaler  logMarshaler
	endpoint   string
}

func newExporter(cfg *Config, params exporter.CreateSettings) (*chronicleExporter, error) {
	var creds *google.Credentials
	var err error

	switch {
	case cfg.Creds != "":
		creds, err = google.CredentialsFromJSON(context.Background(), []byte(cfg.Creds), scope)
		if err != nil {
			return nil, fmt.Errorf("obtain credentials from JSON: %w", err)
		}
	case cfg.CredsFilePath != "":
		credsData, err := os.ReadFile(cfg.CredsFilePath)
		if err != nil {
			return nil, fmt.Errorf("read credentials file: %w", err)
		}

		if len(credsData) == 0 {
			return nil, errors.New("credentials file is empty")
		}

		creds, err = google.CredentialsFromJSON(context.Background(), credsData, scope)
		if err != nil {
			return nil, fmt.Errorf("obtain credentials from JSON: %w", err)
		}
	default:
		creds, err = google.FindDefaultCredentials(context.Background(), scope)
		if err != nil {
			return nil, fmt.Errorf("find default credentials: %w", err)
		}
	}

	// Use the credentials to create an HTTP client
	httpClient := oauth2.NewClient(context.Background(), creds.TokenSource)

	return &chronicleExporter{
		endpoint:   buildEndpoint(cfg),
		cfg:        cfg,
		logger:     params.Logger,
		httpClient: httpClient,
		marshaler:  newMarshaler(*cfg, params.TelemetrySettings, buildLabels(cfg)),
	}, nil
}

func buildLabels(cfg *Config) []label {
	labels := make([]label, 0, len(cfg.IngestionLabels))
	for k, v := range cfg.IngestionLabels {
		labels = append(labels, label{
			Key:   k,
			Value: v,
		})
	}
	return labels
}

// buildEndpoint builds the endpoint to send logs to based on the region. there is a default endpoint `https://malachiteingestion-pa.googleapis.com`
// but there are also regional endpoints that can be used instead. the regional endpoints are listed here: https://cloud.google.com/chronicle/docs/reference/search-api#regional_endpoints
func buildEndpoint(cfg *Config) string {
	return fmt.Sprintf("%s%s", cfg.Endpoint, apiTarget)
}

func (ce *chronicleExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (ce *chronicleExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	payloads, err := ce.marshaler.MarshalRawLogs(ctx, ld)
	if err != nil {
		return fmt.Errorf("marshal logs: %w", err)
	}

	for _, payload := range payloads {
		data, err := json.Marshal(payload)
		if err != nil {
			ce.logger.Warn("Failed to marshal payload", zap.Error(err))
			continue
		}

		if err := ce.uploadToChronicle(ctx, data); err != nil {
			return fmt.Errorf("upload to Chronicle: %w", err)
		}
	}

	return nil
}

func (ce *chronicleExporter) uploadToChronicle(ctx context.Context, data []byte) error {
	var body io.Reader

	if ce.cfg.Compression == gzipCompression {
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

	request, err := http.NewRequestWithContext(ctx, "POST", ce.endpoint, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if ce.cfg.Compression == gzipCompression {
		request.Header.Set("Content-Encoding", "gzip")
	}

	request.Header.Set("Content-Type", "application/json")

	resp, err := ce.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send request to Chronicle: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		if err != nil {
			ce.logger.Warn("Failed to read response body", zap.Error(err))
		} else {
			ce.logger.Warn("Received non-OK response from Chronicle", zap.String("status", resp.Status), zap.ByteString("response", respBody))
		}
		return fmt.Errorf("received non-OK response from Chronicle: %s", resp.Status)
	}

	ce.logger.Debug("Received response from Chronicle", zap.String("status", resp.Status), zap.ByteString("response", respBody))
	return nil
}
