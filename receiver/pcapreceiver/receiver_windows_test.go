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
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

func TestCheckPrivileges_Windows_Success(t *testing.T) {
	testInterface := `\Device\NPF_{12345678-1234-1234-1234-123456789012}`
	cfg := &Config{Interface: testInterface}
	receiver := newTestReceiver(t, cfg, nil, nil)

	// Set up mock with the interface present
	mock := newMockPcapInterface().withDevices(
		pcap.Interface{
			Name:        testInterface,
			Description: "Test Network Interface",
		},
	)
	cleanup := setupMock(mock)
	defer cleanup()

	err := receiver.checkPrivileges()
	require.NoError(t, err)
}

func TestCheckPrivileges_Windows_NpcapNotInstalled(t *testing.T) {
	cfg := &Config{Interface: `\Device\NPF_{12345678-1234-1234-1234-123456789012}`}
	receiver := newTestReceiver(t, cfg, nil, nil)

	// Set up mock to return Npcap not installed error
	mock := newMockPcapInterface().withFindAllError(errNpcapNotInstalled)
	cleanup := setupMock(mock)
	defer cleanup()

	err := receiver.checkPrivileges()
	require.Error(t, err)
	require.Contains(t, err.Error(), "Npcap")
}

func TestCheckPrivileges_Windows_NoDevices(t *testing.T) {
	cfg := &Config{Interface: `\Device\NPF_{12345678-1234-1234-1234-123456789012}`}
	receiver := newTestReceiver(t, cfg, nil, nil)

	// Set up mock with no devices
	mock := newMockPcapInterface().withDevices()
	cleanup := setupMock(mock)
	defer cleanup()

	err := receiver.checkPrivileges()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no network interfaces found")
}

func TestCheckPrivileges_Windows_InterfaceNotFound(t *testing.T) {
	cfg := &Config{Interface: `\Device\NPF_{nonexistent-guid}`}
	receiver := newTestReceiver(t, cfg, nil, nil)

	// Set up mock with different interface
	mock := newMockPcapInterface().withDevices(
		pcap.Interface{
			Name:        `\Device\NPF_{12345678-1234-1234-1234-123456789012}`,
			Description: "Test Network Interface",
		},
	)
	cleanup := setupMock(mock)
	defer cleanup()

	err := receiver.checkPrivileges()
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
	require.Contains(t, err.Error(), "Available interfaces")
}

func TestStart_Success(t *testing.T) {
	testInterface := `\Device\NPF_{12345678-1234-1234-1234-123456789012}`
	cfg := &Config{
		Interface:       testInterface,
		SnapLen:         65535,
		Promiscuous:     true,
		ParseAttributes: true,
	}
	sink := &consumertest.LogsSink{}
	receiver := newTestReceiver(t, cfg, nil, sink)

	// Set up mock with successful operations
	mockHandle := newMockPcapHandle()
	mock := newMockPcapInterface().
		withDevices(pcap.Interface{Name: testInterface}).
		withOpenHandle(mockHandle)
	cleanup := setupMock(mock)
	defer cleanup()

	ctx := context.Background()
	err := receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	// Verify handle was stored
	require.NotNil(t, receiver.pcapHandle)

	// Shutdown
	err = receiver.Shutdown(ctx)
	require.NoError(t, err)
	require.True(t, mockHandle.closed)
}

func TestStart_WithBPFFilter(t *testing.T) {
	testInterface := `\Device\NPF_{12345678-1234-1234-1234-123456789012}`
	cfg := &Config{
		Interface:       testInterface,
		SnapLen:         65535,
		Promiscuous:     true,
		Filter:          "tcp port 443",
		ParseAttributes: true,
	}
	sink := &consumertest.LogsSink{}
	receiver := newTestReceiver(t, cfg, nil, sink)

	// Set up mock with successful operations
	mockHandle := newMockPcapHandle()
	mock := newMockPcapInterface().
		withDevices(pcap.Interface{Name: testInterface}).
		withOpenHandle(mockHandle)
	cleanup := setupMock(mock)
	defer cleanup()

	ctx := context.Background()
	err := receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	// Shutdown
	err = receiver.Shutdown(ctx)
	require.NoError(t, err)
}

func TestStart_BPFFilterError(t *testing.T) {
	testInterface := `\Device\NPF_{12345678-1234-1234-1234-123456789012}`
	cfg := &Config{
		Interface:       testInterface,
		SnapLen:         65535,
		Promiscuous:     true,
		Filter:          "invalid filter syntax",
		ParseAttributes: true,
	}
	sink := &consumertest.LogsSink{}
	receiver := newTestReceiver(t, cfg, nil, sink)

	// Set up mock with BPF filter error
	mockHandle := newMockPcapHandle().withSetBPFError(errNpcapNotInstalled)
	mock := newMockPcapInterface().
		withDevices(pcap.Interface{Name: testInterface}).
		withOpenHandle(mockHandle)
	cleanup := setupMock(mock)
	defer cleanup()

	ctx := context.Background()
	err := receiver.Start(ctx, componenttest.NewNopHost())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to set BPF filter")

	// Handle should be closed on error
	require.True(t, mockHandle.closed)
}

