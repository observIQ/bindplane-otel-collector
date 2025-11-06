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

//go:build windows

package pcapreceiver

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/google/gopacket/pcapgo"
	"github.com/observiq/bindplane-otel-collector/receiver/pcapreceiver/parser"
	"go.uber.org/zap"
)

// checkPrivileges checks if the process has sufficient privileges to capture packets on Windows
func (r *pcapReceiver) checkPrivileges() error {
	// Preflight: ensure dumpcap is usable and we have access
	exe := r.config.ExecutablePath
	if exe == "" {
		// Try common Wireshark installation paths
		commonPaths := []string{
			`C:\Program Files\Wireshark\dumpcap.exe`,
			`C:\Program Files (x86)\Wireshark\dumpcap.exe`,
		}
		for _, path := range commonPaths {
			if _, err := exec.LookPath(path); err == nil {
				exe = path
				break
			}
		}
		if exe == "" {
			exe = "dumpcap"
		}
	}
	// Try listing interfaces to validate Wireshark/Npcap presence
	if err := newCommand(exe, "-D").Run(); err != nil {
		return fmt.Errorf("dumpcap (Wireshark) not available: %w. Install Wireshark (https://www.wireshark.org/download.html) which includes Npcap, or ensure dumpcap.exe is on PATH or set executable_path", err)
	}
	// Try single packet preflight
	preflight := newCommand(exe, "-i", r.config.Interface, "-c", "1", "-x")
	if err := preflight.Start(); err != nil {
		return fmt.Errorf("unable to start dumpcap: %w. Try running the collector as Administrator or ensure Npcap is installed and not in Admin-only mode", err)
	}
	_ = preflight.Process.Kill()
	_, _ = preflight.Process.Wait()
	return nil
}

// readPackets reads and parses packets from dumpcap output using pcapgo (Windows only)
func (r *pcapReceiver) readPackets(ctx context.Context, stdout io.ReadCloser) {
	r.readPacketsWindows(ctx, stdout)
}

// readPacketsWindows reads and parses packets from binary PCAP data using pcapgo (Windows only)
func (r *pcapReceiver) readPacketsWindows(ctx context.Context, stdout io.ReadCloser) {
	r.logger.Debug("Starting Windows packet reader goroutine (pcapgo)")
	defer r.logger.Debug("Windows packet reader goroutine exiting")

	// Create pcapgo reader for binary PCAP data
	reader, err := pcapgo.NewReader(stdout)
	if err != nil {
		r.logger.Error("Failed to create pcapgo reader",
			zap.Error(err))
		return
	}

	packetCount := 0

	for {
		select {
		case <-ctx.Done():
			r.logger.Debug("Windows packet reader context cancelled",
				zap.Int("packets_processed", packetCount))
			return
		default:
			// Read packet data from binary PCAP stream
			data, ci, err := reader.ReadPacketData()
			if err != nil {
				if err == io.EOF {
					r.logger.Debug("PCAP stream ended",
						zap.Int("total_packets", packetCount))
					return
				}
				r.logger.Error("Error reading packet data from dumpcap stdout",
					zap.Error(err),
					zap.Int("packets_processed", packetCount))
				return
			}

			packetCount++
			r.logger.Debug("Reading packet from dumpcap",
				zap.Int("packet_number", packetCount),
				zap.Int("packet_length", len(data)))

			// Parse binary packet using pcapgo parser
			packetInfo, err := parser.ParsePcapgoPacket(data, ci)
			if err != nil {
				r.logger.Warn("Failed to parse binary packet",
					zap.Error(err),
					zap.Int("packet_number", packetCount),
					zap.Int("packet_length", len(data)))
				continue
			}

			r.logger.Debug("Successfully parsed binary packet",
				zap.String("protocol", packetInfo.Protocol),
				zap.String("transport", packetInfo.Transport),
				zap.String("src", packetInfo.SrcAddress),
				zap.String("dst", packetInfo.DstAddress),
				zap.Int("length", packetInfo.Length))

			// Process and emit the packet
			r.processPacketInfo(ctx, packetInfo)
		}
	}
}
