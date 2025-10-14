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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// logCommandReceiver uses exec.Command to run the native macOS `log` command
type unifiedLoggingReceiver struct {
	config   *Config
	logger   *zap.Logger
	consumer consumer.Logs
	cancel   context.CancelFunc
}

func newReceiver(
	config *Config,
	logger *zap.Logger,
	consumer consumer.Logs,
) *unifiedLoggingReceiver {
	return &unifiedLoggingReceiver{
		config:   config,
		logger:   logger,
		consumer: consumer,
	}
}

func (r *unifiedLoggingReceiver) Start(ctx context.Context, _ component.Host) error {
	r.logger.Info("Starting macOS unified logging receiver")

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	// Start reading logs in a goroutine
	go r.readLogs(ctx)

	return nil
}

func (r *unifiedLoggingReceiver) Shutdown(_ context.Context) error {
	r.logger.Info("Shutting down macOS unified logging receiver")
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

// readLogs runs the log command and processes output
func (r *unifiedLoggingReceiver) readLogs(ctx context.Context) {
	ticker := time.NewTicker(r.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.runLogCommand(ctx); err != nil {
				r.logger.Error("Failed to run log command", zap.Error(err))
				continue
			}

			if r.config.ArchivePath != "" {
				r.logger.Info("Finished reading archive logs")
				return
			}
		}
	}
}

// runLogCommand executes the log command and processes output
func (r *unifiedLoggingReceiver) runLogCommand(ctx context.Context) error {
	// Build the log command arguments
	args := r.buildLogCommandArgs()

	r.logger.Info("Running log command", zap.Strings("args", args))

	// Create the command
	cmd := exec.CommandContext(ctx, "log", args...) // #nosec G204 - args are controlled by config

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start log command: %w", err)
	}

	// Ensure the process is properly cleaned up to avoid zombies
	defer func() {
		_ = cmd.Wait()
	}()

	// Read and process output line by line
	scanner := bufio.NewScanner(stdout)
	// Set a large buffer size for long log lines
	buf := make([]byte, 0, 1024*1024) // 1MB buffer
	scanner.Buffer(buf, 10*1024*1024) // 10MB max

	var processedCount int
	var isFirstLine = true
	var killed bool
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			killed = true
			return ctx.Err()
		default:
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			// Skip the header line in raw mode
			if r.config.Raw && isFirstLine {
				isFirstLine = false
				continue
			}
			isFirstLine = false

			// Skip completion/status messages in raw mode
			if r.config.Raw && isCompletionLine(line) {
				continue
			}

			// Parse and send the log entry
			if err := r.processLogLine(ctx, line); err != nil {
				r.logger.Warn("Failed to process log line",
					zap.Error(err))
				continue
			}
			processedCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log output: %w", err)
	}

	// Check if the process was killed due to context cancellation
	if !killed {
		r.logger.Info("Processed logs", zap.Int("count", processedCount))
	}
	return nil
}

// buildLogCommandArgs constructs the arguments for the log command
func (r *unifiedLoggingReceiver) buildLogCommandArgs() []string {
	args := []string{"show"}

	// Add archive path if specified
	if r.config.ArchivePath != "" {
		args = append(args, "--archive", r.config.ArchivePath)
	}

	// Add style for NDJSON output (only when not in raw mode)
	if !r.config.Raw {
		args = append(args, "--style", "ndjson")
	}

	// Add start time
	if r.config.StartTime != "" {
		args = append(args, "--start", r.config.StartTime)
	} else if r.config.MaxLogAge > 0 && r.config.ArchivePath == "" {
		// For live mode, calculate start time from max_log_age
		startTime := time.Now().Add(-r.config.MaxLogAge)
		args = append(args, "--start", startTime.Format("2006-01-02 15:04:05"))
	}

	// Add end time (archive mode only)
	if r.config.EndTime != "" && r.config.ArchivePath != "" {
		args = append(args, "--end", r.config.EndTime)
	}

	// Add predicate filter
	if r.config.Predicate != "" {
		args = append(args, "--predicate", r.config.Predicate)
	}

	return args
}