func TestStart_OpenLiveError(t *testing.T) {
	testInterface := `\Device\NPF_{12345678-1234-1234-1234-123456789012}`
	cfg := &Config{
		Interface:       testInterface,
		SnapLen:         65535,
		Promiscuous:     true,
		ParseAttributes: true,
	}
	sink := &consumertest.LogsSink{}
	receiver := newTestReceiver(t, cfg, nil, sink)

	// Set up mock with OpenLive error
	mock := newMockPcapInterface().
		withDevices(pcap.Interface{Name: testInterface}).
		withOpenError(errNpcapNotInstalled)
	cleanup := setupMock(mock)
	defer cleanup()

	ctx := context.Background()
	err := receiver.Start(ctx, componenttest.NewNopHost())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to open capture handle")
}

func TestStart_PrivilegeCheckFails_NoError(t *testing.T) {
	// When checkPrivileges fails, Start should log a warning but return nil
	testInterface := `\Device\NPF_{12345678-1234-1234-1234-123456789012}`
	cfg := &Config{
		Interface:       testInterface,
		SnapLen:         65535,
		Promiscuous:     true,
		ParseAttributes: true,
	}
	sink := &consumertest.LogsSink{}
	receiver := newTestReceiver(t, cfg, nil, sink)

	// Set up mock with Npcap not installed
	mock := newMockPcapInterface().withFindAllError(errNpcapNotInstalled)
	cleanup := setupMock(mock)
	defer cleanup()

	ctx := context.Background()
	err := receiver.Start(ctx, componenttest.NewNopHost())
	// Should not return error, just log warning
	require.NoError(t, err)
}

