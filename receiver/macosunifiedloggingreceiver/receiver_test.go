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
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestBuildLogCommandArgs(t *testing.T) {
	t.Run("with ndjson style (default)", func(t *testing.T) {
		receiver := &unifiedLoggingReceiver{
			config: &Config{
				ArchivePath: "./testdata/system_logs.logarchive",
				StartTime:   "2024-01-01 00:00:00",
				EndTime:     "2024-01-02 00:00:00",
				Predicate:   "subsystem == 'com.apple.systempreferences'",
				Raw:         false,
			},
		}

		args := receiver.buildLogCommandArgs()
		require.Contains(t, args, "--archive")
		require.Contains(t, args, "./testdata/system_logs.logarchive")
		require.Contains(t, args, "--start")
		require.Contains(t, args, "2024-01-01 00:00:00")
		require.Contains(t, args, "--end")
		require.Contains(t, args, "2024-01-02 00:00:00")
		require.Contains(t, args, "--predicate")
		require.Contains(t, args, "subsystem == 'com.apple.systempreferences'")
		require.Contains(t, args, "--style")
		require.Contains(t, args, "ndjson")
	})

	t.Run("with raw flag enabled", func(t *testing.T) {
		receiver := &unifiedLoggingReceiver{
			config: &Config{
				ArchivePath: "./testdata/system_logs.logarchive",
				StartTime:   "2024-01-01 00:00:00",
				Predicate:   "subsystem == 'com.apple.systempreferences'",
				Raw:         true,
			},
		}

		args := receiver.buildLogCommandArgs()
		require.Contains(t, args, "--archive")
		require.Contains(t, args, "./testdata/system_logs.logarchive")
		require.Contains(t, args, "--start")
		require.Contains(t, args, "2024-01-01 00:00:00")
		require.Contains(t, args, "--predicate")
		require.Contains(t, args, "subsystem == 'com.apple.systempreferences'")
		// Should NOT contain --style ndjson when raw mode is enabled
		require.NotContains(t, args, "--style")
		require.NotContains(t, args, "ndjson")
	})
}

func TestMapMessageTypeToSeverity(t *testing.T) {
	tests := []struct {
		name     string
		msgType  string
		expected plog.SeverityNumber
	}{
		{
			name:     "Error message type",
			msgType:  "Error",
			expected: plog.SeverityNumberError,
		},
		{
			name:     "Fault message type",
			msgType:  "Fault",
			expected: plog.SeverityNumberFatal,
		},
		{
			name:     "Default message type",
			msgType:  "Default",
			expected: plog.SeverityNumberInfo,
		},
		{
			name:     "Info message type",
			msgType:  "Info",
			expected: plog.SeverityNumberInfo,
		},
		{
			name:     "Debug message type",
			msgType:  "Debug",
			expected: plog.SeverityNumberDebug,
		},
		{
			name:     "Unknown message type",
			msgType:  "Unknown",
			expected: plog.SeverityNumberUnspecified,
		},
		{
			name:     "Empty message type",
			msgType:  "",
			expected: plog.SeverityNumberUnspecified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapMessageTypeToSeverity(tt.msgType)
			if result != tt.expected {
				t.Errorf("mapMessageTypeToSeverity(%q) = %v, want %v", tt.msgType, result, tt.expected)
			}
		})
	}
}

func TestIsCompletionLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "JSON completion format",
			line:     `{"count":540659,"finished":1}`,
			expected: true,
		},
		{
			name:     "JSON completion format with whitespace",
			line:     `  {"count":100,"finished":1}  `,
			expected: true,
		},
		{
			name:     "completion line with asterisks",
			line:     "** Processed 574 entries, done. **",
			expected: true,
		},
		{
			name:     "completion line with whitespace",
			line:     "  ** Finished processing **  ",
			expected: true,
		},
		{
			name:     "completion line with Processed and entries",
			line:     "Processed 100 entries successfully",
			expected: true,
		},
		{
			name:     "completion line with Processed and done",
			line:     "Processed all logs, done",
			expected: true,
		},
		{
			name:     "normal log line",
			line:     "2024-01-01 12:00:00.123456-0700  localhost kernel[0]: System initialized",
			expected: false,
		},
		{
			name:     "log line containing Processed word only",
			line:     "2024-01-01 12:00:00.123456-0700  localhost app[123]: Processed user request",
			expected: false,
		},
		{
			name:     "JSON without count and finished",
			line:     `{"timestamp":"2024-01-01","message":"test"}`,
			expected: false,
		},
		{
			name:     "empty line",
			line:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCompletionLine([]byte(tt.line))
			require.Equal(t, tt.expected, result)
		})
	}
}
