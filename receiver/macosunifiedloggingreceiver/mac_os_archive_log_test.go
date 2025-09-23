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
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"
	"github.com/observiq/bindplane-otel-collector/receiver/macosunifiedloggingreceiver/internal/metadata"
)

type mockHostForTest struct {
	extensions map[component.ID]component.Component
}

func (h *mockHostForTest) GetExtensions() map[component.ID]component.Component { return h.extensions }

func TestParseLogBigSur(t *testing.T) {
	filepath := filepath.Join("testdata", "system_logs_big_sur.logarchive")
	sink := new(consumertest.LogsSink)

	err, macOSLogReceiver := setupAndStartReceiver(t, filepath, sink)

	// Verify the log content
	byEventType := countLogInformation(t, sink)

	t.Logf("map: %#v", byEventType)

	require.Equal(t, 50665, byEventType["signpostEvent"])
	require.Equal(t, 5, byEventType["lossEvent"])

	// logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).EventName()
	// assert.Contains(t, logRecord.Body().AsString(), "Read traceV3 file")
	// assert.Contains(t, logRecord.Body().AsString(), testFile)

	err = macOSLogReceiver.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}

func TestParseAllLogsPrivateBigSur(t *testing.T) {
	filePath := filepath.Join("testdata", "system_logs_big_sur_private_enabled.logarchive")
	sink := new(consumertest.LogsSink)

	err, macOSLogReceiver := setupAndStartReceiver(t, filePath, sink)

	// Verify the log content
	byEventType := countLogInformation(t, sink)

	t.Logf("map: %#v", byEventType)

	require.Equal(t, 50665, byEventType["signpostEvent"])
	require.Equal(t, 5, byEventType["lossEvent"])

	// logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).EventName()
	// assert.Contains(t, logRecord.Body().AsString(), "Read traceV3 file")
	// assert.Contains(t, logRecord.Body().AsString(), testFile)

	err = macOSLogReceiver.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}

func TestParseAllLogsPrivateWithPublicMixBigSur(t *testing.T) {
	filePath := filepath.Join("testdata", "system_logs_big_sur_public_private_data_mix.logarchive")
	sink := new(consumertest.LogsSink)

	err, macOSLogReceiver := setupAndStartReceiver(t, filePath, sink)

	// Verify the log content
	byEventType := countLogInformation(t, sink)

	t.Logf("map: %#v", byEventType)

	require.Equal(t, 50665, byEventType["signpostEvent"])
	require.Equal(t, 5, byEventType["lossEvent"])

	err = macOSLogReceiver.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}

func TestBigSurMissingOversizeStrings(t *testing.T) {
	filePath := filepath.Join("testdata", "system_logs_big_sur.logarchive")
	sink := new(consumertest.LogsSink)

	err, macOSLogReceiver := setupAndStartReceiver(t, filePath, sink)

	// Verify the log content
	byEventType := countLogInformation(t, sink)

	t.Logf("map: %#v", byEventType)

	require.Equal(t, 50665, byEventType["signpostEvent"])
	require.Equal(t, 5, byEventType["lossEvent"])

	err = macOSLogReceiver.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}

func TestParseAllLogsHighSierra(t *testing.T) {
	filePath := filepath.Join("testdata", "system_logs_high_sierra.logarchive")
	sink := new(consumertest.LogsSink)

	err, macOSLogReceiver := setupAndStartReceiver(t, filePath, sink)

	// Verify the log content
	byEventType := countLogInformation(t, sink)

	t.Logf("map: %#v", byEventType)

	require.Equal(t, 50665, byEventType["signpostEvent"])
	require.Equal(t, 5, byEventType["lossEvent"])

	err = macOSLogReceiver.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}

func TestParseAllLogsMonterey(t *testing.T) {
	filePath := filepath.Join("testdata", "system_logs_monterey.logarchive")
	sink := new(consumertest.LogsSink)

	err, macOSLogReceiver := setupAndStartReceiver(t, filePath, sink)

	// Verify the log content
	byEventType := countLogInformation(t, sink)

	t.Logf("map: %#v", byEventType)

	require.Equal(t, 50665, byEventType["signpostEvent"])
	require.Equal(t, 5, byEventType["lossEvent"])

	err = macOSLogReceiver.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}

func countLogInformation(t *testing.T, sink *consumertest.LogsSink) map[string]int {
	byEventType := map[string]int{}

	logs := sink.AllLogs()
	require.Len(t, logs, 747616)

	rls := logs[0].ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		sls := rls.At(i).ScopeLogs()
		for j := 0; j < sls.Len(); j++ {
			lrs := sls.At(j).LogRecords()
			for k := 0; k < lrs.Len(); k++ {
				lr := lrs.At(k)

				// Event Types: logEvent, activityEvent, traceEvent, signpostEvent, lossEvent
				if v, ok := lr.Attributes().Get("eventType"); ok {
					switch v.AsString() {
					case "logEvent":
						byEventType["logEvent"]++
					case "activityEvent":
						byEventType["activityEvent"]++
					case "traceEvent":
						byEventType["traceEvent"]++
					case "signpostEvent":
						byEventType["signpostEvent"]++
					case "lossEvent":
						byEventType["lossEvent"]++
					}
				}
			}
		}
	}

	return byEventType
}

func setupAndStartReceiver(t *testing.T, glob string, sink *consumertest.LogsSink) (error, receiver.Logs) {
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

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Encoding = "macosunifiedlogencoding"
	cfg.TraceV3Paths = []string{glob}
	cfg.StartAt = "beginning"
	cfg.PollInterval = 100 * time.Millisecond

	rcv, err := factory.CreateLogs(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, rcv)

	// // Test that we can start and stop the receiver
	err = rcv.Start(context.Background(), host)
	require.NoError(t, err, "failed to start receiver")

	// Wait for the file to be processed
	// Run: "log raw-dump -a testdata/system_logs_big_sur.logarchive"
	// total log entries: 747,294
	// Add Statedump log entries: 322
	// Total log entries: 747,616
	require.Eventually(
		t,
		func() bool { return sink.LogRecordCount() >= 747616 },
		5*time.Second, 10*time.Millisecond,
	)
	return err, rcv
}
