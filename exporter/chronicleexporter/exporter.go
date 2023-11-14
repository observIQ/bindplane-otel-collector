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
	"context"
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
		creds, err = google.CredentialsFromJSON(context.Background(), []byte(cfg.CredsFilePath), scope)
		if err != nil {
			return nil, fmt.Errorf("obtain credentials from JSON: %w", err)
		}
	case cfg.CredsFilePath != "":
		credsData, err := os.ReadFile(cfg.CredsFilePath)
		if err != nil {
			return nil, fmt.Errorf("read credentials file: %w", err)
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
		marshaler:  newMarshaler(*cfg, params.TelemetrySettings),
	}, nil
}

// buildEndpoint builds the endpoint to send logs to based on the region. there is a default endpoint `https://malachiteingestion-pa.googleapis.com`
// but there are also regional endpoints that can be used instead. the regional endpoints are listed here: https://cloud.google.com/chronicle/docs/reference/search-api#regional_endpoints
func buildEndpoint(cfg *Config) string {
	if cfg.Region != "" && regions[cfg.Region] != "" {
		return fmt.Sprintf("%s%s", regions[cfg.Region], apiTarget)
	}
	return fmt.Sprintf("%s%s", baseEndpoint, apiTarget)
}

func (ce *chronicleExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (ce *chronicleExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	udmData, err := ce.marshaler.MarshalRawLogs(ctx, ld)
	if err != nil {
		return fmt.Errorf("marshal logs: %w", err)
	}

	return ce.uploadToChronicle(ctx, udmData)
}

func (ce *chronicleExporter) uploadToChronicle(ctx context.Context, data []byte) error {
	request, err := http.NewRequestWithContext(ctx, "POST", ce.endpoint, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	resp, err := ce.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send request to Chronicle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			ce.logger.Warn("Failed to read response body", zap.Error(err))
		} else {
			ce.logger.Warn("Received non-OK response from Chronicle", zap.String("status", resp.Status), zap.ByteString("body", respBody))
		}
		return fmt.Errorf("received non-OK response from Chronicle: %s", resp.Status)
	}

	return nil
}