// processLogLine parses a single NDJSON log line and sends it to the consumer
func (r *unifiedLoggingReceiver) processLogLine(ctx context.Context, line []byte) error {
	// Convert to OTel plog
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()

	// Handle raw mode - send line as-is
	if r.config.Raw {
		logRecord.Body().SetStr(string(line))
		logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		return r.consumer.ConsumeLogs(ctx, logs)
	}

	// Parse the NDJSON line
	var logEntry map[string]interface{}
	if err := json.Unmarshal(line, &logEntry); err != nil {
		return fmt.Errorf("failed to parse NDJSON: %w", err)
	}

	// Set the log body from the message field
	if msg, ok := logEntry["eventMessage"].(string); ok {
		logRecord.Body().SetStr(msg)
	}

	// Parse and set timestamp
	if ts, ok := logEntry["timestamp"].(string); ok {
		if t, err := time.Parse("2006-01-02 15:04:05.000000-0700", ts); err == nil {
			logRecord.SetTimestamp(pcommon.NewTimestampFromTime(t))
		}
	}
	logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// Set severity from messageType
	if msgType, ok := logEntry["messageType"].(string); ok {
		logRecord.SetSeverityText(msgType)
		logRecord.SetSeverityNumber(mapMessageTypeToSeverity(msgType))
	}

	// Add all fields as attributes
	attrs := logRecord.Attributes()
	for key, value := range logEntry {
		// Skip the message since we put it in the body
		if key == "eventMessage" {
			continue
		}

		// Convert value to appropriate pcommon type
		switch v := value.(type) {
		case string:
			attrs.PutStr(key, v)
		case float64:
			attrs.PutDouble(key, v)
		case bool:
			attrs.PutBool(key, v)
		case int64:
			attrs.PutInt(key, v)
		default:
			// Convert complex types to JSON string
			if jsonBytes, err := json.Marshal(v); err == nil {
				attrs.PutStr(key, string(jsonBytes))
			}
		}
	}

	// Send to consumer
	return r.consumer.ConsumeLogs(ctx, logs)
}

// mapMessageTypeToSeverity maps log messageType to OTel severity
func mapMessageTypeToSeverity(msgType string) plog.SeverityNumber {
	switch msgType {
	case "Error":
		return plog.SeverityNumberError
	case "Fault":
		return plog.SeverityNumberFatal
	case "Default", "Info":
		return plog.SeverityNumberInfo
	case "Debug":
		return plog.SeverityNumberDebug
	default:
		return plog.SeverityNumberUnspecified
	}
}

// isCompletionLine checks if a line is a completion/status message from the log command
// These lines should be filtered out (e.g., {"count":540659,"finished":1})
func isCompletionLine(line []byte) bool {
	// Trim whitespace
	trimmed := bytes.TrimSpace(line)

	// Check if line is empty
	if len(trimmed) == 0 {
		return false
	}

	// Check if line starts with "**" (typical completion message format)
	if bytes.HasPrefix(trimmed, []byte("**")) {
		return true
	}

	// Check for JSON completion format: {"count":N,"finished":1}
	if bytes.HasPrefix(trimmed, []byte("{")) && bytes.HasSuffix(trimmed, []byte("}")) {
		// Quick check for both "count" and "finished" fields
		if bytes.Contains(trimmed, []byte("\"count\"")) &&
			bytes.Contains(trimmed, []byte("\"finished\"")) {
			return true
		}
	}

	// Check for common completion keywords
	if bytes.Contains(trimmed, []byte("Processed")) &&
		(bytes.Contains(trimmed, []byte("entries")) || bytes.Contains(trimmed, []byte("done"))) {
		return true
	}

	return false
}
