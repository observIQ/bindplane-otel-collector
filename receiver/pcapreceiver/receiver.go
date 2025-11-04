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

package pcapreceiver

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/observiq/bindplane-otel-collector/receiver/pcapreceiver/capture"
	"github.com/observiq/bindplane-otel-collector/receiver/pcapreceiver/parser"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

var (
	// newCommand is used to allow tests to stub command creation
	newCommand = exec.Command
)

const defaultSnapLen = 65535

// pcapReceiver receives network packets via tcpdump and emits them as logs
type pcapReceiver struct {
	config   *Config
	logger   *zap.Logger
	consumer consumer.Logs
	cancel   context.CancelFunc
	cmd      *exec.Cmd
}

// newReceiver creates a new PCAP receiver
func newReceiver(config *Config, logger *zap.Logger, consumer consumer.Logs) *pcapReceiver {
	return &pcapReceiver{
		config:   config,
		logger:   logger,
		consumer: consumer,
	}
}

// Start starts the packet capture
func (r *pcapReceiver) Start(ctx context.Context, _ component.Host) error {
	r.logger.Info("Starting PCAP receiver", zap.String("interface", r.config.Interface))

	// Validate configuration first
	if err := r.config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	if err := r.checkPrivileges(); err != nil {
		r.logger.Warn("PCAP receiver cannot collect packets due to insufficient privileges",
			zap.Error(err),
			zap.String("message", "No packets will be collected. Please ensure the collector has the necessary privileges to capture network packets."))
		return nil
	}

	// Set default snaplen if not specified
	snaplen := r.config.SnapLen
	if snaplen == 0 {
		snaplen = defaultSnapLen
	}

	// Build the capture command
	// Use a single cross-platform builder to avoid symbol conflicts across build tags
	r.cmd = capture.BuildCaptureCommand(
		r.config.Interface,
		r.config.Filter,
		snaplen,
		r.config.Promiscuous,
	)

	if r.cmd == nil {
		return fmt.Errorf("failed to build capture command")
	}

	// Log the command being executed for debugging
	r.logger.Debug("Built capture command",
		zap.String("path", r.cmd.Path),
		zap.Strings("args", r.cmd.Args),
		zap.Int("snaplen", snaplen),
		zap.Bool("promiscuous", r.config.Promiscuous),
		zap.String("filter", r.config.Filter))

	// Get stdout pipe
	stdout, err := r.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Get stderr pipe for error messages
	stderr, err := r.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the command
	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start capture command: %w. Ensure tcpdump is installed and you have sufficient privileges", err)
	}

	r.logger.Debug("tcpdump process started", zap.Int("pid", r.cmd.Process.Pid))

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	// Start goroutine to read stderr
	go r.readStderr(ctx, stderr)

	// Start goroutine to read and parse packets
	go r.readPackets(ctx, stdout)

	r.logger.Info("PCAP receiver started successfully",
		zap.String("tcpdump_command", fmt.Sprintf("%s %v", r.cmd.Path, r.cmd.Args)),
		zap.Int("tcpdump_pid", r.cmd.Process.Pid))
	return nil
}

// Shutdown stops the packet capture
func (r *pcapReceiver) Shutdown(_ context.Context) error {
	r.logger.Info("Shutting down PCAP receiver")

	if r.cancel != nil {
		r.cancel()
	}

	if r.cmd != nil && r.cmd.Process != nil {
		r.logger.Debug("Killing tcpdump process", zap.Int("pid", r.cmd.Process.Pid))
		if err := r.cmd.Process.Kill(); err != nil {
			r.logger.Warn("Failed to kill capture process", zap.Error(err), zap.Int("pid", r.cmd.Process.Pid))
		}
		// Wait for process to exit and capture exit status
		waitErr := r.cmd.Wait()
		if waitErr != nil {
			if exitError, ok := waitErr.(*exec.ExitError); ok {
				r.logger.Debug("tcpdump process exited with non-zero status",
					zap.Int("pid", r.cmd.Process.Pid),
					zap.Int("exit_code", exitError.ExitCode()))
			} else {
				r.logger.Debug("tcpdump process wait completed with error",
					zap.Error(waitErr),
					zap.Int("pid", r.cmd.Process.Pid))
			}
		} else {
			r.logger.Debug("tcpdump process exited successfully", zap.Int("pid", r.cmd.Process.Pid))
		}

		// Log final process state if available
		if state := r.cmd.ProcessState; state != nil {
			r.logger.Debug("tcpdump final state",
				zap.Int("pid", r.cmd.Process.Pid),
				zap.String("state", state.String()),
				zap.Bool("exited", state.Exited()),
				zap.Bool("success", state.Success()))
		}
	} else {
		r.logger.Warn("No tcpdump process to shutdown")
	}

	r.logger.Info("PCAP receiver shut down")
	return nil
}

