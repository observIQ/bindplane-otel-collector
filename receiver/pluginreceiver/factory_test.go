// Copyright  observIQ, Inc.
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

package pluginreceiver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
)

func TestCreateReceiver(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         component.Config
		expectedErr error
	}{
		{
			name:        "invalid config type",
			cfg:         &receiver.CreateSettings{},
			expectedErr: errors.New("config is not a plugin receiver config"),
		},
		{
			name: "missing plugin",
			cfg: &Config{
				Path: "./testdata/missing.yaml",
			},
			expectedErr: errors.New("failed to load plugin"),
		},
		{
			name: "invalid plugin yaml",
			cfg: &Config{
				Path: "./testdata/plugin-invalid-yaml.yaml",
			},
			expectedErr: errors.New("failed to load plugin"),
		},
		{
			name: "invalid plugin parameter",
			cfg: &Config{
				Path: "./testdata/plugin-valid.yaml",
				Parameters: map[string]any{
					"env": 5,
				},
			},
			expectedErr: errors.New("invalid plugin parameter"),
		},
		{
			name: "invalid plugin template",
			cfg: &Config{
				Path: "./testdata/plugin-invalid-template.yaml",
				Parameters: map[string]any{
					"env": "prod",
				},
			},
			expectedErr: errors.New("failed to render plugin"),
		},
		{
			name: "valid plugin",
			cfg: &Config{
				Path: "./testdata/plugin-valid.yaml",
				Parameters: map[string]any{
					"env": "prod",
				},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			set := receiver.CreateSettings{}
			consumer := &MockConsumer{}
			emitterFactory := createLogEmitterFactory(consumer)
			receiver, err := createReceiver(tc.cfg, set, emitterFactory)

			switch tc.expectedErr {
			case nil:
				require.NoError(t, err)
				require.IsType(t, &Receiver{}, receiver)
			default:
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr.Error())
			}
		})
	}
}

func TestCreateLogsReceiver(t *testing.T) {
	factory := NewFactory()
	consumer := &MockConsumer{}
	ctx := context.Background()
	set := receiver.CreateSettings{}
	cfg := &Config{
		Path: "./testdata/plugin-valid.yaml",
		Parameters: map[string]any{
			"env": "prod",
		},
	}

	receiver, err := factory.CreateLogsReceiver(ctx, set, cfg, consumer)
	require.NoError(t, err)
	require.IsType(t, &Receiver{}, receiver)
}

func TestCreateMetricsReceiver(t *testing.T) {
	factory := NewFactory()
	consumer := &MockConsumer{}
	ctx := context.Background()
	set := receiver.CreateSettings{}
	cfg := &Config{
		Path: "./testdata/plugin-valid.yaml",
		Parameters: map[string]any{
			"env": "prod",
		},
	}

	receiver, err := factory.CreateMetricsReceiver(ctx, set, cfg, consumer)
	require.NoError(t, err)
	require.IsType(t, &Receiver{}, receiver)
}

func TestCreateTracesReceiver(t *testing.T) {
	factory := NewFactory()
	consumer := &MockConsumer{}
	ctx := context.Background()
	set := receiver.CreateSettings{}
	cfg := &Config{
		Path: "./testdata/plugin-valid.yaml",
		Parameters: map[string]any{
			"env": "prod",
		},
	}

	receiver, err := factory.CreateTracesReceiver(ctx, set, cfg, consumer)
	require.NoError(t, err)
	require.IsType(t, &Receiver{}, receiver)
}

func TestCreateDefaultConfig(t *testing.T) {
	config := createDefaultConfig()

	pluginConfig, ok := config.(*Config)
	require.True(t, ok)
	require.Equal(t, make(map[string]any), pluginConfig.Parameters)
	require.Empty(t, pluginConfig.Path)
}
