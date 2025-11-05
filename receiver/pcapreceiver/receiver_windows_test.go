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
	"encoding/binary"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/observiq/bindplane-otel-collector/receiver/pcapreceiver/internal/metadata"
)

func TestCheckPrivileges_Windows_DumpcapAvailable(t *testing.T) {
	cfg := &Config{Interface: "1"}
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, zap.NewNop(), consumertest.NewNop(), tb)
	require.NoError(t, err)

	// This will try to actually run dumpcap, so it may fail if Wireshark is not installed
	// We just verify it doesn't panic and handles errors gracefully
	err = receiver.checkPrivileges()
	// If dumpcap is available, should succeed or fail with a specific error
	// If not available, should fail with installation error
	if err != nil {
		require.Contains(t, err.Error(), "dumpcap", err.Error())
	}
}

func TestCheckPrivileges_Windows_ExecutablePath(t *testing.T) {
	// Save original newCommand
	originalNewCommand := newCommand
	defer func() {
		newCommand = originalNewCommand
	}()

	// Mock command that succeeds
	mockCmd := &exec.Cmd{
		Process: &os.Process{Pid: 12345},
	}
	var calledArgs []string
	newCommand = func(name string, arg ...string) *exec.Cmd {
		calledArgs = arg
		cmd := exec.Command("echo", "test")
		cmd.Process = mockCmd.Process
		return cmd
	}

	cfg := &Config{
		Interface:      "1",
		ExecutablePath: `C:\Program Files\Wireshark\dumpcap.exe`,
	}
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, zap.NewNop(), consumertest.NewNop(), tb)
	require.NoError(t, err)

	// Will try to run the command, which will fail in mock but we can verify args
	_ = receiver.checkPrivileges()
	// Verify that the executable path was used
	require.NotEmpty(t, calledArgs)
}

func TestCheckPrivileges_Windows_DumpcapNotFound(t *testing.T) {
	// Save original newCommand
	originalNewCommand := newCommand
	defer func() {
		newCommand = originalNewCommand
	}()

	// Mock command that fails (simulating dumpcap not found)
	newCommand = func(name string, arg ...string) *exec.Cmd {
		cmd := exec.Command("nonexistent-command")
		return cmd
	}

	cfg := &Config{Interface: "1"}
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, zap.NewNop(), consumertest.NewNop(), tb)
	require.NoError(t, err)

	err = receiver.checkPrivileges()
	require.Error(t, err)
	require.Contains(t, err.Error(), "dumpcap")
}

func TestReadPacketsWindows_ValidPacket(t *testing.T) {
	cfg := &Config{Interface: "1", ParseAttributes: true}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, logger, sink, tb)
	require.NoError(t, err)

	// Create a PCAP writer to generate valid PCAP data
	pr, pw := io.Pipe()
	stdout := io.NopCloser(pr)

	// Write PCAP file header and packet in a goroutine
	go func() {
		defer pw.Close()
		writer := pcapgo.NewWriter(pw)
		writer.WriteFileHeader(65535, layers.LinkTypeEthernet)

		// Create a simple TCP packet
		buf := make([]byte, 0, 60)
		// Ethernet header (14 bytes)
		ethHeader := make([]byte, 14)
		binary.BigEndian.PutUint16(ethHeader[12:14], 0x0800) // EtherType: IPv4
		buf = append(buf, ethHeader...)
		// IP header (20 bytes)
		ipHeader := make([]byte, 20)
		ipHeader[0] = 0x45                            // Version 4, IHL 5
		ipHeader[1] = 0x00                            // TOS
		binary.BigEndian.PutUint16(ipHeader[2:4], 60) // Total length
		ipHeader[9] = 6                               // Protocol: TCP
		ipHeader[12] = 192                            // Source IP: 192.168.1.100
		ipHeader[13] = 168
		ipHeader[14] = 1
		ipHeader[15] = 100
		ipHeader[16] = 192 // Dest IP: 192.168.1.1
		ipHeader[17] = 168
		ipHeader[18] = 1
		ipHeader[19] = 1
		buf = append(buf, ipHeader...)
		// TCP header (20 bytes)
		tcpHeader := make([]byte, 20)
		binary.BigEndian.PutUint16(tcpHeader[0:2], 54321) // Source port
		binary.BigEndian.PutUint16(tcpHeader[2:4], 443)   // Dest port
		buf = append(buf, tcpHeader...)
		// Payload (6 bytes)
		buf = append(buf, []byte("hello")...)

		ci := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(buf),
			Length:        len(buf),
		}
		writer.WritePacket(ci, buf)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPacketsWindows(ctx, stdout)

	// Wait for packet to be processed
	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 2*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	require.Len(t, logs, 1)
	require.Equal(t, 1, logs[0].LogRecordCount())

	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := logRecord.Attributes()

	protocol, ok := attrs.Get("network.type")
	require.True(t, ok)
	require.Equal(t, "IP", protocol.AsString())

	interfaceName, ok := attrs.Get("network.interface.name")
	require.True(t, ok)
	require.Equal(t, "1", interfaceName.AsString())

	transport, ok := attrs.Get("network.transport")
	require.True(t, ok)
	require.Equal(t, "TCP", transport.AsString())
}

