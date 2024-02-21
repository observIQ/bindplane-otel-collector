// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package snowflakeexporter

import (
	"errors"
	"fmt"

	"github.com/snowflakedb/gosnowflake"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	defaultDatabase      = "otlp"
	defaultLogsSchema    = "logs"
	defaultMetricsSchema = "metrics"
	defaultTracesSchema  = "traces"
	defaultTable         = "data"
)

// Config is the config for the Snowflake exporter
type Config struct {
	exporterhelper.TimeoutSettings `mapstructure:",squash"`
	exporterhelper.QueueSettings   `mapstructure:"sending_queue"`
	configretry.BackOffConfig      `mapstructure:"retry_on_failure"`

	AccountIdentifier string             `mapstructure:"account_identifier"`
	Username          string             `mapstructure:"username"`
	Password          string             `mapstructure:"password"`
	Warehouse         string             `mapstructure:"warehouse"`
	Role              string             `mapstructure:"role"`
	Database          string             `mapstructure:"database"`
	Parameters        map[string]*string `mapstructure:"parameters"`

	Logs    TelemetryConfig `mapstructure:"logs"`
	Metrics TelemetryConfig `mapstructure:"metrics"`
	Traces  TelemetryConfig `mapstructure:"traces"`

	dsn string
}

// TelemetryConfig is a config used by each telemetry type
type TelemetryConfig struct {
	Schema string `mapstructure:"schema"`
	Table  string `mapstructure:"table"`
}

// Validate ensures the config is correct
func (c *Config) Validate() error {
	if c.AccountIdentifier == "" {
		return errors.New("account_identifier is required")
	}
	if c.Username == "" {
		return errors.New("username is required")
	}
	if c.Password == "" {
		return errors.New("password is required")
	}
	if c.Warehouse == "" {
		return errors.New("warehouse is required")
	}
	if c.Database == "" {
		return errors.New("database cannot be set as empty")
	}
	if c.Logs.Schema == "" {
		return errors.New("logs schema cannot be set as empty")
	}
	if c.Logs.Table == "" {
		return errors.New("logs table cannot be set as empty")
	}
	if c.Metrics.Schema == "" {
		return errors.New("metrics schema cannot be set as empty")
	}
	if c.Metrics.Table == "" {
		return errors.New("metrics table cannot be set as empty")
	}
	if c.Traces.Schema == "" {
		return errors.New("traces schema cannot be set as empty")
	}
	if c.Traces.Table == "" {
		return errors.New("traces table cannot be set as empty")
	}

	return c.buildSnowflakeDSN()
}

func (c *Config) buildSnowflakeDSN() error {
	sf := gosnowflake.Config{}

	sf.User = c.Username
	sf.Password = c.Password
	sf.Account = c.AccountIdentifier

	if c.Parameters != nil {
		sf.Params = c.Parameters
	} else {
		sf.Params = map[string]*string{}
	}
	t := "true"
	sf.Params["client_session_keep_alive"] = &t // to prevent auth token expiration after ~4 hours

	dsn, err := gosnowflake.DSN(&sf)
	if err != nil {
		return fmt.Errorf("failed to build DSN: %w", err)
	}

	c.dsn = dsn
	return nil
}
