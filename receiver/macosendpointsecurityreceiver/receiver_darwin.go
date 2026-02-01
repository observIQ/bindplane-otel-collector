// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package macosendpointsecurityreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/macosendpointsecurityreceiver"

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// endpointSecurityReceiver uses exec.Command to run the native macOS `eslogger` command
type endpointSecurityReceiver struct {
	config              *Config
	logger              *zap.Logger
	consumer            consumer.Logs
	cancel              context.CancelFunc
	eventParserRegistry EventParserRegistry
}

func newEndpointSecurityReceiver(
	config *Config,
	logger *zap.Logger,
	consumer consumer.Logs,
) *endpointSecurityReceiver {
	return &endpointSecurityReceiver{
		config:              config,
		logger:              logger,
		consumer:            consumer,
		eventParserRegistry: NewEventParserRegistry(),
	}
}

func (r *endpointSecurityReceiver) Start(ctx context.Context, _ component.Host) error {
	r.logger.Info("Starting macOS Endpoint Security receiver")

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	// Start reading events in a goroutine
	go r.readEvents(ctx)

	return nil
}

func (r *endpointSecurityReceiver) Shutdown(_ context.Context) error {
	r.logger.Info("Shutting down macOS Endpoint Security receiver")
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

// readEvents streams events from eslogger continuously
func (r *endpointSecurityReceiver) readEvents(ctx context.Context) {
	// eslogger streams continuously, so we just run it once
	// If it fails, we can retry with exponential backoff
	minInterval := 100 * time.Millisecond
	maxInterval := r.config.MaxPollInterval
	currentInterval := time.Duration(0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(currentInterval):
		}

		err := r.runEsloggerCommand(ctx)
		if err != nil {
			r.logger.Error("Failed to run eslogger command", zap.Error(err))
			// Exponential backoff for retry
			if currentInterval == 0 {
				currentInterval = minInterval
			} else {
				nextInterval := currentInterval * 2
				if nextInterval > maxInterval {
					currentInterval = maxInterval
				} else {
					currentInterval = nextInterval
				}
			}
			r.logger.Debug("Retrying eslogger command", zap.Duration("interval", currentInterval))
		} else {
			// If command exited normally (shouldn't happen for streaming), exit
			r.logger.Info("eslogger command exited")
			return
		}
	}
}

// runEsloggerCommand executes the eslogger command and processes output
func (r *endpointSecurityReceiver) runEsloggerCommand(ctx context.Context) error {
	// Build the eslogger command arguments
	args := r.buildEsloggerCommandArgs()

	r.logger.Info("Running eslogger command", zap.Strings("args", args))

	// Create the command
	cmd := exec.CommandContext(ctx, "eslogger", args...) // #nosec G204 - args are controlled by config

	// Get stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start eslogger command: %w", err)
	}

	// Read stderr in a goroutine to capture any error messages
	var stderrErr error
	var stderrOutput []byte
	go func() {
		stderrOutput, stderrErr = readStderr(stderr)
		if stderrErr == nil && len(stderrOutput) > 0 {
			r.logger.Warn("eslogger stderr output", zap.ByteString("stderr", stderrOutput))
		}
	}()

	// Ensure stderr is closed when we're done
	defer stderr.Close()

	// Read and process output line by line
	scanner := bufio.NewScanner(stdout)
	// Set a large buffer size for long JSON lines
	buf := make([]byte, 0, 1024*1024) // 1MB buffer
	scanner.Buffer(buf, 10*1024*1024) // 10MB max

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			err := cmd.Process.Kill()
			if err != nil {
				r.logger.Error("Failed to kill eslogger command", zap.Error(err))
			}
			return ctx.Err()
		default:
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			// Parse and send the event
			if err := r.processEventLine(ctx, line); err != nil {
				r.logger.Warn("Failed to process event line",
					zap.Error(err))
				continue
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading eslogger output: %w", err)
	}

	// Scanner finished means EOF was reached, which means eslogger exited
	// This is unexpected for a streaming command - it should run continuously
	// Wait for the process to finish and check the exit status
	waitErr := cmd.Wait()
	if waitErr != nil {
		return fmt.Errorf("eslogger exited with error: %w (stderr: %s)", waitErr, string(stderrOutput))
	}

	// If eslogger exited successfully (exit code 0), that's unexpected for streaming
	if len(stderrOutput) > 0 {
		return fmt.Errorf("eslogger exited unexpectedly (stderr: %s)", string(stderrOutput))
	}

	return fmt.Errorf("eslogger exited unexpectedly (no error output)")
}

// readStderr reads all data from stderr
func readStderr(stderr io.ReadCloser) ([]byte, error) {
	defer stderr.Close()
	return io.ReadAll(stderr)
}

// buildEsloggerCommandArgs constructs the arguments for the eslogger command
func (r *endpointSecurityReceiver) buildEsloggerCommandArgs() []string {
	args := []string{}

	// Format is always JSON (default), no need to specify

	// Add --select flags for each path
	for _, path := range r.config.SelectPaths {
		args = append(args, "--select", path)
	}

	// Convert EventType enums to strings for command arguments
	eventTypeSet := NewEventTypeSet(r.config.EventTypes)
	args = append(args, eventTypeSet.ToSlice()...)

	return args
}

// processEventLine processes an event line and sends it to the consumer
func (r *endpointSecurityReceiver) processEventLine(ctx context.Context, line []byte) error {
	// Parse JSON to determine event type
	var baseEvent struct {
		Event struct {
			Type EventType `json:"type"`
		} `json:"event"`
	}

	if err := json.Unmarshal(line, &baseEvent); err != nil {
		return fmt.Errorf("failed to parse event type: %w", err)
	}

	// Get type-safe parser from registry
	parser := r.eventParserRegistry.GetParser(baseEvent.Event.Type)

	// Parse event using type-safe parser
	event, err := parser.ParseEvent(line)
	if err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	// Convert to OTel log record
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()

	// Use type-safe conversion
	event.ToLogRecord(logRecord)

	// Send to consumer
	return r.consumer.ConsumeLogs(ctx, logs)
}
