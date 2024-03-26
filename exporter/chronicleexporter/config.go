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
	"errors"
	"fmt"

	"github.com/observiq/bindplane-agent/expr"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/zap"
	"google.golang.org/grpc/encoding/gzip"
)

const (
	// noCompression is the no compression type.
	noCompression = "none"
)

// Config defines configuration for the Chronicle exporter.
type Config struct {
	exporterhelper.TimeoutSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.QueueSettings   `mapstructure:"sending_queue"`
	configretry.BackOffConfig      `mapstructure:"retry_on_failure"`

	// Endpoint is the URL where Chronicle data will be sent.
	Endpoint string `mapstructure:"endpoint"`

	// CredsFilePath is the file path to the Google credentials JSON file.
	CredsFilePath string `mapstructure:"creds_file_path"`

	// Creds are the Google credentials JSON file.
	Creds string `mapstructure:"creds"`

	// LogType is the type of log that will be sent to Chronicle.
	LogType string `mapstructure:"log_type"`

	// OverrideLogType is a flag that determines whether or not to override the `log_type` in the config with `attributes["log_type"]`.
	OverrideLogType bool `mapstructure:"override_log_type"`

	// RawLogField is the field name that will be used to send raw logs to Chronicle.
	RawLogField string `mapstructure:"raw_log_field"`

	// CustomerID is the customer ID that will be used to send logs to Chronicle.
	CustomerID string `mapstructure:"customer_id"`

	// Namespace is the namespace that will be used to send logs to Chronicle.
	Namespace string `mapstructure:"namespace"`

	// Compression is the compression type that will be used to send logs to Chronicle.
	Compression string `mapstructure:"compression"`

	// IngestionLabels are the labels that will be attached to logs when sent to Chronicle.
	IngestionLabels map[string]string `mapstructure:"ingestion_labels"`
}

// Validate checks if the configuration is valid.
func (cfg *Config) Validate() error {
	if cfg.CredsFilePath != "" && cfg.Creds != "" {
		return errors.New("can only specify creds_file_path or creds")
	}

	if cfg.LogType == "" {
		return errors.New("log_type is required")
	}

	if cfg.RawLogField != "" {
		_, err := expr.NewOTTLLogRecordExpression(cfg.RawLogField, component.TelemetrySettings{
			Logger: zap.NewNop(),
		})
		if err != nil {
			return fmt.Errorf("raw_log_field is invalid: %s", err)
		}
	}

	if cfg.Compression != gzip.Name && cfg.Compression != noCompression {
		return fmt.Errorf("invalid compression type: %s", cfg.Compression)
	}

	return nil
}
