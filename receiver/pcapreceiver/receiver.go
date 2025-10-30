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
	"os"
	"os/exec"
	"runtime"

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

	// Check privileges
	if err := r.checkPrivileges(); err != nil {
		return fmt.Errorf("privilege check failed: %w", err)
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
		return fmt.Errorf("failed to build capture command for platform: %s", runtime.GOOS)
	}

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

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	// Start goroutine to read stderr
	go r.readStderr(ctx, stderr)

	// Start goroutine to read and parse packets
	go r.readPackets(ctx, stdout)

	r.logger.Info("PCAP receiver started successfully")
	return nil
}

// Shutdown stops the packet capture
func (r *pcapReceiver) Shutdown(_ context.Context) error {
	r.logger.Info("Shutting down PCAP receiver")

	if r.cancel != nil {
		r.cancel()
	}

	if r.cmd != nil && r.cmd.Process != nil {
		if err := r.cmd.Process.Kill(); err != nil {
			r.logger.Warn("Failed to kill capture process", zap.Error(err))
		}
		_ = r.cmd.Wait()
	}

	r.logger.Info("PCAP receiver shut down")
	return nil
}

// checkPrivileges checks if the process has sufficient privileges to capture packets
func (r *pcapReceiver) checkPrivileges() error {
	if runtime.GOOS == "darwin" {
		if os.Geteuid() != 0 {
			return fmt.Errorf("packet capture requires root privileges. Please run the collector with sudo")
		}
		r.logger.Info("Running with root privileges")
		return nil
	}

	if runtime.GOOS == "linux" {
		if os.Geteuid() == 0 {
			r.logger.Info("Running with root privileges")
			return nil
		}

		// Perform a lightweight preflight to detect permission issues.
		preflight := newCommand("tcpdump", "-i", r.config.Interface, "-c", "1", "-w", "-")
		if err := preflight.Start(); err != nil {
			return fmt.Errorf("insufficient privileges to start tcpdump: %w. Either run with sudo, or grant capabilities: 'sudo setcap cap_net_raw,cap_net_admin=eip /usr/sbin/tcpdump' (or grant to the collector binary). Then verify: 'getcap /usr/sbin/tcpdump'", err)
		}
		_ = preflight.Process.Kill()
		_, _ = preflight.Process.Wait()
		return nil
	}

	if runtime.GOOS == "windows" {
		// Preflight: ensure windump is usable and we have access
		exe := r.config.ExecutablePath
		if exe == "" {
			exe = "windump.exe"
		}
		// Try listing interfaces to validate Npcap presence
		if err := newCommand(exe, "-D").Run(); err != nil {
			return fmt.Errorf("npcap windump not available: %w. Install Npcap (https://nmap.org/npcap/) and ensure windump.exe is on PATH or set executable_path", err)
		}
		// Try single packet preflight
		preflight := newCommand(exe, "-i", r.config.Interface, "-c", "1", "-w", "-")
		if err := preflight.Start(); err != nil {
			return fmt.Errorf("unable to start windump: %w. Try running the collector as Administrator or reinstall Npcap without Admin-only mode", err)
		}
		_ = preflight.Process.Kill()
		_, _ = preflight.Process.Wait()
		return nil
	}

	return nil
}

// readStderr reads error messages from tcpdump
func (r *pcapReceiver) readStderr(ctx context.Context, stderr io.ReadCloser) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			line := scanner.Text()
			// Log stderr output at debug level (tcpdump writes stats and warnings to stderr)
			r.logger.Debug("tcpdump stderr", zap.String("message", line))
		}
	}
}

// readPackets reads and parses packets from tcpdump output
func (r *pcapReceiver) readPackets(ctx context.Context, stdout io.ReadCloser) {
	scanner := bufio.NewScanner(stdout)
	// Increase buffer size for potentially large packets
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var packetLines []string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			line := scanner.Text()

			// Check if this is the start of a new packet (timestamp line)
			if isTimestampLine(line) {
				// Process previous packet if we have one
				if len(packetLines) > 0 {
					r.processPacket(ctx, packetLines)
					packetLines = nil
				}
				// Start new packet
				packetLines = append(packetLines, line)
			} else if len(line) > 0 {
				// Continuation of current packet (hex data)
				packetLines = append(packetLines, line)
			}
		}
	}

	// Process last packet
	if len(packetLines) > 0 {
		// If we're shutting down, skip processing any buffered packet
		select {
		case <-ctx.Done():
			return
		default:
			r.processPacket(ctx, packetLines)
		}
	}

	if err := scanner.Err(); err != nil {
		r.logger.Error("Error reading packet data", zap.Error(err))
	}
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
	packetInfo, err := parser.ParseTcpdumpPacket(lines)
	if err != nil {
		r.logger.Warn("Failed to parse packet", zap.Error(err))
		return
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

	// Set attributes
	attrs := logRecord.Attributes()
	attrs.PutStr("network.protocol", packetInfo.Protocol)
	attrs.PutStr("network.transport", packetInfo.Transport)
	attrs.PutStr("network.src.address", packetInfo.SrcAddress)
	attrs.PutStr("network.dst.address", packetInfo.DstAddress)

	if packetInfo.SrcPort > 0 {
		attrs.PutInt("network.src.port", int64(packetInfo.SrcPort))
	}
	if packetInfo.DstPort > 0 {
		attrs.PutInt("network.dst.port", int64(packetInfo.DstPort))
	}

	attrs.PutInt("packet.length", int64(packetInfo.Length))

	// Consume the log
	if err := r.consumer.ConsumeLogs(ctx, logs); err != nil {
		r.logger.Error("Failed to consume packet log", zap.Error(err))
	}
}
