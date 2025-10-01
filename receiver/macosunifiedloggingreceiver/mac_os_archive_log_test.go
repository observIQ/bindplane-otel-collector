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
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pdata/plog"
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
	filePath := filepath.Join("testdata", "system_logs_big_sur.logarchive")
	sink := new(consumertest.LogsSink)
	var macOSLogReceiver receiver.Logs
	var err error

	macOSLogReceiver, err = setupAndStartReceiver(t, filePath, sink)
	require.NoError(t, err, "failed to setup and start receiver")

	// Verify the log content
	logCounts := countLogInformation(sink.AllLogs())

	// require.Equal(t, 50665, logCounts["signpostEvent"])
	// require.Equal(t, 5, logCounts["lossEvent"])

	// logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).EventName()
	// assert.Contains(t, logRecord.Body().AsString(), "Read traceV3 file")
	// assert.Contains(t, logRecord.Body().AsString(), testFile)

	err = macOSLogReceiver.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}

func TestParseAllLogsPrivateBigSur(t *testing.T) {
	filePath := filepath.Join("testdata", "system_logs_big_sur_private_enabled.logarchive")
	sink := new(consumertest.LogsSink)
	var macOSLogReceiver receiver.Logs
	var err error

	macOSLogReceiver, err = setupAndStartReceiver(t, filePath, sink)
	require.NoError(t, err, "failed to setup and start receiver")

	// Verify the log content
	logCounts := countLogInformation(sink.AllLogs())

	// require.Equal(t, 50665, logCounts["byEventType"]["signpostEvent"])
	// require.Equal(t, 5, logCounts["byEventType"]["lossEvent"])

	// logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).EventName()
	// assert.Contains(t, logRecord.Body().AsString(), "Read traceV3 file")
	// assert.Contains(t, logRecord.Body().AsString(), testFile)

	err = macOSLogReceiver.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}

func TestParseAllLogsPrivateWithPublicMixBigSur(t *testing.T) {
	filePath := filepath.Join("testdata", "system_logs_big_sur_public_private_data_mix.logarchive")
	sink := new(consumertest.LogsSink)
	var macOSLogReceiver receiver.Logs
	var err error

	macOSLogReceiver, err = setupAndStartReceiver(t, filePath, sink)
	require.NoError(t, err, "failed to setup and start receiver")

	// Verify the log content
	logCounts := countLogInformation(sink.AllLogs())

	// require.Equal(t, 50665, logCounts["byEventType"]["signpostEvent"])
	// require.Equal(t, 5, logCounts["byEventType"]["lossEvent"])

	err = macOSLogReceiver.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}

func TestBigSurMissingOversizeStrings(t *testing.T) {
	filePath := filepath.Join("testdata", "system_logs_big_sur.logarchive")
	sink := new(consumertest.LogsSink)
	var macOSLogReceiver receiver.Logs
	var err error

	macOSLogReceiver, err = setupAndStartReceiver(t, filePath, sink)
	require.NoError(t, err, "failed to setup and start receiver")

	// Verify the log content
	logCounts := countLogInformation(sink.AllLogs())

	// require.Equal(t, 50665, logCounts["signpostEvent"])
	// require.Equal(t, 5, logCounts["lossEvent"])

	err = macOSLogReceiver.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}

func TestParseAllLogsHighSierra(t *testing.T) {
	filePath := filepath.Join("testdata", "system_logs_high_sierra.logarchive")
	sink := new(consumertest.LogsSink)
	var macOSLogReceiver receiver.Logs
	var err error

	macOSLogReceiver, err = setupAndStartReceiver(t, filePath, sink)
	require.NoError(t, err, "failed to setup and start receiver")

	// Verify the log content
	byEventType := countLogInformation(sink.AllLogs())

	t.Logf("map: %#v", byEventType)

	require.Equal(t, 50665, byEventType["signpostEvent"])
	require.Equal(t, 5, byEventType["lossEvent"])

	err = macOSLogReceiver.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}