func TestStart_InvalidConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError string
	}{
		{
			name: "empty interface",
			config: &Config{
				Interface: "",
			},
			wantError: "interface must be specified",
		},
		{
			name: "invalid interface with shell injection",
			config: &Config{
				Interface: "1; whoami",
			},
			wantError: "invalid character",
		},
		{
			name: "invalid filter",
			config: &Config{
				Interface: `\Device\NPF_{12345678-1234-1234-1234-123456789012}`,
				Filter:    "tcp port 80 && whoami",
			},
			wantError: "invalid character",
		},
		{
			name: "invalid snaplen",
			config: &Config{
				Interface: `\Device\NPF_{12345678-1234-1234-1234-123456789012}`,
				SnapLen:   10,
			},
			wantError: "snaplen must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := newTestReceiver(t, tt.config, nil, nil)

			err := receiver.Start(context.Background(), componenttest.NewNopHost())
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestShutdown_Windows(t *testing.T) {
	testInterface := `\Device\NPF_{12345678-1234-1234-1234-123456789012}`
	cfg := &Config{
		Interface:   testInterface,
		SnapLen:     65535,
		Promiscuous: true,
	}
	receiver := newTestReceiver(t, cfg, nil, nil)

	// Set up mock
	mockHandle := newMockPcapHandle()
	mock := newMockPcapInterface().
		withDevices(pcap.Interface{Name: testInterface}).
		withOpenHandle(mockHandle)
	cleanup := setupMock(mock)
	defer cleanup()

	ctx := context.Background()

	// Start receiver
	err := receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	// Wait a short time for goroutines to start
	time.Sleep(10 * time.Millisecond)

	// Shutdown
	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Verify handle was closed
	require.True(t, mockHandle.closed)
}

func TestShutdown_Windows_NoHandle(t *testing.T) {
	testInterface := `\Device\NPF_{12345678-1234-1234-1234-123456789012}`
	cfg := &Config{
		Interface:   testInterface,
		SnapLen:     65535,
		Promiscuous: true,
	}
	receiver := newTestReceiver(t, cfg, nil, nil)

	// Shutdown without starting (no handle)
	err := receiver.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestReadPacketsWindows_TCPPacket(t *testing.T) {
	testInterface := `\Device\NPF_{12345678-1234-1234-1234-123456789012}`
	cfg := &Config{
		Interface:       testInterface,
		SnapLen:         65535,
		Promiscuous:     true,
		ParseAttributes: true,
	}
	sink := &consumertest.LogsSink{}
	receiver := newTestReceiver(t, cfg, nil, sink)

	// Create a mock handle that returns a TCP packet then EOF
	mockHandle := newMockPcapHandle().withPackets(
		mockPacket{
			data: sampleTCPPacket,
			ci: gopacket.CaptureInfo{
				Timestamp:     time.Now(),
				CaptureLength: len(sampleTCPPacket),
				Length:        len(sampleTCPPacket),
			},
		},
	)

	mock := newMockPcapInterface().
		withDevices(pcap.Interface{Name: testInterface}).
		withOpenHandle(mockHandle)
	cleanup := setupMock(mock)
	defer cleanup()

	ctx := context.Background()
	err := receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	// Wait for packet to be processed
	require.Eventually(t, func() bool {
		return sink.LogRecordCount() >= 1
	}, 2*time.Second, 10*time.Millisecond, "Expected at least 1 log record")

	// Shutdown
	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Verify packet attributes
	logs := sink.AllLogs()
	require.GreaterOrEqual(t, len(logs), 1)
	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := logRecord.Attributes()

	// Check network type
	networkType, ok := attrs.Get("network.type")
	require.True(t, ok)
	require.Equal(t, "IP", networkType.AsString())

	// Check transport
	transport, ok := attrs.Get("network.transport")
	require.True(t, ok)
	require.Equal(t, "TCP", transport.AsString())

	// Check source address
	srcAddr, ok := attrs.Get("source.address")
	require.True(t, ok)
	require.Equal(t, "192.168.1.100", srcAddr.AsString())

	// Check destination address
	dstAddr, ok := attrs.Get("destination.address")
	require.True(t, ok)
	require.Equal(t, "192.168.1.1", dstAddr.AsString())

	// Check ports
	srcPort, ok := attrs.Get("source.port")
	require.True(t, ok)
	require.Equal(t, int64(54321), srcPort.Int())

	dstPort, ok := attrs.Get("destination.port")
	require.True(t, ok)
	require.Equal(t, int64(443), dstPort.Int())
}

func TestReadPacketsWindows_UDPPacket(t *testing.T) {
	testInterface := `\Device\NPF_{12345678-1234-1234-1234-123456789012}`
	cfg := &Config{
		Interface:       testInterface,
		SnapLen:         65535,
		Promiscuous:     true,
		ParseAttributes: true,
	}
	sink := &consumertest.LogsSink{}
	receiver := newTestReceiver(t, cfg, nil, sink)

	// Create a mock handle that returns a UDP packet then EOF
	mockHandle := newMockPcapHandle().withPackets(
		mockPacket{
			data: sampleUDPPacket,
			ci: gopacket.CaptureInfo{
				Timestamp:     time.Now(),
				CaptureLength: len(sampleUDPPacket),
				Length:        len(sampleUDPPacket),
			},
		},
	)

	mock := newMockPcapInterface().
		withDevices(pcap.Interface{Name: testInterface}).
		withOpenHandle(mockHandle)
	cleanup := setupMock(mock)
	defer cleanup()

	ctx := context.Background()
	err := receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	// Wait for packet to be processed
	require.Eventually(t, func() bool {
		return sink.LogRecordCount() >= 1
	}, 2*time.Second, 10*time.Millisecond, "Expected at least 1 log record")

	// Shutdown
	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Verify packet attributes
	logs := sink.AllLogs()
	require.GreaterOrEqual(t, len(logs), 1)
	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := logRecord.Attributes()

	// Check transport
	transport, ok := attrs.Get("network.transport")
	require.True(t, ok)
	require.Equal(t, "UDP", transport.AsString())

	// Check source address
	srcAddr, ok := attrs.Get("source.address")
	require.True(t, ok)
	require.Equal(t, "10.0.0.5", srcAddr.AsString())

	// Check destination address
	dstAddr, ok := attrs.Get("destination.address")
	require.True(t, ok)
	require.Equal(t, "8.8.8.8", dstAddr.AsString())

	// Check ports
	srcPort, ok := attrs.Get("source.port")
	require.True(t, ok)
	require.Equal(t, int64(12345), srcPort.Int())

	dstPort, ok := attrs.Get("destination.port")
	require.True(t, ok)
	require.Equal(t, int64(53), dstPort.Int())
}

func TestReadPacketsWindows_MultiplePackets(t *testing.T) {
	testInterface := `\Device\NPF_{12345678-1234-1234-1234-123456789012}`
	cfg := &Config{
		Interface:       testInterface,
		SnapLen:         65535,
		Promiscuous:     true,
		ParseAttributes: true,
	}
	sink := &consumertest.LogsSink{}
	receiver := newTestReceiver(t, cfg, nil, sink)

	// Create a mock handle that returns multiple packets then EOF
	mockHandle := newMockPcapHandle().withPackets(
		mockPacket{
			data: sampleTCPPacket,
			ci: gopacket.CaptureInfo{
				Timestamp:     time.Now(),
				CaptureLength: len(sampleTCPPacket),
				Length:        len(sampleTCPPacket),
			},
		},
		mockPacket{
			data: sampleUDPPacket,
			ci: gopacket.CaptureInfo{
				Timestamp:     time.Now(),
				CaptureLength: len(sampleUDPPacket),
				Length:        len(sampleUDPPacket),
			},
		},
	)

	mock := newMockPcapInterface().
		withDevices(pcap.Interface{Name: testInterface}).
		withOpenHandle(mockHandle)
	cleanup := setupMock(mock)
	defer cleanup()

	ctx := context.Background()
	err := receiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	// Wait for both packets to be processed
	require.Eventually(t, func() bool {
		return sink.LogRecordCount() >= 2
	}, 2*time.Second, 10*time.Millisecond, "Expected at least 2 log records")

	// Shutdown
	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Verify we got both packets
	require.GreaterOrEqual(t, sink.LogRecordCount(), 2)
}
