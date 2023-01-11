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
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/otelcol"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestWrapLogger(t *testing.T) {
	baseLogger := zap.NewNop()
	opts := createServiceLoggerOpts(baseLogger)
	serviceLogger := zap.NewNop().WithOptions(opts...)
	require.Equal(t, baseLogger.Core(), serviceLogger.Core())

	infoLevel := serviceLogger.Core().Enabled(zapcore.InfoLevel)
	require.False(t, infoLevel)
}

func TestCreateService(t *testing.T) {
	renderedCfg := &RenderedConfig{}
	configProvider, err := renderedCfg.GetConfigProvider()
	require.NoError(t, err)

	factories := otelcol.Factories{}
	logger := zap.NewNop()
	_, err = createService(factories, configProvider, logger)
	require.NoError(t, err)
}
