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

package snapshotprocessor_v2

import (
	"context"
	"testing"

	contrib "github.com/observiq/bindplane-otel-contrib/processor/snapshotprocessor"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor"
)

func TestFactoryType(t *testing.T) {
	f := NewFactory()
	require.Equal(t, "snapshotprocessor_v2", f.Type().String())
}

func TestFactoryDefaultConfigMatchesContrib(t *testing.T) {
	wrapper := NewFactory().CreateDefaultConfig()
	contribCfg := contrib.NewFactory().CreateDefaultConfig()
	require.Equal(t, contribCfg, wrapper, "wrapper must defer to contrib for default config so the schema stays in lockstep")
}

func TestFactoryStabilityMirrorsContrib(t *testing.T) {
	wrapper := NewFactory()
	c := contrib.NewFactory()
	require.Equal(t, c.TracesStability(), wrapper.TracesStability())
	require.Equal(t, c.LogsStability(), wrapper.LogsStability())
	require.Equal(t, c.MetricsStability(), wrapper.MetricsStability())
}

func TestCreateProcessorsAcceptOurType(t *testing.T) {
	// Verifies that the wrapper rewrites the ID before delegating, so contrib's
	// component-type validation accepts a Settings whose ID has type
	// "snapshotprocessor_v2".
	f := NewFactory()
	cfg := f.CreateDefaultConfig()

	set := processor.Settings{
		ID:                component.NewIDWithName(component.MustNewType("snapshotprocessor_v2"), "test"),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}

	t.Run("traces", func(t *testing.T) {
		p, err := f.CreateTraces(context.Background(), set, cfg, consumertest.NewNop())
		require.NoError(t, err)
		require.NotNil(t, p)
	})
	t.Run("logs", func(t *testing.T) {
		p, err := f.CreateLogs(context.Background(), set, cfg, consumertest.NewNop())
		require.NoError(t, err)
		require.NotNil(t, p)
	})
	t.Run("metrics", func(t *testing.T) {
		p, err := f.CreateMetrics(context.Background(), set, cfg, consumertest.NewNop())
		require.NoError(t, err)
		require.NotNil(t, p)
	})
}
