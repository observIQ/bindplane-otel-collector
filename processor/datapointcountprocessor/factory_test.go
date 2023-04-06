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

package datapointcountprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/processor"
)

func TestNewProcessorFactory(t *testing.T) {
	f := NewFactory()
	require.Equal(t, component.NewID(typeStr).Type(), f.Type())
	require.Equal(t, stability, f.MetricsProcessorStability())
	require.NotNil(t, f.CreateDefaultConfig())
	require.NotNil(t, f.CreateMetricsProcessor)
}

func TestCreateMetricsProcessor(t *testing.T) {
	var testCases = []struct {
		name        string
		cfg         component.Config
		expectedErr string
	}{
		{
			name: "valid config",
			cfg: &Config{
				Match: "true",
			},
		},
		{
			name: "invalid match",
			cfg: &Config{
				Match: "++",
			},
			expectedErr: "invalid match expression",
		},
		{
			name: "invalid attribute",
			cfg: &Config{
				Match:      "true",
				Attributes: map[string]string{"a": "++"},
			},
			expectedErr: "invalid attribute expression",
		},
		{
			name:        "invalid config type",
			cfg:         nil,
			expectedErr: "invalid config type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewFactory()
			p, err := f.CreateMetricsProcessor(context.Background(), processor.CreateSettings{}, tc.cfg, nil)
			if tc.expectedErr == "" {
				require.NoError(t, err)
				require.IsType(t, &metricCountProcessor{}, p)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
			}
		})
	}
}
