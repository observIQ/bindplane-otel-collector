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

package snapshotprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	require.Equal(t, componentType, factory.Type())

	expectedCfg := &Config{
		Enabled: true,
		OpAMP:   defaultOpAMPExtensionID,
	}

	cfg, ok := factory.CreateDefaultConfig().(*Config)
	require.True(t, ok)
	require.Equal(t, expectedCfg, cfg)
}

func TestCreateOrGetProcessorProcessor(t *testing.T) {
	p1Settings := processortest.NewNopCreateSettings()
	p1Settings.ID = component.MustNewIDWithName(componentType.String(), "proc1")

	p1 := createOrGetProcessor(p1Settings, createDefaultConfig().(*Config))
	p1Copy := createOrGetProcessor(p1Settings, createDefaultConfig().(*Config))

	// p1 and p1Copy should be the same pointer
	require.True(t, p1 == p1Copy, "p1 and p1Copy are not the same pointer")

	p2Settings := processortest.NewNopCreateSettings()
	p2Settings.ID = component.MustNewIDWithName(componentType.String(), "proc2")

	p2 := createOrGetProcessor(p2Settings, createDefaultConfig().(*Config))
	require.True(t, p2 != p1, "p2 and p1 are the same, but they should be different objects")
}
