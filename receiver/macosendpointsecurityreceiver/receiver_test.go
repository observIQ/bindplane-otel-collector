// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package macosendpointsecurityreceiver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap"
)

func setupFakeEsloggerBinary(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "eslogger")
	script := `#!/bin/sh
set -e
if [ -n "$FAKE_ESLOGGER_CALLS_FILE" ]; then
  cmd=""
  if [ "$#" -gt 0 ]; then
    for arg in "$@"
    do
      cmd="$cmd $arg"
    done
  fi
  printf "%s\n" "$cmd" >> "$FAKE_ESLOGGER_CALLS_FILE"
fi

if [ -n "$FAKE_ESLOGGER_STREAM_LINE" ]; then
  while true
  do
    printf "%s\n" "$FAKE_ESLOGGER_STREAM_LINE"
    if [ -n "$FAKE_ESLOGGER_STREAM_DELAY" ]; then
      sleep "$FAKE_ESLOGGER_STREAM_DELAY"
    fi
  done
fi

if [ -n "$FAKE_ESLOGGER_OUTPUT_PATH" ] && [ -f "$FAKE_ESLOGGER_OUTPUT_PATH" ]; then
  cat "$FAKE_ESLOGGER_OUTPUT_PATH"
fi
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0o755)) //nolint:gosec // script needs to be executable
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
}

func writeFakeEsloggerOutput(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "eslogger_output.txt")
	content := strings.Join(lines, "\n") + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func readRecordedCalls(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func TestBuildEsloggerCommandArgs(t *testing.T) {
	t.Run("with event types and select paths", func(t *testing.T) {
		receiver := &endpointSecurityReceiver{
			config: &Config{
				EventTypes:  []EventType{EventTypeExec, EventTypeFork, EventTypeExit},
				SelectPaths: []string{"/bin/zsh", "/usr/bin"},
			},
		}

		args := receiver.buildEsloggerCommandArgs()
		require.Contains(t, args, "--select")
		require.Contains(t, args, "/bin/zsh")
		require.Contains(t, args, "/usr/bin")
		require.Contains(t, args, "exec")
		require.Contains(t, args, "fork")
		require.Contains(t, args, "exit")
	})

	t.Run("with event types only", func(t *testing.T) {
		receiver := &endpointSecurityReceiver{
			config: &Config{
				EventTypes: []EventType{EventTypeOpen, EventTypeClose},
			},
		}

		args := receiver.buildEsloggerCommandArgs()
		require.Contains(t, args, "open")
		require.Contains(t, args, "close")
		require.NotContains(t, args, "--select")
	})
}

func TestProcessEventLine(t *testing.T) {
	t.Run("processes exec event", func(t *testing.T) {
		sink := &consumertest.LogsSink{}
		receiver := &endpointSecurityReceiver{
			config:              &Config{},
			consumer:            sink,
			logger:              zap.NewNop(),
			eventParserRegistry: NewEventParserRegistry(),
		}

		jsonLine := []byte(`{"event":{"type":"exec"},"process":{"executable":{"path":"/bin/zsh"},"audit_token":{"pid":1234}},"timestamp":"2024-01-01T12:00:00Z"}`)
		err := receiver.processEventLine(t.Context(), jsonLine)
		require.NoError(t, err)

		// Verify the log was consumed
		require.Len(t, sink.AllLogs(), 1)
		logs := sink.AllLogs()[0]
		require.Equal(t, 1, logs.LogRecordCount())

		// Verify the log record contains the JSON as body
		logRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
		require.Equal(t, string(jsonLine), logRecord.Body().Str())
		require.NotZero(t, logRecord.Timestamp())
	})

	t.Run("handles invalid json gracefully", func(t *testing.T) {
		sink := &consumertest.LogsSink{}
		receiver := &endpointSecurityReceiver{
			config:              &Config{},
			consumer:            sink,
			logger:              zap.NewNop(),
			eventParserRegistry: NewEventParserRegistry(),
		}

		invalidJSON := []byte(`{invalid json}`)
		err := receiver.processEventLine(t.Context(), invalidJSON)
		require.Error(t, err) // Should error on invalid JSON
	})
}

func TestRunEsloggerCommandSkipsEmptyLines(t *testing.T) {
	setupFakeEsloggerBinary(t)
	outputPath := writeFakeEsloggerOutput(t,
		`{"event":{"type":"exec"},"process":{"executable":{"path":"/bin/zsh"}},"timestamp":"2024-01-01T12:00:00Z"}`,
		"",
		`{"event":{"type":"fork"},"process":{"executable":{"path":"/bin/zsh"}},"timestamp":"2024-01-01T12:00:01Z"}`,
	)
	t.Setenv("FAKE_ESLOGGER_OUTPUT_PATH", outputPath)

	sink := &consumertest.LogsSink{}
	receiver := newEndpointSecurityReceiver(&Config{EventTypes: []EventType{EventTypeExec}}, zap.NewNop(), sink)

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	err := receiver.runEsloggerCommand(ctx)
	require.NoError(t, err)

	allLogs := sink.AllLogs()
	require.GreaterOrEqual(t, len(allLogs), 1)
}

func TestReadEventsRespectsContextCancellation(t *testing.T) {
	setupFakeEsloggerBinary(t)
	t.Setenv("FAKE_ESLOGGER_STREAM_LINE", `{"event":{"type":"exec"},"process":{"executable":{"path":"/bin/zsh"}},"timestamp":"2024-01-01T12:00:00Z"}`)
	t.Setenv("FAKE_ESLOGGER_STREAM_DELAY", "0.01")

	sink := &consumertest.LogsSink{}
	receiver := newEndpointSecurityReceiver(&Config{EventTypes: []EventType{EventTypeExec}}, zap.NewNop(), sink)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	receiver.readEvents(ctx)

	// Should have processed some events before cancellation
	allLogs := sink.AllLogs()
	require.Greater(t, len(allLogs), 0)
}

func TestEventTypeSet(t *testing.T) {
	set := NewEventTypeSet([]EventType{EventTypeExec, EventTypeFork, EventTypeExit})

	require.True(t, set.Contains(EventTypeExec))
	require.True(t, set.Contains(EventTypeFork))
	require.True(t, set.Contains(EventTypeExit))
	require.False(t, set.Contains(EventTypeOpen))

	slice := set.ToSlice()
	require.Len(t, slice, 3)
	require.Contains(t, slice, "exec")
	require.Contains(t, slice, "fork")
	require.Contains(t, slice, "exit")
}