// readStderr reads error messages from capture tool (tcpdump/dumpcap)
func (r *pcapReceiver) readStderr(ctx context.Context, stderr io.ReadCloser) {
	r.logger.Debug("Starting stderr reader goroutine")
	defer r.logger.Debug("Stderr reader goroutine exiting")

	scanner := bufio.NewScanner(stderr)
	lineCount := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			r.logger.Debug("Stderr reader context cancelled", zap.Int("lines_read", lineCount))
			return
		default:
			line := scanner.Text()
			lineCount++
			// Log stderr output at info level to see what the capture tool is saying (it writes important messages to stderr)
			r.logger.Info("capture tool stderr", zap.String("message", line), zap.Int("line_number", lineCount))
		}
	}

	if err := scanner.Err(); err != nil {
		r.logger.Error("Error reading from capture tool stderr", zap.Error(err), zap.Int("total_lines", lineCount))
	} else {
		r.logger.Debug("Stderr scanner closed", zap.Int("total_lines_read", lineCount))
	}
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// isTimestampLine checks if a line starts with a timestamp (HH:MM:SS)
func isTimestampLine(line string) bool {
	if len(line) < 8 {
		return false
	}
	// Simple check for timestamp format: HH:MM:SS
	return line[2] == ':' && line[5] == ':'
}

// processPacket parses and emits a packet as an OTel log
func (r *pcapReceiver) processPacket(ctx context.Context, lines []string) {
	// Do not process or emit if shutdown has been initiated
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Parse the packet
	packetInfo, err := parser.ParsePacket(lines)
	if err != nil {
		r.logger.Warn("Failed to parse packet",
			zap.Error(err),
			zap.Int("line_count", len(lines)),
			zap.String("first_line", truncateString(lines[0], 100)))
		return
	}

	r.logger.Debug("Successfully parsed packet",
		zap.String("protocol", packetInfo.Protocol),
		zap.String("transport", packetInfo.Transport),
		zap.String("src", packetInfo.SrcAddress),
		zap.String("dst", packetInfo.DstAddress),
		zap.Int("length", packetInfo.Length))

	// Process the parsed packet info
	r.processPacketInfo(ctx, packetInfo)
}

// processPacketInfo emits a parsed packet as an OTel log (common for both text and binary parsing)
func (r *pcapReceiver) processPacketInfo(ctx context.Context, packetInfo *parser.PacketInfo) {
	// Do not process or emit if shutdown has been initiated
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Create OTel log
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()

	// Set timestamp
	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(packetInfo.Timestamp))
	logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(packetInfo.Timestamp))

	// Set body as hex-encoded packet data (with 0x prefix)
	logRecord.Body().SetStr("0x" + packetInfo.HexData)

	// Set attributes if enabled
	if r.config.ParseAttributes {
		attrs := logRecord.Attributes()
		attrs.PutStr("network.type", packetInfo.Protocol)
		attrs.PutStr("network.transport", packetInfo.Transport)
		attrs.PutStr("source.address", packetInfo.SrcAddress)
		attrs.PutStr("destination.address", packetInfo.DstAddress)

		if packetInfo.SrcPort > 0 {
			attrs.PutInt("source.port", int64(packetInfo.SrcPort))
		}
		if packetInfo.DstPort > 0 {
			attrs.PutInt("destination.port", int64(packetInfo.DstPort))
		}

		attrs.PutInt("packet.length", int64(packetInfo.Length))
	}

	// Consume the log
	if err := r.consumer.ConsumeLogs(ctx, logs); err != nil {
		r.logger.Error("Failed to consume packet log",
			zap.Error(err),
			zap.String("protocol", packetInfo.Protocol),
			zap.String("src", packetInfo.SrcAddress),
			zap.String("dst", packetInfo.DstAddress))
	} else {
		r.logger.Debug("Successfully consumed packet log",
			zap.String("protocol", packetInfo.Protocol),
			zap.String("transport", packetInfo.Transport))
	}
}