func TestParseAllLogsMonterey(t *testing.T) {
	filePath := filepath.Join("testdata", "system_logs_monterey.logarchive")
	sink := new(consumertest.LogsSink)
	var macOSLogReceiver receiver.Logs
	var err error

	macOSLogReceiver, err = setupAndStartReceiver(t, filePath, sink)
	require.NoError(t, err, "failed to setup and start receiver")

	// Verify the log content
	logCounts := countLogInformation(sink.AllLogs())

	require.Equal(t, 50665, logCounts["byEventType"]["signpostEvent"])
	require.Equal(t, 5, logCounts["byEventType"]["lossEvent"])

	err = macOSLogReceiver.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}

func setupAndStartReceiver(t *testing.T, glob string, sink *consumertest.LogsSink) (receiver.Logs, error) {
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
		func() bool { return sink.LogRecordCount() >= 1 },
		5*time.Second, 10*time.Millisecond,
	)
	return rcv, err
}

func countLogInformation(logs []plog.Logs) map[string]map[string]int {
	logCounts := map[string]map[string]int{
		"byEventType":    map[string]int{},
		"byLogType":      map[string]int{},
		"byMessage":      map[string]int{},
		"byProcess":      map[string]int{},
		"bySubsystem":    map[string]int{},
		"byCategory":     map[string]int{},
		"byActivityType": map[string]int{},
	}

	for _, log := range logs {
		rls := log.ResourceLogs()
		for i := 0; i < rls.Len(); i++ {
			sls := rls.At(i).ScopeLogs()
			for j := 0; j < sls.Len(); j++ {
				lrs := sls.At(j).LogRecords()
				for k := 0; k < lrs.Len(); k++ {
					lr := lrs.At(k)
					// Count Log Types
					if v, ok := lr.Attributes().Get("log_type"); ok {
						logCounts["byLogType"][v.AsString()]++
					}
					// Count Event Types
					if v, ok := lr.Attributes().Get("event_type"); ok {
						logCounts["byEventType"][v.AsString()]++
					}
					// Count Categories
					if v, ok := lr.Attributes().Get("category"); ok {
						logCounts["byCategory"][v.AsString()]++
					}
					// Count Activity Types
					if v, ok := lr.Attributes().Get("activity_type"); ok {
						logCounts["byActivityType"][v.AsString()]++
					}

					// Messages - Match a pattern in the message to count similar types of messages
					if v, ok := lr.Attributes().Get("message"); ok {
						msg := v.AsString()
						switch {
						case strings.TrimSpace(msg) == "":
							logCounts["byMessageMatches"]["emptyMessage"]++
						case strings.Contains(msg, "<Missing message data>"):
							logCounts["byMessageMatches"]["missingStrings"]++
						case strings.Contains(msg, "user: -1 <not found>"):
							logCounts["byMessageMatches"]["userNotFound"]++
						case strings.Contains(msg, "refreshing: details, reason: expired, user: mobile <not found>"):
							logCounts["byMessageMatches"]["mobileNotFound"]++
						case strings.Contains(msg, "BSSID 00:00:00:00:00:00"):
							logCounts["byMessageMatches"]["bssidCount"]++
						case strings.Contains(msg, "https://doh.dns.apple.com/dns-query"):
							logCounts["byMessageMatches"]["dnsQueryCount"]++
						case strings.Contains(msg, "bankofamerica"):
							logCounts["byMessageMatches"]["bofaCount"]++
						case strings.Contains(msg, "<not found>"):
							logCounts["byMessageMatches"]["notFound"]++
						case strings.Contains(msg, "group: staff@/Local/Default"):
							logCounts["byMessageMatches"]["staffCount"]++
						}
					}

					// Processes - Match a pattern in the process to count similar types of processes
					re := regexp.MustCompile(`your-process-pattern`)
					if v, ok := lr.Attributes().Get("process"); ok && re.MatchString(v.AsString()) {
						// e.g., increment a counter or take action on match
						logCounts["byProcessMatches"]["processName"]++
					}

					// Subsystems - Match a pattern in the subsystem to count similar types of subsystems
					re = regexp.MustCompile(`your-process-pattern`)
					if v, ok := lr.Attributes().Get("subsystem"); ok && re.MatchString(v.AsString()) {
						// e.g., increment a counter or take action on match
						logCounts["bySubsystemMatches"]["subsystemName"]++
					}
				}
			}
		}
	}
	return logCounts
}
