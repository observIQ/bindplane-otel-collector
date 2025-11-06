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

//go:build !windows

package pcapreceiver

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"
)

// checkPrivileges checks if the process has sufficient privileges to capture packets on Unix systems
func (r *pcapReceiver) checkPrivileges() error {
	// Check root privileges
	if os.Geteuid() != 0 {
		return fmt.Errorf("packet capture requires root privileges. Please run the collector with sudo")
	}
	r.logger.Info("Running with root privileges")

	// Perform a lightweight preflight to detect permission issues and validate tcpdump availability
	preflight := newCommand("tcpdump", "-i", r.config.Interface, "-c", "1", "-w", "-")
	if err := preflight.Start(); err != nil {
		return fmt.Errorf("insufficient privileges to start tcpdump: %w. Either run with sudo, or grant capabilities: 'sudo setcap cap_net_raw,cap_net_admin=eip /usr/sbin/tcpdump' (or grant to the collector binary). Then verify: 'getcap /usr/sbin/tcpdump'", err)
	}
	_ = preflight.Process.Kill()
	_, _ = preflight.Process.Wait()
	return nil
}

// readPackets reads and parses packets from tcpdump output using text-based parsing (Unix: darwin/linux)
func (r *pcapReceiver) readPackets(ctx context.Context, stdout io.ReadCloser) {
	r.logger.Debug("Starting packet reader goroutine")
	defer r.logger.Debug("Packet reader goroutine exiting")

	scanner := bufio.NewScanner(stdout)
	// Increase buffer size for potentially large packets
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var packetLines []string
	lineCount := 0
	packetCount := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			r.logger.Debug("Packet reader context cancelled",
				zap.Int("total_lines_read", lineCount),
				zap.Int("packets_processed", packetCount))
			return
		default:
			line := scanner.Text()
			lineCount++

			// Log first few lines to see what we're getting
			if lineCount <= 10 {
				r.logger.Debug("Reading line from tcpdump stdout",
					zap.Int("line_number", lineCount),
					zap.Int("line_length", len(line)),
					zap.String("line_preview", truncateString(line, 100)))
			}

			// Check if this is the start of a new packet (timestamp line)
			if isTimestampLine(line) {
				// Process previous packet if we have one
				if len(packetLines) > 0 {
					packetCount++
					r.logger.Debug("Processing complete packet",
						zap.Int("packet_number", packetCount),
						zap.Int("packet_lines", len(packetLines)))
					r.processPacket(ctx, packetLines)
					packetLines = nil
				}
				// Start new packet
				packetLines = append(packetLines, line)
				r.logger.Debug("Detected new packet start", zap.String("header_line", truncateString(line, 150)))
			} else if len(line) > 0 {
				// Continuation of current packet (hex data)
				packetLines = append(packetLines, line)
			}
		}
	}

	r.logger.Info("Scanner finished reading",
		zap.Int("total_lines_read", lineCount),
		zap.Int("packets_processed", packetCount),
		zap.Int("buffered_lines", len(packetLines)))

	// Process last packet
	if len(packetLines) > 0 {
		// If we're shutting down, skip processing any buffered packet
		select {
		case <-ctx.Done():
			r.logger.Debug("Skipping final packet due to shutdown")
			return
		default:
			packetCount++
			r.logger.Debug("Processing final buffered packet", zap.Int("packet_number", packetCount))
			r.processPacket(ctx, packetLines)
		}
	}

	if err := scanner.Err(); err != nil {
		r.logger.Error("Error reading packet data from tcpdump stdout",
			zap.Error(err),
			zap.Int("lines_read_before_error", lineCount),
			zap.Int("packets_processed", packetCount))
	} else {
		r.logger.Debug("Scanner closed normally", zap.Int("total_lines", lineCount), zap.Int("total_packets", packetCount))
	}
}
