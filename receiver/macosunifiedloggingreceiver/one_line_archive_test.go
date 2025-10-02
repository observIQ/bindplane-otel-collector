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

package macosunifiedloggingreceiver

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"
	"github.com/observiq/bindplane-otel-collector/receiver/macosunifiedloggingreceiver/internal/metadata"
)

func TestParseOneLineArchive(t *testing.T) {
	baseArchivePath := filepath.Join("testdata", "one-line.logarchive")
	traceV3Paths := filepath.Join(baseArchivePath, "**", "*.tracev3")

	// traceV3Paths := filepath.Join(baseArchivePath, "Persist", "00000000000060d5.tracev3")
	// timesyncPaths := filepath.Join(baseArchivePath, "timesync", "*.timesync")
	// dscPaths := filepath.Join(baseArchivePath, "dsc", "*.dsc")

	extFactory := macosunifiedloggingencodingextension.NewFactory()
	extCfg := extFactory.CreateDefaultConfig()

	ext, err := extFactory.Create(
		context.Background(),
		extensiontest.NewNopSettings(extFactory.Type()),
		extCfg,
	)
	require.NoError(t, err)

	host := &mockHostForTest{
		extensions: map[component.ID]component.Component{
			component.MustNewID("macosunifiedlogencoding"): ext,
		},
	}

	set := receivertest.NewNopSettings(metadata.Type)
	// set.Logger = zap.NewNop()
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Encoding = "macosunifiedlogencoding"
	cfg.TraceV3Paths = []string{traceV3Paths}
	cfg.TimesyncPaths = []string{}
	cfg.DSCPaths = []string{}
	cfg.UUIDTextPaths = []string{}
	cfg.StartAt = "beginning"
	cfg.PollInterval = 100 * time.Millisecond

	sink := new(consumertest.LogsSink)
	rcv, err := factory.CreateLogs(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, rcv)

	// Test that we can start and stop the receiver
	err = ext.Start(context.Background(), host)
	require.NoError(t, err, "failed to start extension")

	err = rcv.Start(context.Background(), host)
	require.NoError(t, err, "failed to start receiver")

	// Wait for the file to be processed
	// Run: "log raw-dump -a testdata/one-line.logarchive"
	// total log entries: 1
	require.Eventually(
		t,
		func() bool { return sink.LogRecordCount() > 0 },
		200*time.Second, 10*time.Millisecond,
	)

	// Verify the log content
	logs := sink.AllLogs()
	require.Len(t, logs, 5)

	logCounts := countLogInformation(logs)

	t.Logf("map: %#v", logCounts["byEventType"])
	t.Logf("map: %#v", logCounts["byLogType"])
	t.Logf("map: %#v", logCounts["byMessageMatches"])
	t.Logf("map: %#v", logCounts["byProcessMatches"])
	t.Logf("map: %#v", logCounts["bySubsystemMatches"])

	// require.Equal(t, 1, byEventType["signpostEvent"])

	// logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).EventName()
	// assert.Contains(t, logRecord.Body().AsString(), "Read traceV3 file")
	// assert.Contains(t, logRecord.Body().AsString(), testFile)

	err = rcv.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")

	err = ext.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown extension")
}
