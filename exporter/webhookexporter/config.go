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
package webhookexporter

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Endpoint represents a validated HTTP/HTTPS endpoint URL
type Endpoint string

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (e *Endpoint) unmarshalText(text []byte) error {
	endpoint := string(text)
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		return fmt.Errorf("endpoint must start with http:// or https://, got: %s", endpoint)
	}
	*e = Endpoint(endpoint)
	return nil
}

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
func (v *HTTPVerb) unmarshalText(text []byte) error {
	verb := HTTPVerb(text)
	switch verb {
	case POST, PATCH, PUT:
		*v = verb
		return nil
	default:
		return fmt.Errorf("invalid HTTP verb: %s, must be one of: POST, PATCH, PUT", text)
	}
}

type Config struct {
	LogsConfig    *SignalConfig `mapstructure:"logs,omitempty"`
	MetricsConfig *SignalConfig `mapstructure:"metrics,omitempty"`
	TracesConfig  *SignalConfig `mapstructure:"traces,omitempty"`
}

type SignalConfig struct {
	// TimeoutConfig contains settings for request timeouts
	TimeoutConfig exporterhelper.TimeoutConfig `mapstructure:",squash"`

	// QueueBatchConfig contains settings for the sending queue and batching
	QueueBatchConfig exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`

	// BackOffConfig contains settings for retry behavior on failures
	BackOffConfig configretry.BackOffConfig `mapstructure:"retry_on_failure"`

	// Endpoint is the URL where the webhook requests will be sent
	// Must start with http:// or https://
	Endpoint Endpoint `mapstructure:"endpoint"`

	// Verb specifies the HTTP method to use for the webhook requests
	// Must be one of: POST, PATCH, PUT
	Verb HTTPVerb `mapstructure:"verb"`

	// Headers contains additional HTTP headers to include in the webhook requests
	Headers map[string]string `mapstructure:"headers"`

	// ContentType specifies the Content-Type header for the webhook requests
	// This field is required
	ContentType string `mapstructure:"content_type"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *configtls.ClientConfig `mapstructure:"tls"`

	// Limit specifies the maximum number of signals to send in a single request
	Limit int `mapstructure:"limit"`
}

// Validate checks if the configuration is valid
func (c *SignalConfig) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	if c.Verb == "" {
		return fmt.Errorf("verb is required")
	}
	if c.ContentType == "" {
		return fmt.Errorf("content_type is required")
	}

	if err := c.Endpoint.unmarshalText([]byte(c.Endpoint)); err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}
	if err := c.Verb.unmarshalText([]byte(c.Verb)); err != nil {
		return fmt.Errorf("invalid verb: %w", err)
	}

	if c.TLSSetting != nil {
		if err := c.TLSSetting.Validate(); err != nil {
			return fmt.Errorf("invalid tls setting: %w", err)
		}
	}

	return nil
}

func (c *Config) Validate() error {
	if c.LogsConfig != nil {
		if err := c.LogsConfig.Validate(); err != nil {
			return err
		}
	}
	if c.MetricsConfig != nil {
		if err := c.MetricsConfig.Validate(); err != nil {
			return err
		}
	}
	if c.TracesConfig != nil {
		if err := c.TracesConfig.Validate(); err != nil {
			return err
		}
	}
	if c.LogsConfig == nil && c.MetricsConfig == nil && c.TracesConfig == nil {
		return fmt.Errorf("at least one signal config is required")
	}
	return nil
}
