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

package splunksearchapireceiver

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
)

var (
	errNonStandaloneSearchQuery = errors.New("only standalone search commands can be used for scraping data")
	// TokenTypeBearer is the token type for Bearer tokens
	TokenTypeBearer = "Bearer"
	// TokenTypeSplunk is the token type for Splunk tokens
	TokenTypeSplunk = "Splunk"
)

// Config struct to represent the configuration for the Splunk Search API receiver
type Config struct {
	confighttp.ClientConfig `mapstructure:",squash"`
	Endpoint                string        `mapstructure:"endpoint"`
	Username                string        `mapstructure:"splunk_username,omitempty"`
	Password                string        `mapstructure:"splunk_password,omitempty"`
	AuthToken               string        `mapstructure:"auth_token,omitempty"`
	TokenType               string        `mapstructure:"token_type,omitempty"`
	Searches                []Search      `mapstructure:"searches"`
	JobPollInterval         time.Duration `mapstructure:"job_poll_interval"`
	StorageID               *component.ID `mapstructure:"storage"`
}

// Search struct to represent a Splunk search
type Search struct {
	Query          string `mapstructure:"query"`
	EarliestTime   string `mapstructure:"earliest_time"`
	LatestTime     string `mapstructure:"latest_time"`
	Limit          int    `mapstructure:"limit"`
	EventBatchSize int    `mapstructure:"event_batch_size"`
}

// Validate validates the Splunk Search API receiver configuration
func (cfg *Config) Validate() error {
	if cfg.Endpoint == "" {
		return errors.New("missing Splunk server endpoint")
	}

	if cfg.AuthToken != "" {
		if cfg.TokenType == "" {
			return errors.New("auth_token provided without a token type")
		}
		if !strings.EqualFold(cfg.TokenType, TokenTypeBearer) && !strings.EqualFold(cfg.TokenType, TokenTypeSplunk) {
			return fmt.Errorf("auth_token provided without a correct token type, valid token types are %v", []string{TokenTypeBearer, TokenTypeSplunk})
		}
		if cfg.Username != "" || cfg.Password != "" {
			return errors.New("auth_token and username/password were both provided, only one can be provided to authenticate with Splunk")
		}
	} else if cfg.Username == "" || cfg.Password == "" {
		return errors.New("missing Splunk basic auth credentials, need username and password")
	}

	if len(cfg.Searches) == 0 {
		return errors.New("at least one search must be provided")
	}

	if cfg.StorageID == nil {
		return errors.New("storage configuration is required for this receiver")
	}

	for _, search := range cfg.Searches {
		if search.Query == "" {
			return errors.New("missing query in search")
		}

		// query must start with "search" command
		if !strings.HasPrefix(search.Query, "search ") {
			return errNonStandaloneSearchQuery
		}

		if strings.Contains(search.Query, "|") {
			return errNonStandaloneSearchQuery
		}

		// ensure user query does not include time parameters
		if strings.Contains(search.Query, "earliest=") ||
			strings.Contains(search.Query, "latest=") ||
			strings.Contains(search.Query, "starttime=") ||
			strings.Contains(search.Query, "endtime=") ||
			strings.Contains(search.Query, "timeformat=") {
			return errors.New("time query parameters must be configured using only the 'earliest_time' and 'latest_time' configuration parameters")
		}

		if search.EarliestTime == "" {
			return errors.New("missing earliest_time in search")
		}
		if search.LatestTime == "" {
			return errors.New("missing latest_time in search")
		}

		// parse time strings to time.Time
		_, err := time.Parse(time.RFC3339, search.EarliestTime)
		if err != nil {
			return errors.New("earliest_time failed to parse as RFC3339")
		}

		_, err = time.Parse(time.RFC3339, search.LatestTime)
		if err != nil {
			return errors.New("latest_time failed to parse as RFC3339")
		}

	}
	return nil
}
