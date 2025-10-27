// Copyright observIQ, Inc.
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

package pcapreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory, "factory should not be nil")
	require.Equal(t, "pcap", factory.Type().String(), "factory type should be 'pcap'")
}

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	require.NotNil(t, cfg, "config should not be nil")

	pcapCfg, ok := cfg.(*Config)
	require.True(t, ok, "config should be of type *Config")

	// Verify default values
	require.Equal(t, "", pcapCfg.Interface, "default interface should be empty (auto-detect)")
	require.Equal(t, int32(65535), pcapCfg.SnapLength, "default snap length should be 65535")
	require.Equal(t, "", pcapCfg.BPFFilter, "default BPF filter should be empty")
	require.Equal(t, true, pcapCfg.Promiscuous, "default promiscuous mode should be true")
}

func TestCreateLogsReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := createDefaultConfig()

	set := receivertest.NewNopSettings(factory.Type())
	receiver, err := factory.CreateLogs(context.Background(), set, cfg, consumertest.NewNop())

	require.NoError(t, err, "creating logs receiver should not fail")
	require.NotNil(t, receiver, "logs receiver should not be nil")
}
