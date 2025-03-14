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

package unrollprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	require.Equal(t, componentType, factory.Type())

	expectedCfg := &Config{
		Field: UnrollFieldBody,
	}

	cfg, ok := factory.CreateDefaultConfig().(*Config)
	require.True(t, ok)
	require.Equal(t, expectedCfg, cfg)
}

func TestBadFactory(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Field = "invalid"

	_, err := factory.CreateLogs(context.Background(), processortest.NewNopSettings(componentType), cfg, &consumertest.LogsSink{})
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid config for \"unroll\" processor")
}
