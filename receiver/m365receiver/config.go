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

package m365receiver // import "github.com/observiq/bindplane-agent/receiver/m365receiver"

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/observiq/bindplane-agent/receiver/m365receiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

var clientSecretRegEx = regexp.MustCompile("^[a-zA-Z0-9-_.~]{1,40}$")

// Config defines configuration for Microsoft Office 365 receiver.
type Config struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	confighttp.HTTPClientSettings           `mapstructure:",squash"`
	MetricsBuilderConfig                    metadata.MetricsBuilderConfig `mapstructure:",squash"`
	TenantID                                string                        `mapstructure:"tenant_id"`
	ClientID                                string                        `mapstructure:"client_id"`
	ClientSecret                            string                        `mapstructure:"client_secret"`
	Logs                                    *LogsConfig                   `mapstructure:"logs"`
	StorageID                               *component.ID                 `mapstructure:"storage"`
}

// LogsConfig defines configuration for the M365 logs receiver
type LogsConfig struct {
	PollInterval   time.Duration `mapstructure:"poll_interval"`
	GeneralLogs    bool          `mapstructure:"general"`
	ExchangeLogs   bool          `mapstructure:"exchange"`
	SharepointLogs bool          `mapstructure:"sharepoint"`
	AzureADLogs    bool          `mapstructure:"azureAD"`
	DLPLogs        bool          `mapstructure:"dlp"`
}

// Validate validates the configuration by checking for missing or invalid fields
func (c *Config) Validate() error {
	if c.TenantID == "" {
		return fmt.Errorf("missing tenant_id; required")
	}
	_, err := uuid.Parse(c.TenantID)
	if err != nil {
		return fmt.Errorf("tenant_id is invalid; must be a GUID")
	}

	if c.ClientID == "" {
		return fmt.Errorf("missing client_id; required")
	}
	_, err = uuid.Parse(c.ClientID)
	if err != nil {
		return fmt.Errorf("client_id is invalid; must be a GUID")
	}

	if c.ClientSecret == "" {
		return fmt.Errorf("missing client_secret; required")
	}

	if !clientSecretRegEx.MatchString(c.ClientSecret) {
		return fmt.Errorf("client_secret is invalid; does not follow correct structure")
	}

	return nil
}
