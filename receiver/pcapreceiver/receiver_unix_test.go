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
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestCheckPrivileges_Unix_Root(t *testing.T) {
	// Save original newCommand
	originalNewCommand := newCommand
	defer func() {
		newCommand = originalNewCommand
	}()

	// Mock successful command execution (simulating root privileges)
	mockCmd := &exec.Cmd{
		Process: &os.Process{Pid: 12345},
	}
	newCommand = func(_ string, _ ...string) *exec.Cmd {
		cmd := exec.Command("echo", "test")
		cmd.Process = mockCmd.Process
		return cmd
	}

	cfg := &Config{Interface: "en0"}
	receiver := newReceiver(cfg, zap.NewNop(), consumertest.NewNop())

	// Test with root privileges (Geteuid == 0)
	if os.Geteuid() == 0 {
		// If we're actually root, the test will try to run real tcpdump
		// So we'll just verify the error message structure
		err := receiver.checkPrivileges()
		// If tcpdump is available and we have privileges, it should succeed
		// If not, it will fail with a specific error
		if err != nil {
			// Should be about insufficient privileges or tcpdump not available
			require.Contains(t, err.Error(), "privileges", err.Error())
		}
	} else {
		// Not root, should fail with root privileges error
		err := receiver.checkPrivileges()
		require.Error(t, err)
		require.Contains(t, err.Error(), "root privileges")
	}
}

func TestCheckPrivileges_Unix_NoRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Test requires running without root privileges")
	}

	cfg := &Config{Interface: "en0"}
	receiver := newReceiver(cfg, zap.NewNop(), consumertest.NewNop())

	err := receiver.checkPrivileges()
	require.Error(t, err)
	require.Contains(t, err.Error(), "root privileges")
	require.Contains(t, err.Error(), "sudo")
}

func TestCheckPrivileges_Unix_TcpdumpNotFound(t *testing.T) {
	// Save original newCommand
	originalNewCommand := newCommand
	defer func() {
		newCommand = originalNewCommand
	}()

	// Mock command that fails to start
	newCommand = func(_ string, _ ...string) *exec.Cmd {
		cmd := exec.Command("nonexistent-command-that-does-not-exist")
		return cmd
	}

	cfg := &Config{Interface: "en0"}
	receiver := newReceiver(cfg, zap.NewNop(), consumertest.NewNop())

	err := receiver.checkPrivileges()
	// Should fail when trying to start the preflight command
	require.Error(t, err)
	require.Contains(t, err.Error(), "privileges")
}

