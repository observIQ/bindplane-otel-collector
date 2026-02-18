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
	"time"

	"github.com/gopacket/gopacket"
	"github.com/observiq/bindplane-otel-collector/receiver/pcapreceiver/parser"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

// Start starts the packet capture using gopacket/pcap on Windows
func (r *pcapReceiver) Start(ctx context.Context, _ component.Host) error {
	r.logger.Debug("Starting PCAP receiver", zap.String("interface", r.config.Interface))

	// Validate configuration first
	if err := r.config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	if err := r.checkPrivileges(); err != nil {
		r.logger.Warn("PCAP receiver cannot collect packets due to Npcap driver issues",
			zap.Error(err),
			zap.String("message", "No packets will be collected. Please ensure Npcap is installed and the interface is available."))
		return nil
	}

	// Open live capture handle
	handle, err := pcapOps.OpenLive(
		r.config.Interface,
		int32(r.config.SnapLen),
		r.config.Promiscuous,
		time.Second, // Timeout for packet buffer
	)
	if err != nil {
		return fmt.Errorf("failed to open capture handle. Ensure Npcap is installed and the interface exists: %w", err)
	}

	// Set BPF filter if configured
	if r.config.Filter != "" {
		if err := handle.SetBPFFilter(r.config.Filter); err != nil {
			handle.Close()
			return fmt.Errorf("failed to set BPF filter %q: %w", r.config.Filter, err)
		}
		r.logger.Debug("Applied BPF filter", zap.String("filter", r.config.Filter))
	}

	// Store handle for cleanup
	r.pcapHandle = handle

	r.logger.Info("PCAP capture handle activated",
		zap.String("interface", r.config.Interface),
		zap.Int("snaplen", r.config.SnapLen),
		zap.Bool("promiscuous", r.config.Promiscuous),
		zap.String("filter", r.config.Filter))

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	// Start goroutine to read and parse packets
	go r.readPacketsWindows(ctx, handle)

	r.logger.Debug("PCAP receiver started successfully")
	return nil
}

// Shutdown stops the packet capture on Windows
func (r *pcapReceiver) Shutdown(_ context.Context) error {
	r.logger.Info("Shutting down PCAP receiver")

	if r.cancel != nil {
		r.cancel()
	}

	if r.pcapHandle != nil {
		if handle, ok := r.pcapHandle.(PcapHandle); ok {
			r.logger.Debug("Closing pcap handle")
			handle.Close()
			r.logger.Debug("Pcap handle closed successfully")
		}
	} else {
		r.logger.Warn("No pcap handle to shutdown")
	}

	r.logger.Info("PCAP receiver shut down")
	return nil
}

// checkPrivileges checks if Npcap driver is available on Windows
func (r *pcapReceiver) checkPrivileges() error {
	// Try to find all devices to validate Npcap presence
	devices, err := pcapOps.FindAllDevs()
	if err != nil {
		return fmt.Errorf("Npcap driver not available: %w. Install Npcap from https://npcap.com/ (included with Wireshark)", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no network interfaces found. Ensure Npcap is properly installed")
	}

	// Validate requested interface exists
	found := false
	var availableDevices []string
	for _, dev := range devices {
		availableDevices = append(availableDevices, dev.Name)
		if dev.Name == r.config.Interface {
			found = true
			r.logger.Debug("Found requested interface",
				zap.String("interface", dev.Name),
				zap.String("description", dev.Description))
			break
		}
	}

	if !found {
		return fmt.Errorf("interface %s not found. Available interfaces: %v", r.config.Interface, availableDevices)
	}

	return nil
}

// readPacketsWindows reads and parses packets from gopacket/pcap (Windows only)
func (r *pcapReceiver) readPacketsWindows(ctx context.Context, handle PcapHandle) {
	r.logger.Debug("Starting Windows packet reader goroutine (gopacket/pcap)")
	defer r.logger.Debug("Windows packet reader goroutine exiting")

	packetCount := 0
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for {
		select {
		case <-ctx.Done():
			r.logger.Debug("Windows packet reader context cancelled",
				zap.Int("packets_processed", packetCount))
			return
		case packet := <-packetSource.Packets():
			if packet == nil {
				r.logger.Debug("Packet source closed",
					zap.Int("total_packets", packetCount))
				return
			}

			packetCount++
			r.logger.Debug("Reading packet from pcap",
				zap.Int("packet_number", packetCount),
				zap.Int("packet_length", len(packet.Data())))

			// Convert packet to PacketInfo using gopacket metadata
			captureInfo := packet.Metadata().CaptureInfo

			// Parse binary packet using gopacket parser
			packetInfo, err := parser.ParsePcapgoPacket(packet.Data(), captureInfo)
			if err != nil {
				r.logger.Warn("Failed to parse binary packet",
					zap.Error(err),
					zap.Int("packet_number", packetCount),
					zap.Int("packet_length", len(packet.Data())))
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
