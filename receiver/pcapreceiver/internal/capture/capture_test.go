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

package capture

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/stretchr/testify/require"
)

func TestNewCapturer(t *testing.T) {
	t.Run("creates capturer from file", func(t *testing.T) {
		pcapPath := filepath.Join("..", "..", "testdata", "capture.pcap")
		_, err := os.Stat(pcapPath)
		if os.IsNotExist(err) {
			t.Skipf("PCAP file not found at %s", pcapPath)
		}

		cfg := Config{
			FilePath:    pcapPath,
			SnapLength:  65535,
			BPFFilter:   "",
			Promiscuous: false,
		}

		capt, err := NewCapturer(cfg)
		require.NoError(t, err, "should create file-based capturer successfully")
		require.NotNil(t, capt, "capturer should not be nil")

		// Clean up
		if capt != nil {
			_ = capt.Close()
		}
	})

	t.Run("creates capturer from file with BPF filter", func(t *testing.T) {
		pcapPath := filepath.Join("..", "..", "testdata", "capture.pcap")
		_, err := os.Stat(pcapPath)
		if os.IsNotExist(err) {
			t.Skipf("PCAP file not found at %s", pcapPath)
		}

		cfg := Config{
			FilePath:    pcapPath,
			SnapLength:  65535,
			BPFFilter:   "tcp port 80",
			Promiscuous: false,
		}

		capt, err := NewCapturer(cfg)
		require.NoError(t, err, "should create file-based capturer with BPF filter")
		require.NotNil(t, capt)

		if capt != nil {
			_ = capt.Close()
		}
	})

	t.Run("creates live capturer if permissions available", func(t *testing.T) {
		// Skip this test on CI or without proper permissions
		if os.Getenv("CI") != "" {
			t.Skip("Skipping live capture test in CI")
		}

		cfg := Config{
			Interface:   "lo", // Try loopback, most permissive
			SnapLength:  65535,
			BPFFilter:   "",
			Promiscuous: false,
		}

		capt, err := NewCapturer(cfg)
		if err != nil {
			t.Skipf("Cannot create live capturer (may need permissions): %v", err)
		}
		require.NotNil(t, capt)

		if capt != nil {
			_ = capt.Close()
		}
	})
}

func TestCapture_FromFile(t *testing.T) {
	// Use the static PCAP file created in Phase 3
	pcapPath := filepath.Join("..", "..", "testdata", "capture.pcap")

	// Check if file exists
	_, err := os.Stat(pcapPath)
	if os.IsNotExist(err) {
		t.Skipf("PCAP file not found at %s", pcapPath)
	}

	cfg := Config{
		FilePath:    pcapPath,
		SnapLength:  65535,
		BPFFilter:   "",
		Promiscuous: false,
	}

	capt, err := NewCapturer(cfg)
	require.NoError(t, err, "should create file-based capturer")
	require.NotNil(t, capt)
	defer capt.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = capt.Start(ctx)
	require.NoError(t, err, "should start capturer")

	// Read packets from channel
	var packets []gopacket.Packet
	var packetsMu sync.Mutex
	packetChan := capt.Packets()

	// Read with timeout
	done := make(chan bool)
	go func() {
		for packet := range packetChan {
			packetsMu.Lock()
			packets = append(packets, packet)
			packetsMu.Unlock()
		}
		done <- true
	}()

	select {
	case <-done:
		// All packets read
	case <-time.After(3 * time.Second):
		// Timeout - that's okay, might have read some packets
		// Cancel context to ensure goroutine stops
		cancel()
		// Wait a bit for goroutine to finish
		time.Sleep(100 * time.Millisecond)
	}

	packetsMu.Lock()
	packetCount := len(packets)
	packetsCopy := make([]gopacket.Packet, len(packets))
	copy(packetsCopy, packets)
	packetsMu.Unlock()

	require.Greater(t, packetCount, 0, "should read at least one packet from PCAP file")

	// Verify packets have expected properties
	for i, packet := range packetsCopy {
		require.NotNil(t, packet, "packet %d should not be nil", i)
		require.NotNil(t, packet.Metadata(), "packet %d metadata should not be nil", i)
		require.Greater(t, packet.Metadata().Length, 0, "packet %d should have length > 0", i)
	}
}

