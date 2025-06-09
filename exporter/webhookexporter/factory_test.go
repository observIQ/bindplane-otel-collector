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

package webhookexporter

import (
	"context"
	"testing"

	"github.com/observiq/bindplane-otel-collector/exporter/webhookexporter/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, metadata.Type, factory.Type())
	assert.Equal(t, metadata.LogsStability, factory.LogsStability())
}

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.NotNil(t, cfg)

	webhookCfg, ok := cfg.(*Config)
	require.True(t, ok)
	assert.Equal(t, &SignalConfig{
		ClientConfig: confighttp.ClientConfig{
			Endpoint: "https://localhost",
		},
		Verb:        POST,
		ContentType: "application/json",
	}, webhookCfg.LogsConfig)
}

func TestCreateLogsExporter(t *testing.T) {
	tests := []struct {
		name    string
		config  component.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				LogsConfig: &SignalConfig{
					ClientConfig: confighttp.ClientConfig{
						Endpoint: "https://example.com",
					},
					Verb:        POST,
					ContentType: "application/json",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid config type",
			config: &struct {
				component.Config
			}{},
			wantErr: true,
		},
		{
			name: "invalid config validation",
			config: &Config{
				LogsConfig: &SignalConfig{
					ClientConfig: confighttp.ClientConfig{
						Endpoint: "invalid-url",
					},
					Verb:        "INVALID",
					ContentType: "application/json",
				},
			},
			wantErr: true,
		},
		{
			name: "missing content type",
			config: &Config{
				LogsConfig: &SignalConfig{
					ClientConfig: confighttp.ClientConfig{
						Endpoint: "https://example.com",
					},
					Verb: POST,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := exportertest.NewNopSettings(metadata.Type)
			exporter, err := createLogsExporter(
				context.Background(),
				settings,
				tt.config,
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, exporter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, exporter)
				assert.NoError(t, exporter.Start(context.Background(), componenttest.NewNopHost()))
				assert.NoError(t, exporter.Shutdown(context.Background()))
			}
		})
	}
}
