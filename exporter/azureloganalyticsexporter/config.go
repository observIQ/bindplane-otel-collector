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

package azureloganalyticsexporter

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/observiq/bindplane-otel-collector/expr"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/zap"
)

// Config defines the configuration for the Azure Log Analytics exporter
type Config struct {
	// Endpoint is the DCR or DCE ingestion endpoint
	Endpoint string `mapstructure:"endpoint"`

	// Authenticaton options
	ClientID string `mapstructure:"client_id"`

	ClientSecret string `mapstructure:"client_secret"`

	TenantID string `mapstructure:"tenant_id"`

	// RuleID is the Data Collection Rule (DCR) ID or immutableId
	RuleID string `mapstructure:"rule_id"`

	// StreamName is the name of the custom log table in Log Analytics workspace
	StreamName string `mapstructure:"stream_name"`

	// RawLogField is the field name that will be used to send raw logs to the Log Analytics workspace.
	RawLogField string `mapstructure:"raw_log_field"`

	// TimeoutConfig configures timeout settings for exporter operations.
	TimeoutConfig exporterhelper.TimeoutConfig `mapstructure:",squash"`

	// QueueConfig defines the queuing behavior for the exporter.
	QueueConfig exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`

	// BackOffConfig defines the retry behavior for failed operations.
	BackOffConfig configretry.BackOffConfig `mapstructure:"retry_on_failure"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return errors.New("endpoint is required")
	}

	// Validate endpoint URL format
	endpointURL, err := url.Parse(c.Endpoint)
	if err != nil {
		return fmt.Errorf("endpoint is not a valid URL: %w", err)
	}
	if endpointURL.Scheme == "" {
		return errors.New("endpoint must include scheme (e.g. https://)")
	}
	if endpointURL.Host == "" {
		return errors.New("endpoint must include host")
	}

	if c.ClientID == "" {
		return errors.New("client id is required")
	}

	if c.ClientSecret == "" {
		return errors.New("client secret is required")
	}

	if c.TenantID == "" {
		return errors.New("tenant id is required")
	}

	if c.RuleID == "" {
		return errors.New("rule_id is required")
	}

	if c.StreamName == "" {
		return errors.New("stream_name is required")
	}

	// Validate raw_log_field if it's set
	if c.RawLogField != "" {
		// Create a temporary telemetry settings for validation
		teleSettings := component.TelemetrySettings{
			Logger: zap.NewNop(),
		}

		// Try to create the OTTL expression to validate it
		_, err := expr.NewOTTLLogRecordExpression(c.RawLogField, teleSettings)
		if err != nil {
			return fmt.Errorf("raw_log_field is invalid: %w", err)
		}
	}

	return nil
}
