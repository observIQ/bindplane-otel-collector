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

package qradar

import (
	"errors"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/internal/expr"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/zap"
)

// Config defines configuration for the QRadar exporter.
type Config struct {
	exporterhelper.TimeoutConfig `mapstructure:",squash"`
	exporterhelper.QueueConfig   `mapstructure:"sending_queue"`
	configretry.BackOffConfig    `mapstructure:"retry_on_failure"`

	// Syslog is the configuration for the connection to QRadar.
	Syslog SyslogConfig `mapstructure:"syslog"`

	// RawLogField is the field name that will be used to send raw logs to QRadar.
	RawLogField string `mapstructure:"raw_log_field"`
}

// SyslogConfig defines configuration for QRadar connection.
type SyslogConfig struct {
	confignet.AddrConfig `mapstructure:",squash"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *configtls.ClientConfig `mapstructure:"tls"`
}

// validate validates the Syslog configuration.
func (s *SyslogConfig) validate() error {
	if s.AddrConfig.Endpoint == "" {
		return errors.New("incomplete syslog configuration: endpoint is required")
	}
	return nil
}

// Validate validates the QRadar exporter configuration.
func (cfg *Config) Validate() error {

	if err := cfg.Syslog.validate(); err != nil {
		return err
	}

	if cfg.RawLogField != "" {
		_, err := expr.NewOTTLLogRecordExpression(cfg.RawLogField, component.TelemetrySettings{
			Logger: zap.NewNop(),
		})
		if err != nil {
			return fmt.Errorf("raw_log_field is invalid: %s", err)
		}
	}
	return nil
}
