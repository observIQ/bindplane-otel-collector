// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package webhookexporter implements an OpenTelemetry Logs exporter that sends logs to a webhook.
package webhookexporter

import (
	"fmt"
	"net/url"

	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// HTTPVerb represents the allowed HTTP methods for the webhook exporter
type HTTPVerb string

const (
	// POST represents the HTTP POST method
	POST HTTPVerb = "POST"
	// PATCH represents the HTTP PATCH method
	PATCH HTTPVerb = "PATCH"
	// PUT represents the HTTP PUT method
	PUT HTTPVerb = "PUT"
)

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (v *HTTPVerb) UnmarshalText(text []byte) error {
	verb := HTTPVerb(text)
	switch verb {
	case POST, PATCH, PUT:
		*v = verb
		return nil
	default:
		return fmt.Errorf("invalid HTTP verb: %s, must be one of: POST, PATCH, PUT", text)
	}
}

// Config defines the configuration for the webhookexporter
type Config struct {
	LogsConfig *SignalConfig `mapstructure:"logs,omitempty"`
}

// SignalConfig defines the configuration for a single signal type (logs, metrics, traces)
type SignalConfig struct {
	confighttp.ClientConfig `mapstructure:",squash"`

	// QueueBatchConfig contains settings for the sending queue and batching
	QueueBatchConfig exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`

	// BackOffConfig contains settings for retry behavior on failures
	BackOffConfig configretry.BackOffConfig `mapstructure:"retry_on_failure"`

	// Verb specifies the HTTP method to use for the webhook requests
	// Must be one of: POST, PATCH, PUT
	Verb HTTPVerb `mapstructure:"verb"`

	// ContentType specifies the Content-Type header for the webhook requests
	// This field is required
	ContentType string `mapstructure:"content_type"`

	// ConfigAuth contains the authentication configuration for the webhook requests
	ConfigAuth *configauth.Config `mapstructure:"auth"`
}

// Validate checks if the configuration is valid
func (c *SignalConfig) Validate() error {
	endpoint, err := url.Parse(c.ClientConfig.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}
	if endpoint.String() == "" {
		return fmt.Errorf("endpoint is required")
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return fmt.Errorf("endpoint must start with http:// or https://, got: %s", endpoint.String())
	}

	if c.Verb == "" {
		return fmt.Errorf("verb is required")
	}
	if c.ContentType == "" {
		return fmt.Errorf("content_type is required")
	}

	if err := c.Verb.UnmarshalText([]byte(c.Verb)); err != nil {
		return fmt.Errorf("invalid verb: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.LogsConfig != nil {
		if err := c.LogsConfig.Validate(); err != nil {
			return fmt.Errorf("logs config validation failed: %w", err)
		}
	} else {
		return fmt.Errorf("logs config is required")
	}
	return nil
}
