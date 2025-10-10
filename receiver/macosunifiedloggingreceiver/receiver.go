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
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := r.runLogCommand(ctx); err != nil {
				r.logger.Error("Failed to run log command", zap.Error(err))
				// Wait before retrying
				select {
				case <-time.After(r.config.PollInterval):
					continue
				case <-ctx.Done():
					return
				}
			}

			// If reading from archive, we're done
			if r.config.ArchivePath != "" {
				r.logger.Info("Finished reading archive logs")
				return
			}

			// For live mode, wait before polling again
			select {
			case <-time.After(r.config.PollInterval):
			case <-ctx.Done():
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

	// Read and process output line by line
	scanner := bufio.NewScanner(stdout)
	// Set a large buffer size for long log lines
	buf := make([]byte, 0, 1024*1024) // 1MB buffer
	scanner.Buffer(buf, 10*1024*1024) // 10MB max

	var processedCount int
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return ctx.Err()
		default:
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			// Parse and send the log entry
			if err := r.processLogLine(ctx, line); err != nil {
				r.logger.Warn("Failed to process log line",
					zap.Error(err),
					zap.String("line", string(line[:min(len(line), 200)])))
				continue
			}
			processedCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log output: %w", err)
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("log command failed: %w", err)
	}

	r.logger.Info("Processed logs", zap.Int("count", processedCount))
	return nil
}

// buildLogCommandArgs constructs the arguments for the log command
func (r *unifiedLoggingReceiver) buildLogCommandArgs() []string {
	args := []string{"show"}

	// Add archive path if specified
	if r.config.ArchivePath != "" {
		args = append(args, "--archive", r.config.ArchivePath)
	}

	// Add style for NDJSON output
	args = append(args, "--style", "ndjson")

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
	// Parse the NDJSON line
	var logEntry map[string]interface{}
	if err := json.Unmarshal(line, &logEntry); err != nil {
		return fmt.Errorf("failed to parse NDJSON: %w", err)
	}

	// Convert to OTel plog
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()

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
