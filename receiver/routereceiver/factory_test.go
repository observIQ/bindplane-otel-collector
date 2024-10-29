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

package routereceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
)

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	require.Equal(t, component.NewID(componentType).Type(), f.Type())
	require.Equal(t, stability, f.MetricsStability())
	require.NotNil(t, f.CreateDefaultConfig())
	require.NotNil(t, f.CreateMetrics)
	require.NotNil(t, f.CreateLogs)
	require.NotNil(t, f.CreateTraces)
}

func TestCreateMetricsReceiver(t *testing.T) {
	f := NewFactory()
	r, err := f.CreateMetrics(context.Background(), receiver.Settings{}, createDefaultConfig(), nil)
	require.NoError(t, err)
	require.IsType(t, &routeReceiver{}, r)
}

func TestCreateLogsReceiver(t *testing.T) {
	f := NewFactory()
	r, err := f.CreateLogs(context.Background(), receiver.Settings{}, createDefaultConfig(), nil)
	require.NoError(t, err)
	require.IsType(t, &routeReceiver{}, r)
}

func TestCreateTracesReceiver(t *testing.T) {
	f := NewFactory()
	r, err := f.CreateTraces(context.Background(), receiver.Settings{}, createDefaultConfig(), nil)
	require.NoError(t, err)
	require.IsType(t, &routeReceiver{}, r)
}
