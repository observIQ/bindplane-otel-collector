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
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"
	"github.com/observiq/bindplane-otel-collector/receiver/macosunifiedloggingreceiver/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

type mockHostForTest struct {
	extensions map[component.ID]component.Component
}

func (h *mockHostForTest) GetExtensions() map[component.ID]component.Component { return h.extensions }

// skipIfNoLogArchiveTestdata skips the test if the testdata directory doesn't exist
func skipIfNoLogArchiveTestdata(t *testing.T) {
	if _, err := os.Stat("testdata/logarchivetestdata"); os.IsNotExist(err) {
		t.Skip("Skipping test: testdata/logarchivetestdata directory not found")
	}
}

func TestParseLogOneLogFromBigSur(t *testing.T) {
	skipIfNoLogArchiveTestdata(t)
	filePaths := map[string]string{
		"tracev3":  filepath.Join("testdata", "logarchivetestdata", "system_logs_big_sur.logarchive", "**", "0000000000000004.tracev3"),
		"timesync": filepath.Join(""),
		"dsc":      filepath.Join(""),
		"uuidtext": filepath.Join(""),
	}
	sink := new(consumertest.LogsSink)

	setupAndStartReceiver(t, filePaths, sink, 1)

	// Verify the log content
	logCounts := countLogInformation(sink.AllLogs())
	logLogCounts(t, logCounts)

	require.Less(t, 1, sink.LogRecordCount())
	require.LessOrEqual(t, 0, logCounts["byEventType"]["Statedump"])
}

func TestParseLogAllBigSur(t *testing.T) {
	skipIfNoLogArchiveTestdata(t)
	filePaths := getFilePathsForLogArchiveInTestData("system_logs_big_sur.logarchive")
	sink := new(consumertest.LogsSink)

	setupAndStartReceiver(t, filePaths, sink, 1)

	// Verify the log content
	logCounts := countLogInformation(sink.AllLogs())
	logLogCounts(t, logCounts)

	require.Less(t, 1, sink.LogRecordCount())
	require.LessOrEqual(t, 0, logCounts["byEventType"]["Statedump"])
}

func TestParseAllLogsPrivateBigSur(t *testing.T) {
	skipIfNoLogArchiveTestdata(t)
	filePaths := getFilePathsForLogArchiveInTestData("system_logs_big_sur_private_enabled.logarchive")
	sink := new(consumertest.LogsSink)

	setupAndStartReceiver(t, filePaths, sink, 1)

	// Verify the log content
	logCounts := countLogInformation(sink.AllLogs())

	require.Less(t, 1, sink.LogRecordCount())
	require.LessOrEqual(t, 0, logCounts["byEventType"]["Statedump"])
}

func TestParseAllLogsPrivateWithPublicMixBigSur(t *testing.T) {
	skipIfNoLogArchiveTestdata(t)
	filePaths := getFilePathsForLogArchiveInTestData("system_logs_big_sur_public_private_data_mix.logarchive")
	sink := new(consumertest.LogsSink)

	setupAndStartReceiver(t, filePaths, sink, 1)

	// Verify the log content
	logCounts := countLogInformation(sink.AllLogs())

	require.Less(t, 1, sink.LogRecordCount())
	require.LessOrEqual(t, 0, logCounts["byEventType"]["Statedump"])
}

func TestParseAllLogsHighSierra(t *testing.T) {
	skipIfNoLogArchiveTestdata(t)
	filePaths := getFilePathsForLogArchiveInTestData("system_logs_high_sierra.logarchive")
	sink := new(consumertest.LogsSink)

	setupAndStartReceiver(t, filePaths, sink, 1)

	// Verify the log content
	logCounts := countLogInformation(sink.AllLogs())

	require.Less(t, 1, sink.LogRecordCount())
	require.LessOrEqual(t, 0, logCounts["byEventType"]["Statedump"])
}

func TestParseAllLogsMonterey(t *testing.T) {
	skipIfNoLogArchiveTestdata(t)
	filePaths := getFilePathsForLogArchiveInTestData("system_logs_monterey.logarchive")
	sink := new(consumertest.LogsSink)

	setupAndStartReceiver(t, filePaths, sink, 1)

	// Verify the log content
	logCounts := countLogInformation(sink.AllLogs())

	require.Less(t, 1, sink.LogRecordCount())
	require.LessOrEqual(t, 0, logCounts["byEventType"]["Statedump"])
}

func getFilePathsForLogArchiveInTestData(archivePath string) map[string]string {
	return map[string]string{
		"tracev3":  filepath.Join("testdata", "logarchivetestdata", archivePath, "**", "*.tracev3"),
		"timesync": filepath.Join("testdata", "logarchivetestdata", archivePath, "timesync", "*.timesync"),
		"dsc":      filepath.Join("testdata", "logarchivetestdata", archivePath, "dsc", "*"),
		"uuidtext": filepath.Join("testdata", "logarchivetestdata", archivePath, "[0-9A-F][0-9A-F]", "*"),
	}
}

func setupAndStartReceiver(t *testing.T, filePaths map[string]string, sink *consumertest.LogsSink, expectedLogCount int) {
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
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Encoding = "macosunifiedlogencoding"
	cfg.TraceV3Paths = []string{filePaths["tracev3"]}
	cfg.TimesyncPaths = []string{filePaths["timesync"]}
	cfg.DSCPaths = []string{filePaths["dsc"]}
	cfg.UUIDTextPaths = []string{filePaths["uuidtext"]}
	cfg.StartAt = "beginning"
	cfg.PollInterval = 100 * time.Millisecond

	rcv, err := factory.CreateLogs(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, rcv)

	// Start the Extension and the Receiver for the test
	err = ext.Start(context.Background(), host)
	require.NoError(t, err, "failed to start extension")

	// Test that we can start and stop the receiver
	err = rcv.Start(context.Background(), host)
	require.NoError(t, err, "failed to start receiver")

	require.Eventually(
		t,
		func() bool { return sink.LogRecordCount() >= expectedLogCount },
		30*time.Second, 10*time.Millisecond,
	)

	err = rcv.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")

	err = ext.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown extension")

	return
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

func logLogCounts(t *testing.T, logCounts map[string]map[string]int) {
	t.Helper()

	// Sort outer categories for stable output
	categories := make([]string, 0, len(logCounts))
	for cat := range logCounts {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	for _, cat := range categories {
		t.Logf("== %s ==", cat)
		inner := logCounts[cat]

		// Sort inner keys for stable output
		keys := make([]string, 0, len(inner))
		for k := range inner {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		if len(keys) == 0 {
			t.Logf("(empty)")
			continue
		}
		for _, k := range keys {
			t.Logf("%s: %d", k, inner[k])
		}
	}
}
