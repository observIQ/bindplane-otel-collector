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
	// Endpoint is the Microsoft Sentinel Data Collection Endpoint (DCE)
	Endpoint string `mapstructure:"endpoint"`

	// CredentialPath is the path to the Azure credential file
	CredentialPath string `mapstructure:"credential_path"`

	// RuleID is the Data Collection Rule (DCR) ID or immutableId
	RuleID string `mapstructure:"rule_id"`

	// StreamName is the name of the custom log table in Microsoft Sentinel
	StreamName string `mapstructure:"stream_name"`

	// TimeoutSettings configures timeout for the exporter
	TimeoutSettings exporterhelper.TimeoutConfig `mapstructure:",squash"`

	// QueueSettings configures batching
	QueueSettings exporterhelper.QueueConfig `mapstructure:",squash"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return errors.New("endpoint is required")
	}

	if c.CredentialPath == "" {
		return errors.New("credential_path is required")
	}

	if c.RuleID == "" {
		return errors.New("rule_id is required")
	}

	if c.StreamName == "" {
		return errors.New("stream_name is required")
	}

	return nil
}