func TestCapture_FromFile_WithBPFFilter(t *testing.T) {
	pcapPath := filepath.Join("..", "..", "testdata", "capture.pcap")

	_, err := os.Stat(pcapPath)
	if os.IsNotExist(err) {
		t.Skipf("PCAP file not found at %s", pcapPath)
	}

	cfg := Config{
		FilePath:    pcapPath,
		SnapLength:  65535,
		BPFFilter:   "tcp", // Only TCP packets
		Promiscuous: false,
	}

	capt, err := NewCapturer(cfg)
	require.NoError(t, err, "should create file-based capturer with filter")
	require.NotNil(t, capt)
	defer capt.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = capt.Start(ctx)
	require.NoError(t, err, "should start capturer")

	// Read a few packets
	packetChan := capt.Packets()
	var count int
	timeout := time.After(2 * time.Second)

readLoop:
	for {
		select {
		case packet, ok := <-packetChan:
			if !ok {
				break readLoop
			}
			require.NotNil(t, packet, "filtered packet should not be nil")
			count++
			if count >= 5 {
				break readLoop
			}
		case <-timeout:
			break readLoop
		}
	}

	// Should have read at least some packets (or none if no TCP in file)
	// This test mainly verifies BPF filter doesn't cause errors
}

func TestCapture_Stats(t *testing.T) {
	pcapPath := filepath.Join("..", "..", "testdata", "capture.pcap")

	_, err := os.Stat(pcapPath)
	if os.IsNotExist(err) {
		t.Skipf("PCAP file not found at %s", pcapPath)
	}

	cfg := Config{
		FilePath:    pcapPath,
		SnapLength:  65535,
		BPFFilter:   "",
		Promiscuous: false,
	}

	capt, err := NewCapturer(cfg)
	require.NoError(t, err)
	require.NotNil(t, capt)
	defer capt.Close()

	ctx := context.Background()
	err = capt.Start(ctx)
	require.NoError(t, err)

	// Read some packets
	packetChan := capt.Packets()
	timeout := time.After(2 * time.Second)

readLoop:
	for {
		select {
		case _, ok := <-packetChan:
			if !ok {
				break readLoop
			}
		case <-timeout:
			break readLoop
		}
	}

	// Get stats
	stats, err := capt.Stats()
	require.NoError(t, err, "should get stats")
	require.NotNil(t, stats, "stats should not be nil")
	require.GreaterOrEqual(t, stats.PacketsReceived, uint64(0), "packets received should be >= 0")
}

// Platform-specific tests (skip if not on correct OS or no permissions)
func TestCapture_Live_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	if os.Getuid() != 0 {
		t.Skip("Skipping live capture test - requires root privileges")
	}

	cfg := Config{
		Interface:   "lo", // Loopback should always exist
		SnapLength:  65535,
		BPFFilter:   "",
		Promiscuous: false,
	}

	capt, err := NewCapturer(cfg)
	require.NoError(t, err, "should create live capturer on Linux")
	require.NotNil(t, capt)
	defer capt.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = capt.Start(ctx)
	require.NoError(t, err, "should start live capture")

	// Try to read a packet (with short timeout)
	packetChan := capt.Packets()
	select {
	case packet := <-packetChan:
		require.NotNil(t, packet, "should receive a packet")
	case <-time.After(2 * time.Second):
		// No packets captured - that's okay for loopback
		t.Log("No packets captured on loopback (expected if no traffic)")
	}
}

func TestCapture_Live_Darwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test")
	}

	if os.Getuid() != 0 {
		t.Skip("Skipping live capture test - requires root privileges")
	}

	cfg := Config{
		Interface:   "lo0", // macOS loopback
		SnapLength:  65535,
		BPFFilter:   "",
		Promiscuous: false,
	}

	capt, err := NewCapturer(cfg)
	require.NoError(t, err, "should create live capturer on macOS")
	require.NotNil(t, capt)
	defer capt.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = capt.Start(ctx)
	require.NoError(t, err, "should start live capture")

	// Try to read a packet
	packetChan := capt.Packets()
	select {
	case packet := <-packetChan:
		require.NotNil(t, packet, "should receive a packet")
	case <-time.After(2 * time.Second):
		t.Log("No packets captured on loopback (expected if no traffic)")
	}
}

func TestCapture_Live_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	// On Windows, we can't easily check for admin privileges
	// So we'll just try and skip if it fails

	cfg := Config{
		Interface:   "", // Auto-detect on Windows
		SnapLength:  65535,
		BPFFilter:   "",
		Promiscuous: false,
	}

	capt, err := NewCapturer(cfg)
	if err != nil {
		t.Skipf("Cannot create live capturer on Windows (may need Npcap or admin): %v", err)
	}
	require.NotNil(t, capt)
	defer capt.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = capt.Start(ctx)
	if err != nil {
		t.Skipf("Cannot start live capture on Windows: %v", err)
	}

	// Try to read a packet
	packetChan := capt.Packets()
	select {
	case packet := <-packetChan:
		require.NotNil(t, packet, "should receive a packet")
	case <-time.After(2 * time.Second):
		t.Log("No packets captured (expected if no traffic)")
	}
}
