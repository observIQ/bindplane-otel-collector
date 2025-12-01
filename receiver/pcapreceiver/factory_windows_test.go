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

//go:build windows

package pcapreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestCreateDefaultConfig_Windows(t *testing.T) {
	cfg := createDefaultConfig()
	require.NotNil(t, cfg)

	pcapCfg, ok := cfg.(*Config)
	require.True(t, ok)
	require.Equal(t, `\Device\NPF_{12345678-1234-1234-1234-123456789012}`, pcapCfg.Interface)
	require.Equal(t, 65535, pcapCfg.SnapLen)
	require.True(t, pcapCfg.Promiscuous)
	require.True(t, pcapCfg.ParseAttributes)
}

func TestCreateLogsReceiver_Windows(t *testing.T) {
	factory := NewFactory()
	cfg := createDefaultConfig()

	// Create an observed logger to capture log messages
	core, observed := observer.New(zapcore.WarnLevel)

	settings := receivertest.NewNopSettings(factory.Type())
	settings.Logger = zap.New(core)

	recv, err := factory.CreateLogs(context.Background(), settings, cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, recv)

	// Verify the Npcap warning was logged
	logs := observed.All()
	require.Len(t, logs, 1)
	require.Equal(t, zapcore.WarnLevel, logs[0].Level)
	require.Contains(t, logs[0].Message, "PCAP receiver on Windows requires Npcap")
}
