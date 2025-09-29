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

package macosunifiedloggingencodingextension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.uber.org/zap"
)

func TestFactory(t *testing.T) {
	factory := NewFactory()
	assert.NotNil(t, factory)
	assert.Equal(t, "macosunifiedlogencoding", factory.Type().String())
}

func TestDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	config, ok := cfg.(*Config)
	require.True(t, ok)
	assert.False(t, config.DebugMode)
}

func TestCreateExtension(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	extension, err := factory.Create(
		context.Background(),
		extensiontest.NewNopSettings(factory.Type()),
		cfg,
	)

	require.NoError(t, err)
	assert.NotNil(t, extension)

	// Start the extension
	err = extension.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// Shutdown the extension
	err = extension.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestUnmarshalLogs_EmptyData(t *testing.T) {
	ext := &MacosUnifiedLoggingExtension{
		config: &Config{DebugMode: false},
		logger: zap.NewNop(),
	}

	err := ext.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// Test empty data
	logs, err := ext.UnmarshalLogs([]byte{})
	require.NoError(t, err)
	assert.Equal(t, 0, logs.ResourceLogs().Len())
}

func TestUnmarshalLogs_HeaderBinaryData(t *testing.T) {
	ext := &MacosUnifiedLoggingExtension{
		config: &Config{DebugMode: true},
		logger: zap.NewNop(),
	}

	err := ext.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// Test with some binary data
	testData := dataForTest()

	logs, err := ext.UnmarshalLogs(testData)
	require.NoError(t, err)
	assert.Equal(t, 0, logs.ResourceLogs().Len())

	// TODO: add better test data to include integration tests for Chunks with debug mode on

	// resourceLogs := logs.ResourceLogs().At(0)
	// assert.Equal(t, 1, resourceLogs.ScopeLogs().Len())

	// scopeLogs := resourceLogs.ScopeLogs().At(0)
	// assert.Equal(t, 1, scopeLogs.LogRecords().Len())

	// logRecord := scopeLogs.LogRecords().At(0)

	// // Check the message contains information about the binary data
	// message := logRecord.Body().AsString()
	// assert.Contains(t, message, "Read 16 bytes of macOS Unified Logging binary data")
	// assert.Contains(t, message, "000102030405060708090a0b0c0d0e0f")

	// // Check attributes
	// attrs := logRecord.Attributes()
	// source, exists := attrs.Get("source")
	// assert.True(t, exists)
	// assert.Equal(t, "macos_unified_logging", source.AsString())

	// dataSize, exists := attrs.Get("data_size")
	// assert.True(t, exists)
	// assert.Equal(t, int64(16), dataSize.Int())

	// decoded, exists := attrs.Get("decoded")
	// assert.True(t, exists)
	// assert.False(t, decoded.Bool())
}

func TestUnmarshalLogs_DebugModeOff(t *testing.T) {
	ext := &MacosUnifiedLoggingExtension{
		config: &Config{DebugMode: false},
		logger: zap.NewNop(),
	}

	err := ext.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// Test with some binary data but debug mode off
	testData := dataForTest()

	logs, err := ext.UnmarshalLogs(testData)
	require.NoError(t, err)
	assert.Equal(t, 0, logs.ResourceLogs().Len())

	// TODO: add better test data to include integration tests for Chunks with debug mode off
	// logRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	// message := logRecord.Body().AsString()

	// // Should not contain hex dump when debug mode is off
	// assert.Contains(t, message, "Read 4 bytes of macOS Unified Logging binary data")
	// assert.NotContains(t, message, "00010203")
}

func dataForTest() []byte {
	return []byte{
		0, 16, 0, 0, 17, 0, 0, 0, 208, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 15, 105,
		217, 162, 204, 126, 0, 0, 48, 215, 18, 98, 0, 0, 0, 0, 203, 138, 9, 0, 44, 1, 0, 0, 0,
		0, 0, 0, 1, 0, 0, 0, 0, 97, 0, 0, 8, 0, 0, 0, 6, 112, 124, 198, 169, 153, 1, 0, 1, 97,
		0, 0, 56, 0, 0, 0, 7, 0, 0, 0, 8, 0, 0, 0, 50, 49, 65, 53, 53, 57, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 77, 97, 99, 66, 111, 111, 107, 80, 114, 111, 49, 54, 44, 49, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 97, 0, 0, 24, 0, 0, 0, 195, 32, 184, 206, 151,
		250, 77, 165, 159, 49, 125, 57, 46, 56, 156, 234, 85, 0, 0, 0, 0, 0, 0, 0, 3, 97, 0, 0,
		48, 0, 0, 0, 47, 118, 97, 114, 47, 100, 98, 47, 116, 105, 109, 101, 122, 111, 110, 101,
		47, 122, 111, 110, 101, 105, 110, 102, 111, 47, 65, 109, 101, 114, 105, 99, 97, 47, 78,
		101, 119, 95, 89, 111, 114, 107, 0, 0, 0, 0, 0, 0,
	}
}
