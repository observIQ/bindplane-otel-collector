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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	require.NotNil(t, f)
}

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	require.NotNil(t, cfg)
	require.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreateMetricsProcessor(t *testing.T) {
	cfg := createDefaultConfig()
	p, err := createMetricsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	require.NotNil(t, p)
	require.NoError(t, err)
}

func TestCreateMetricsProcessorNilConfig(t *testing.T) {
	_, err := createMetricsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), nil, consumertest.NewNop())
	require.Error(t, err)
}

func TestCreateLogsProcessor(t *testing.T) {
	cfg := createDefaultConfig()
	p, err := createLogsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	require.NotNil(t, p)
	require.NoError(t, err)
}

func TestCreateLogsProcessorNilConfig(t *testing.T) {
	_, err := createLogsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), nil, consumertest.NewNop())
	require.Error(t, err)
}
