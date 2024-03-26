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
	"errors"
	"fmt"
	"os"

	"github.com/observiq/bindplane-agent/exporter/chronicleexporter/protos/api"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/encoding/gzip"
)

const scope = "https://www.googleapis.com/auth/malachite-ingestion"

const baseEndpoint = "malachiteingestion-pa.googleapis.com"

type chronicleExporter struct {
	cfg       *Config
	logger    *zap.Logger
	client    api.IngestionServiceV2Client
	marshaler logMarshaler
}

func newExporter(cfg *Config, params exporter.CreateSettings) (*chronicleExporter, error) {
	creds, err := loadGoogleCredentials(cfg)
	if err != nil {
		return nil, fmt.Errorf("load Google credentials: %w", err)
	}

	ts := creds.TokenSource
	opts := []grpc.DialOption{
		// Apply OAuth tokens for each RPC call
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: ts}),
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
	}

	conn, err := grpc.DialContext(context.Background(), cfg.Endpoint+":443", opts...)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	marshaller, err := newProtoMarshaler(*cfg, params.TelemetrySettings, buildLabels(cfg))
	if err != nil {
		return nil, fmt.Errorf("create proto marshaller: %w", err)
	}

	return &chronicleExporter{
		cfg:       cfg,
		logger:    params.Logger,
		client:    api.NewIngestionServiceV2Client(conn),
		marshaler: marshaller,
	}, nil
}

func loadGoogleCredentials(cfg *Config) (*google.Credentials, error) {
	switch {
	case cfg.Creds != "":
		return google.CredentialsFromJSON(context.Background(), []byte(cfg.Creds), scope)
	case cfg.CredsFilePath != "":
		credsData, err := os.ReadFile(cfg.CredsFilePath)
		if err != nil {
			return nil, fmt.Errorf("read credentials file: %w", err)
		}

		if len(credsData) == 0 {
			return nil, errors.New("credentials file is empty")
		}

		return google.CredentialsFromJSON(context.Background(), credsData, scope)
	default:
		return google.FindDefaultCredentials(context.Background(), scope)
	}
}

func buildLabels(cfg *Config) []*api.Label {
	labels := make([]*api.Label, 0, len(cfg.IngestionLabels))
	for k, v := range cfg.IngestionLabels {
		labels = append(labels, &api.Label{
			Key:   k,
			Value: v,
		})
	}
	return labels
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
		if err := ce.uploadToChronicle(ctx, payload); err != nil {
			return fmt.Errorf("upload to chronicle: %w", err)
		}
	}

	return nil
}

func (ce *chronicleExporter) uploadToChronicle(ctx context.Context, request *api.BatchCreateLogsRequest) error {
	_, err := ce.client.BatchCreateLogs(ctx, request, ce.buildOptions()...)
	if err != nil {
		return fmt.Errorf("upload logs to chronicle: %w", err)
	}

	return nil
}

func (ce *chronicleExporter) buildOptions() []grpc.CallOption {
	opts := make([]grpc.CallOption, 0)

	if ce.cfg.Compression == gzip.Name {
		opts = append(opts, grpc.UseCompressor(gzip.Name))
	}

	return opts
}
