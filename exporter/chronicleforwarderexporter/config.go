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

package chronicleforwarderexporter

import (
	"errors"
	"fmt"

	"github.com/observiq/bindplane-agent/expr"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/zap"
)

const (
	// exportTypeSyslog is the syslog export type.
	exportTypeSyslog = "syslog"

	// exportTypeFile is the file export type.
	exportTypeFile = "file"
)

// Config defines configuration for the Chronicle exporter.
type Config struct {
	exporterhelper.TimeoutSettings `mapstructure:",squash"`
	exporterhelper.QueueSettings   `mapstructure:"sending_queue"`
	configretry.BackOffConfig      `mapstructure:"retry_on_failure"`

	// ExportType is the type of export to use.
	ExportType string `mapstructure:"export_type"`

	// Syslog is the configuration for the connection to the Chronicle forwarder.
	Syslog SyslogConfig `mapstructure:"syslog"`

	// File is the configuration for the connection to the Chronicle forwarder.
	File File `mapstructure:"file"`

	// RawLogField is the field name that will be used to send raw logs to Chronicle.
	RawLogField string `mapstructure:"raw_log_field"`
}

// SyslogConfig defines configuration for the Chronicle forwarder connection.
type SyslogConfig struct {
	confignet.AddrConfig `mapstructure:",squash"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *configtls.TLSClientSetting `mapstructure:"tls"`
}

// File defines configuration for sending to.
type File struct {
	// Path is the path to the file to send to Chronicle.
	Path string `mapstructure:"path"`
}

// validate validates the Syslog configuration.
func (s *SyslogConfig) validate() error {
	if s.AddrConfig.Endpoint == "" {
		return errors.New("incomplete syslog configuration: endpoint is required")
	}
	return nil
}

// validate validates the File configuration.
func (f *File) validate() error {
	if f.Path == "" {
		return errors.New("file path is required for file export type")
	}
	return nil
}

// Validate validates the Chronicle exporter configuration.
func (cfg *Config) Validate() error {
	if cfg.ExportType != exportTypeSyslog && cfg.ExportType != exportTypeFile {
		return errors.New("export_type must be either 'syslog' or 'file'")
	}

	if cfg.ExportType == exportTypeSyslog {
		if err := cfg.Syslog.validate(); err != nil {
			return err
		}
	}

	if cfg.ExportType == exportTypeFile {
		if err := cfg.File.validate(); err != nil {
			return err
		}
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
