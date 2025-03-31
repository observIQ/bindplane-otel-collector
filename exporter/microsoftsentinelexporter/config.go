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

package microsoftsentinelexporter

import (
	"errors"

	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines the configuration for the Microsoft Sentinel exporter
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

	TimeoutSettings exporterhelper.TimeoutConfig `mapstructure:",squash"`

	QueueSettings exporterhelper.QueueConfig `mapstructure:",squash"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return errors.New("endpoint is required")
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

	return nil
}
