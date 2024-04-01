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

package resourceattributetransposerprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/otelcol/otelcoltest"
)

func TestConfig(t *testing.T) {
	factories, err := otelcoltest.NopFactories()
	require.NoError(t, err)

	factory := NewFactory()
	factories.Processors[componentType] = factory
	cfg, err := otelcoltest.LoadConfigAndValidate(path.Join(".", "testdata", "config.yaml"), factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, len(cfg.Processors), 2)

	// Loaded config should be equal to default config
	defaultCfg := factory.CreateDefaultConfig()
	r0 := cfg.Processors[component.NewID(componentType)]
	require.Equal(t, r0, defaultCfg)

	customComponentID := component.NewIDWithName(componentType, "customname")
	r1 := cfg.Processors[customComponentID].(*Config)
	require.Equal(t, &Config{
		Operations: []CopyResourceConfig{
			{
				From: "some.resource.level.attr",
				To:   "some.metricdatapoint.level.attr",
			},
			{
				From: "another.resource.attr",
				To:   "another.datapoint.attr",
			},
		},
	}, r1)
}
