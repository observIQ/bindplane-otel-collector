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

// Package capture implements a packet capturer that captures network packets from interfaces and converts packet metadata to OpenTelemetry logs.
package capture

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// Capturer defines the interface for packet capture
type Capturer interface {
	// Start begins packet capture
	Start(ctx context.Context) error

	// Packets returns a channel of captured packets
	Packets() <-chan gopacket.Packet

	// Stats returns capture statistics
	Stats() (*Stats, error)

	// Close stops capture and releases resources
	Close() error
}

// Stats contains packet capture statistics
type Stats struct {
	PacketsReceived uint64
	PacketsDropped  uint64
	InterfaceName   string
}

// Config contains configuration for the capturer
type Config struct {
	// Interface specifies the network interface to capture from
	// If empty and FilePath is empty, auto-detects the first available interface
	Interface string

	// FilePath specifies a PCAP file to read from (for testing/replay)
	// If set, live capture is disabled
	FilePath string

	// SnapLength is the maximum bytes to capture per packet
	SnapLength int32

	// BPFFilter is a Berkeley Packet Filter expression
	BPFFilter string

	// Promiscuous enables promiscuous mode
	Promiscuous bool
}

// capturer implements the Capturer interface
type capturer struct {
	handle        *pcap.Handle
	packets       chan gopacket.Packet
	config        Config
	interfaceName string
	mu            sync.Mutex
	started       bool
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewCapturer creates a new packet capturer
func NewCapturer(config Config) (Capturer, error) {
	var handle *pcap.Handle
	var err error
	var interfaceName string

	// Determine if we're reading from a file or live interface
	if config.FilePath != "" {
		// File-based capture
		handle, err = pcap.OpenOffline(config.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open pcap file %s: %w", config.FilePath, err)
		}
		interfaceName = config.FilePath
	} else {
		// Live capture
		interfaceName = config.Interface
		if interfaceName == "" {
			// Auto-detect interface
			interfaceName, err = autoDetectInterface()
			if err != nil {
				return nil, fmt.Errorf("failed to auto-detect interface: %w", err)
			}
		}

		// Open live capture
		handle, err = pcap.OpenLive(
			interfaceName,
			int32(config.SnapLength),
			config.Promiscuous,
			pcap.BlockForever,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to open interface %s: %w", interfaceName, err)
		}
	}

	// Apply BPF filter if specified
	if config.BPFFilter != "" {
		err = handle.SetBPFFilter(config.BPFFilter)
		if err != nil {
			handle.Close()
			return nil, fmt.Errorf("failed to set BPF filter '%s': %w", config.BPFFilter, err)
		}
	}

	return &capturer{
		handle:        handle,
		packets:       make(chan gopacket.Packet, 100),
		config:        config,
		interfaceName: interfaceName,
		started:       false,
	}, nil
}

// Start begins packet capture
func (c *capturer) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return fmt.Errorf("capturer already started")
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.started = true

	// Start packet reading goroutine
	c.wg.Add(1)
	go c.readPackets()

	return nil
}

// Packets returns the channel of captured packets
func (c *capturer) Packets() <-chan gopacket.Packet {
	return c.packets
}

// Stats returns capture statistics
func (c *capturer) Stats() (*Stats, error) {
	if c.handle == nil {
		return nil, fmt.Errorf("capturer not initialized")
	}

	pcapStats, err := c.handle.Stats()
	if err != nil {
		// For file-based capture, stats might not be available
		// Return zero stats instead of error
		return &Stats{
			PacketsReceived: 0,
			PacketsDropped:  0,
			InterfaceName:   c.interfaceName,
		}, nil
	}

	// Check for potential overflow
	packetsReceived := uint64(0)
	if pcapStats.PacketsReceived >= 0 {
		packetsReceived = uint64(pcapStats.PacketsReceived)
	}

	packetsDropped := uint64(0)
	if pcapStats.PacketsDropped >= 0 {
		packetsDropped = uint64(pcapStats.PacketsDropped)
	}

	return &Stats{
		PacketsReceived: packetsReceived,
		PacketsDropped:  packetsDropped,
		InterfaceName:   c.interfaceName,
	}, nil
}

// Close stops capture and cleans up resources
func (c *capturer) Close() error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Unlock()

	// Wait for goroutine to finish
	c.wg.Wait()

	// Close packet channel
	close(c.packets)

	// Close pcap handle
	if c.handle != nil {
		c.handle.Close()
	}

	return nil
}

// readPackets reads packets from the handle and sends them to the channel
func (c *capturer) readPackets() {
	defer c.wg.Done()

	packetSource := gopacket.NewPacketSource(c.handle, c.handle.LinkType())

	for {
		select {
		case <-c.ctx.Done():
			return
		case packet, ok := <-packetSource.Packets():
			if !ok {
				// No more packets (EOF for file-based capture)
				return
			}
			select {
			case c.packets <- packet:
				// Packet sent successfully
			case <-c.ctx.Done():
				return
			}
		}
	}
}

// autoDetectInterface finds the first available network interface
func autoDetectInterface() (string, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return "", fmt.Errorf("failed to find devices: %w", err)
	}

	if len(devices) == 0 {
		return "", fmt.Errorf("no network interfaces found")
	}

	// Return the first non-loopback interface, or loopback if that's all we have
	for _, device := range devices {
		// Skip loopback unless it's the only option
		if len(device.Addresses) > 0 && !isLoopback(device.Name) {
			return device.Name, nil
		}
	}

	// If we didn't find a non-loopback interface, use the first one
	return devices[0].Name, nil
}

// isLoopback checks if an interface name suggests it's a loopback interface
func isLoopback(name string) bool {
	// Common loopback interface names
	loopbackNames := []string{"lo", "lo0", "Loopback"}
	for _, lb := range loopbackNames {
		if name == lb {
			return true
		}
	}
	return false
}
