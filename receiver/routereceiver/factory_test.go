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
)

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	require.Equal(t, component.NewID(typeStr).Type(), f.Type())
	require.Equal(t, stability, f.MetricsReceiverStability())
	require.NotNil(t, f.CreateDefaultConfig())
	require.NotNil(t, f.CreateMetricsReceiver)
	require.NotNil(t, f.CreateLogsReceiver)
	require.NotNil(t, f.CreateTracesReceiver)
}

func TestCreateMetricsReceiver(t *testing.T) {
	f := NewFactory()
	r, err := f.CreateMetricsReceiver(context.Background(), component.ReceiverCreateSettings{}, createDefaultConfig(), nil)
	require.NoError(t, err)
	require.IsType(t, &receiver{}, r)
}

func TestCreateLogsReceiver(t *testing.T) {
	f := NewFactory()
	r, err := f.CreateLogsReceiver(context.Background(), component.ReceiverCreateSettings{}, createDefaultConfig(), nil)
	require.NoError(t, err)
	require.IsType(t, &receiver{}, r)
}

func TestCreateTracesReceiver(t *testing.T) {
	f := NewFactory()
	r, err := f.CreateTracesReceiver(context.Background(), component.ReceiverCreateSettings{}, createDefaultConfig(), nil)
	require.NoError(t, err)
	require.IsType(t, &receiver{}, r)
}
