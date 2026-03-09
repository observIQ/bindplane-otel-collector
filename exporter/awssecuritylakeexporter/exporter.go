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

package awssecuritylakeexporter // import "github.com/observiq/bindplane-otel-collector/exporter/awssecuritylakeexporter"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// securityLakeExporter exports OCSF-formatted logs as Parquet to AWS Security Lake S3.
type securityLakeExporter struct {
	cfg    *Config
	logger *zap.Logger
}

// newExporter creates a new Security Lake exporter.
func newExporter(cfg *Config, params exporter.Settings) (*securityLakeExporter, error) {
	return &securityLakeExporter{
		cfg:    cfg,
		logger: params.Logger,
	}, nil
}

// Capabilities returns the exporter's capabilities.
func (e *securityLakeExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// logsDataPusher pushes OCSF logs to Security Lake as Parquet files on S3.
func (e *securityLakeExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	return nil
}