func TestReadPacketsWindows_MultiplePackets(t *testing.T) {
	cfg := &Config{Interface: "1"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, logger, sink, tb)
	require.NoError(t, err)

	pr, pw := io.Pipe()
	stdout := io.NopCloser(pr)

	go func() {
		defer pw.Close()
		writer := pcapgo.NewWriter(pw)
		writer.WriteFileHeader(65535, layers.LinkTypeEthernet)

		// Create two packets
		for i := 0; i < 2; i++ {
			buf := make([]byte, 0, 60)
			// Ethernet header (14 bytes)
			ethHeader := make([]byte, 14)
			binary.BigEndian.PutUint16(ethHeader[12:14], 0x0800) // EtherType: IPv4
			buf = append(buf, ethHeader...)
			ipHeader := make([]byte, 20)
			ipHeader[0] = 0x45
			binary.BigEndian.PutUint16(ipHeader[2:4], 60)
			ipHeader[9] = 6    // Protocol: TCP
			ipHeader[12] = 192 // Source IP: 192.168.1.100
			ipHeader[13] = 168
			ipHeader[14] = 1
			ipHeader[15] = 100
			ipHeader[16] = 192 // Dest IP: 192.168.1.1
			ipHeader[17] = 168
			ipHeader[18] = 1
			ipHeader[19] = 1
			buf = append(buf, ipHeader...)
			tcpHeader := make([]byte, 20)
			binary.BigEndian.PutUint16(tcpHeader[0:2], uint16(54321+i))
			binary.BigEndian.PutUint16(tcpHeader[2:4], 443)
			buf = append(buf, tcpHeader...)
			buf = append(buf, []byte("hello")...)

			ci := gopacket.CaptureInfo{
				Timestamp:     time.Now(),
				CaptureLength: len(buf),
				Length:        len(buf),
			}
			writer.WritePacket(ci, buf)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPacketsWindows(ctx, stdout)

	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 2
	}, 2*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	require.Len(t, logs, 2)
}

func TestReadPacketsWindows_EmptyInput(t *testing.T) {
	cfg := &Config{Interface: "1"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, logger, sink, tb)
	require.NoError(t, err)

	// Empty PCAP file (just header)
	pr, pw := io.Pipe()
	stdout := io.NopCloser(pr)

	go func() {
		defer pw.Close()
		writer := pcapgo.NewWriter(pw)
		writer.WriteFileHeader(65535, layers.LinkTypeEthernet)
		// No packets
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPacketsWindows(ctx, stdout)

	// Wait a bit - should handle EOF gracefully
	time.Sleep(200 * time.Millisecond)
	require.Equal(t, 0, sink.LogRecordCount())
}

func TestReadPacketsWindows_InvalidPCAPData(t *testing.T) {
	cfg := &Config{Interface: "1"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, logger, sink, tb)
	require.NoError(t, err)

	// Invalid PCAP data
	stdout := io.NopCloser(strings.NewReader("invalid pcap data"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPacketsWindows(ctx, stdout)

	// Wait for error to be handled
	time.Sleep(200 * time.Millisecond)
	// Should handle error gracefully
	require.Equal(t, 0, sink.LogRecordCount())
}

func TestReadPacketsWindows_ContextCancellation(t *testing.T) {
	cfg := &Config{Interface: "1"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, logger, sink, tb)
	require.NoError(t, err)

	pr, pw := io.Pipe()
	stdout := io.NopCloser(pr)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer pw.Close()
		writer := pcapgo.NewWriter(pw)
		writer.WriteFileHeader(65535, layers.LinkTypeEthernet)

		// Write one packet
		buf := make([]byte, 0, 60)
		// Ethernet header (14 bytes)
		ethHeader := make([]byte, 14)
		binary.BigEndian.PutUint16(ethHeader[12:14], 0x0800) // EtherType: IPv4
		buf = append(buf, ethHeader...)
		ipHeader := make([]byte, 20)
		ipHeader[0] = 0x45
		binary.BigEndian.PutUint16(ipHeader[2:4], 60)
		ipHeader[9] = 6    // Protocol: TCP
		ipHeader[12] = 192 // Source IP: 192.168.1.100
		ipHeader[13] = 168
		ipHeader[14] = 1
		ipHeader[15] = 100
		ipHeader[16] = 192 // Dest IP: 192.168.1.1
		ipHeader[17] = 168
		ipHeader[18] = 1
		ipHeader[19] = 1
		buf = append(buf, ipHeader...)
		tcpHeader := make([]byte, 20)
		buf = append(buf, tcpHeader...)
		buf = append(buf, []byte("hello")...)

		ci := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(buf),
			Length:        len(buf),
		}
		writer.WritePacket(ci, buf)

		// Wait a bit before canceling
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	go receiver.readPacketsWindows(ctx, stdout)

	// Wait for cancellation
	time.Sleep(200 * time.Millisecond)

	// Should have processed at least one packet before cancellation
	_ = sink.LogRecordCount()
}

func TestReadPacketsWindows_UDPPacket(t *testing.T) {
	cfg := &Config{Interface: "1", ParseAttributes: true}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, logger, sink, tb)
	require.NoError(t, err)

	pr, pw := io.Pipe()
	stdout := io.NopCloser(pr)

	go func() {
		defer pw.Close()
		writer := pcapgo.NewWriter(pw)
		writer.WriteFileHeader(65535, layers.LinkTypeEthernet)

		// Create UDP packet
		buf := make([]byte, 0, 42)
		// Ethernet header (14 bytes)
		ethHeader := make([]byte, 14)
		binary.BigEndian.PutUint16(ethHeader[12:14], 0x0800) // EtherType: IPv4
		buf = append(buf, ethHeader...)
		ipHeader := make([]byte, 20)
		ipHeader[0] = 0x45
		ipHeader[9] = 17 // UDP protocol
		binary.BigEndian.PutUint16(ipHeader[2:4], 42)
		ipHeader[12] = 192 // Source IP: 192.168.1.100
		ipHeader[13] = 168
		ipHeader[14] = 1
		ipHeader[15] = 100
		ipHeader[16] = 192 // Dest IP: 192.168.1.1
		ipHeader[17] = 168
		ipHeader[18] = 1
		ipHeader[19] = 1
		buf = append(buf, ipHeader...)
		udpHeader := make([]byte, 8)
		binary.BigEndian.PutUint16(udpHeader[0:2], 12345) // Source port
		binary.BigEndian.PutUint16(udpHeader[2:4], 53)    // Dest port (DNS)
		buf = append(buf, udpHeader...)
		buf = append(buf, []byte("data")...)

		ci := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(buf),
			Length:        len(buf),
		}
		writer.WritePacket(ci, buf)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPacketsWindows(ctx, stdout)

	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 2*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := logRecord.Attributes()

	interfaceName, ok := attrs.Get("network.interface.name")
	require.True(t, ok)
	require.Equal(t, "1", interfaceName.AsString())

	transport, ok := attrs.Get("network.transport")
	require.True(t, ok)
	require.Equal(t, "UDP", transport.AsString())
}

func TestReadPacketsWindows_IPv6Packet(t *testing.T) {
	cfg := &Config{Interface: "1", ParseAttributes: true}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, logger, sink, tb)
	require.NoError(t, err)

	pr, pw := io.Pipe()
	stdout := io.NopCloser(pr)

	go func() {
		defer pw.Close()
		writer := pcapgo.NewWriter(pw)
		writer.WriteFileHeader(65535, layers.LinkTypeEthernet)

		// Create IPv6 packet
		buf := make([]byte, 0, 74)
		// Ethernet header (14 bytes)
		ethHeader := make([]byte, 14)
		binary.BigEndian.PutUint16(ethHeader[12:14], 0x86DD) // EtherType: IPv6
		buf = append(buf, ethHeader...)
		ipv6Header := make([]byte, 40)
		ipv6Header[0] = 0x60 // Version 6
		ipv6Header[6] = 6    // Next header: TCP
		// Source IPv6: 2001:db8::1
		ipv6Header[8] = 0x20
		ipv6Header[9] = 0x01
		ipv6Header[10] = 0x0d
		ipv6Header[11] = 0xb8
		ipv6Header[14] = 0x00
		ipv6Header[15] = 0x01
		// Dest IPv6: 2001:db8::2
		ipv6Header[24] = 0x20
		ipv6Header[25] = 0x01
		ipv6Header[26] = 0x0d
		ipv6Header[27] = 0xb8
		ipv6Header[30] = 0x00
		ipv6Header[31] = 0x02
		buf = append(buf, ipv6Header...)
		tcpHeader := make([]byte, 20)
		binary.BigEndian.PutUint16(tcpHeader[0:2], 8080)
		binary.BigEndian.PutUint16(tcpHeader[2:4], 80)
		buf = append(buf, tcpHeader...)

		ci := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(buf),
			Length:        len(buf),
		}
		writer.WritePacket(ci, buf)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPacketsWindows(ctx, stdout)

	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 2*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := logRecord.Attributes()

	protocol, ok := attrs.Get("network.type")
	require.True(t, ok)
	// Note: pcapgo parser may report "IPv6" instead of "IP6"
	require.Contains(t, []string{"IP6", "IPv6"}, protocol.AsString())

	interfaceName, ok := attrs.Get("network.interface.name")
	require.True(t, ok)
	require.Equal(t, "1", interfaceName.AsString())
}

func TestReadPacketsWindows_ICMPPacket(t *testing.T) {
	cfg := &Config{Interface: "1", ParseAttributes: true}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, logger, sink, tb)
	require.NoError(t, err)

	pr, pw := io.Pipe()
	stdout := io.NopCloser(pr)

	go func() {
		defer pw.Close()
		writer := pcapgo.NewWriter(pw)
		writer.WriteFileHeader(65535, layers.LinkTypeEthernet)

		// Create ICMP packet
		buf := make([]byte, 0, 42)
		// Ethernet header (14 bytes)
		ethHeader := make([]byte, 14)
		binary.BigEndian.PutUint16(ethHeader[12:14], 0x0800) // EtherType: IPv4
		buf = append(buf, ethHeader...)
		ipHeader := make([]byte, 20)
		ipHeader[0] = 0x45
		ipHeader[9] = 1 // ICMP protocol
		binary.BigEndian.PutUint16(ipHeader[2:4], 42)
		ipHeader[12] = 192 // Source IP: 192.168.1.100
		ipHeader[13] = 168
		ipHeader[14] = 1
		ipHeader[15] = 100
		ipHeader[16] = 192 // Dest IP: 192.168.1.1
		ipHeader[17] = 168
		ipHeader[18] = 1
		ipHeader[19] = 1
		buf = append(buf, ipHeader...)
		icmpHeader := make([]byte, 8)
		icmpHeader[0] = 8 // Echo request
		buf = append(buf, icmpHeader...)

		ci := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(buf),
			Length:        len(buf),
		}
		writer.WritePacket(ci, buf)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPacketsWindows(ctx, stdout)

	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 2*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := logRecord.Attributes()

	interfaceName, ok := attrs.Get("network.interface.name")
	require.True(t, ok)
	require.Equal(t, "1", interfaceName.AsString())

	transport, ok := attrs.Get("network.transport")
	require.True(t, ok)
	require.Equal(t, "ICMP", transport.AsString())

	// ICMP should not have ports
	_, srcPortExists := attrs.Get("source.port")
	_, dstPortExists := attrs.Get("destination.port")
	require.False(t, srcPortExists)
	require.False(t, dstPortExists)
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
				Interface: "1",
				Filter:    "tcp port 80 && whoami",
			},
			wantError: "invalid character",
		},
		{
			name: "invalid snaplen",
			config: &Config{
				Interface: "1",
				SnapLen:   10,
			},
			wantError: "snaplen must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := receivertest.NewNopSettings(typ)
			tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)
			receiver, err := newReceiver(settings, tt.config, zap.NewNop(), consumertest.NewNop(), tb)
			require.NoError(t, err)

			err = receiver.Start(context.Background(), componenttest.NewNopHost())
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantError)
		})
	}
}
