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

package regexmatchprocessor_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"

	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor"
)

func TestNewFactory(t *testing.T) {
	f := regexmatchprocessor.NewFactory()
	require.Equal(t, component.MustNewType("regexmatch"), f.Type())

	expectedCfg := &regexmatchprocessor.Config{
		AttributeName: "log.type",
	}

	cfg, ok := f.CreateDefaultConfig().(*regexmatchprocessor.Config)
	require.True(t, ok)
	require.Equal(t, expectedCfg, cfg)
}

func TestBadFactory(t *testing.T) {
	f := regexmatchprocessor.NewFactory()
	cfg := f.CreateDefaultConfig().(*regexmatchprocessor.Config)
	cfg.AttributeName = "invalid"

	_, err := f.CreateLogs(context.Background(), processortest.NewNopSettings(f.Type()), cfg, &consumertest.LogsSink{})
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid config for \"regexmatch\" processor")
}