func TestReadPackets_Unix_SinglePacket(t *testing.T) {
	cfg := &Config{Interface: "en0"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	receiver := newReceiver(cfg, logger, sink)

	// Create input simulating tcpdump output
	input := "12:34:56.789012 IP 192.168.1.100.54321 > 192.168.1.1.443: Flags [S]\n" +
		"\t0x0000:  4500 003c 1c46 4000 4006 b1e6 c0a8 0164\n" +
		"\t0x0010:  c0a8 0101 d431 01bb 4996 02d2 0000 0000\n"

	stdout := io.NopCloser(strings.NewReader(input))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start reading packets
	go receiver.readPackets(ctx, stdout)

	// Wait for packet to be processed
	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 2*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	require.Len(t, logs, 1)
	require.Equal(t, 1, logs[0].LogRecordCount())

	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := logRecord.Attributes()

	protocol, ok := attrs.Get("network.protocol")
	require.True(t, ok)
	require.Equal(t, "IP", protocol.AsString())

	transport, ok := attrs.Get("network.transport")
	require.True(t, ok)
	require.Equal(t, "TCP", transport.AsString())

	srcAddr, ok := attrs.Get("network.src.address")
	require.True(t, ok)
	require.Equal(t, "192.168.1.100", srcAddr.AsString())

	dstAddr, ok := attrs.Get("network.dst.address")
	require.True(t, ok)
	require.Equal(t, "192.168.1.1", dstAddr.AsString())

	srcPort, ok := attrs.Get("network.src.port")
	require.True(t, ok)
	require.Equal(t, int64(54321), srcPort.Int())

	dstPort, ok := attrs.Get("network.dst.port")
	require.True(t, ok)
	require.Equal(t, int64(443), dstPort.Int())

	length, ok := attrs.Get("packet.length")
	require.True(t, ok)
	require.Greater(t, length.Int(), int64(0))
}

func TestReadPackets_Unix_MultiplePackets(t *testing.T) {
	cfg := &Config{Interface: "en0"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	receiver := newReceiver(cfg, logger, sink)

	// Create input with multiple packets
	input := "12:34:56.789012 IP 192.168.1.100.54321 > 192.168.1.1.443: Flags [S]\n" +
		"\t0x0000:  4500 003c 1c46 4000 4006 b1e6 c0a8 0164\n" +
		"\t0x0010:  c0a8 0101 d431 01bb 4996 02d2 0000 0000\n" +
		"13:45:22.123456 IP 10.0.0.5.12345 > 8.8.8.8.53:\n" +
		"\t0x0000:  4500 0039 0000 4000 4011 3cc6 0a00 0005\n" +
		"\t0x0010:  0808 0808 3039 0035 0025 1234 3039 0100\n"

	stdout := io.NopCloser(strings.NewReader(input))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPackets(ctx, stdout)

	// Wait for both packets to be processed
	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 2
	}, 2*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	require.Len(t, logs, 2)
	require.Equal(t, 1, logs[0].LogRecordCount())
	require.Equal(t, 1, logs[1].LogRecordCount())
}

func TestReadPackets_Unix_EmptyInput(t *testing.T) {
	cfg := &Config{Interface: "en0"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	receiver := newReceiver(cfg, logger, sink)

	stdout := io.NopCloser(strings.NewReader(""))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPackets(ctx, stdout)

	// Wait a bit to ensure no packets are processed
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, 0, sink.LogRecordCount())
}

func TestReadPackets_Unix_MalformedPacket(t *testing.T) {
	cfg := &Config{Interface: "en0"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	receiver := newReceiver(cfg, logger, sink)

	// Create input with malformed packet (no valid timestamp)
	input := "not a timestamp line\n" +
		"\t0x0000:  4500 003c 1c46 4000\n" +
		"also not a timestamp\n"

	stdout := io.NopCloser(strings.NewReader(input))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPackets(ctx, stdout)

	// Wait a bit - malformed packets should not be processed
	time.Sleep(200 * time.Millisecond)
	require.Equal(t, 0, sink.LogRecordCount())
}

func TestReadPackets_Unix_LargePacket(t *testing.T) {
	cfg := &Config{Interface: "en0"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	receiver := newReceiver(cfg, logger, sink)

	// Create a large packet with many hex lines
	var sb strings.Builder
	sb.WriteString("12:34:56.789012 IP 192.168.1.100.54321 > 192.168.1.1.443: Flags [S]\n")
	for i := 0; i < 100; i++ {
		sb.WriteString("\t0x")
		sb.WriteString(fmt.Sprintf("%04x", i*16))
		sb.WriteString(":  ")
		for j := 0; j < 8; j++ {
			sb.WriteString("0123 ")
		}
		sb.WriteString("\n")
	}

	stdout := io.NopCloser(strings.NewReader(sb.String()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPackets(ctx, stdout)

	// Wait for packet to be processed
	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 2*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	require.Len(t, logs, 1)
	require.Equal(t, 1, logs[0].LogRecordCount())
}

func TestReadPackets_Unix_PacketWithOnlyTimestamp(t *testing.T) {
	cfg := &Config{Interface: "en0"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	receiver := newReceiver(cfg, logger, sink)

	// Packet with only timestamp line, no hex data
	input := "12:34:56.789012 IP 192.168.1.100.54321 > 192.168.1.1.443: Flags [S]\n" +
		"13:45:22.123456 IP 10.0.0.5.12345 > 8.8.8.8.53:\n"

	stdout := io.NopCloser(strings.NewReader(input))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPackets(ctx, stdout)

	// Wait a bit - packets without hex data should fail parsing
	time.Sleep(200 * time.Millisecond)
	// Should have attempted to process but failed due to missing hex data
	require.Equal(t, 0, sink.LogRecordCount())
}

func TestReadPackets_Unix_IPv6Packet(t *testing.T) {
	cfg := &Config{Interface: "en0"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	receiver := newReceiver(cfg, logger, sink)

	// IPv6 packet
	input := "15:30:45.678901 IP6 2001:db8::1.8080 > 2001:db8::2.80: Flags [S]\n" +
		"\t0x0000:  6000 0000 0014 0640 2001 0db8 0000 0000\n" +
		"\t0x0010:  0000 0000 0000 0001 2001 0db8 0000 0000\n" +
		"\t0x0020:  0000 0000 0000 0002 1f90 0050 3ade 68b1\n"

	stdout := io.NopCloser(strings.NewReader(input))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPackets(ctx, stdout)

	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 2*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := logRecord.Attributes()

	protocol, ok := attrs.Get("network.protocol")
	require.True(t, ok)
	require.Equal(t, "IP6", protocol.AsString())
}

func TestReadPackets_Unix_ICMPPacket(t *testing.T) {
	cfg := &Config{Interface: "en0"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	receiver := newReceiver(cfg, logger, sink)

	// ICMP packet (no ports)
	input := "14:20:30.555555 IP 192.168.1.100 > 192.168.1.1: ICMP echo request\n" +
		"\t0x0000:  4500 0054 0000 4000 4001 b6e8 c0a8 0164\n" +
		"\t0x0010:  c0a8 0101 0800 f7ff 04d2 0001 0001 0203\n"

	stdout := io.NopCloser(strings.NewReader(input))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPackets(ctx, stdout)

	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 2*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := logRecord.Attributes()

	transport, ok := attrs.Get("network.transport")
	require.True(t, ok)
	require.Equal(t, "ICMP", transport.AsString())

	// ICMP should not have ports
	_, srcPortExists := attrs.Get("network.src.port")
	_, dstPortExists := attrs.Get("network.dst.port")
	require.False(t, srcPortExists)
	require.False(t, dstPortExists)
}

func TestReadPackets_Unix_ScannerError(t *testing.T) {
	cfg := &Config{Interface: "en0"}
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	receiver := newReceiver(cfg, logger, sink)

	// Create a reader that will cause an error
	// Use a custom reader that returns an error after some data
	errorReader := &errorReader{
		data: []byte("12:34:56.789012 IP 192.168.1.100.54321 > 192.168.1.1.443: Flags [S]\n"),
		err:  io.ErrUnexpectedEOF,
	}

	stdout := io.NopCloser(errorReader)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go receiver.readPackets(ctx, stdout)

	// Wait for error to be handled
	time.Sleep(200 * time.Millisecond)
	// Should handle error gracefully without panicking
	_ = sink.LogRecordCount()
}

// errorReader is a test utility that returns an error after reading some data
type errorReader struct {
	data []byte
	pos  int
	err  error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	if e.pos >= len(e.data) {
		return 0, e.err
	}
	n = copy(p, e.data[e.pos:])
	e.pos += n
	if e.pos >= len(e.data) {
		return n, e.err
	}
	return n, nil
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
				Interface: "eth0; rm -rf /",
			},
			wantError: "invalid character",
		},
		{
			name: "invalid filter",
			config: &Config{
				Interface: "any",
				Filter:    "tcp port 80 && whoami",
			},
			wantError: "invalid character",
		},
		{
			name: "invalid snaplen",
			config: &Config{
				Interface: "any",
				SnapLen:   10,
			},
			wantError: "snaplen must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := newReceiver(tt.config, zap.NewNop(), consumertest.NewNop())

			err := receiver.Start(context.Background(), componenttest.NewNopHost())
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestStart_WithoutRootPrivileges(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Test requires running without root privileges")
	}

	cfg := &Config{
		Interface: "any",
		SnapLen:   65535,
	}
	receiver := newReceiver(cfg, zap.NewNop(), consumertest.NewNop())

	// Start should succeed even without privileges, but won't capture packets
	err := receiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// Clean up
	_ = receiver.Shutdown(context.Background())
}
