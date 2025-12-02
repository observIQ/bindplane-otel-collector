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
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/observiq/bindplane-otel-collector/receiver/pcapreceiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"
)

// newTestReceiver creates a test receiver with default settings.
// If logger is nil, uses zap.NewNop(). If consumer is nil, uses consumertest.NewNop().
func newTestReceiver(t *testing.T, cfg *Config, logger *zap.Logger, con consumer.Logs) *pcapReceiver {
	t.Helper()

	if logger == nil {
		logger = zap.NewNop()
	}
	if con == nil {
		con = consumertest.NewNop()
	}

	settings := receivertest.NewNopSettings(metadata.Type)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	receiver, err := newReceiver(settings, cfg, logger, con, tb)
	require.NoError(t, err)
	require.NotNil(t, receiver)

	return receiver
}

func TestNewReceiver(t *testing.T) {
	cfg := &Config{
		Interface:       "en0",
		Filter:          "tcp port 443",
		SnapLen:         65535,
		Promiscuous:     true,
		ParseAttributes: true,
	}
	logger := zap.NewNop()
	consumer := consumertest.NewNop()
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	settings := receivertest.NewNopSettings(typ)
	receiver, err := newReceiver(settings, cfg, logger, consumer, tb)
	require.NoError(t, err)
	require.NotNil(t, receiver)
	require.Equal(t, cfg, receiver.config)
	require.Equal(t, logger, receiver.logger)
	require.Equal(t, consumer, receiver.consumer)
}

func TestIsTimestampLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "valid timestamp",
			line: "12:34:56.789012 IP 192.168.1.1 > 192.168.1.2",
			want: true,
		},
		{
			name: "valid timestamp at midnight",
			line: "00:00:00.000000 IP 10.0.0.1 > 10.0.0.2",
			want: true,
		},
		{
			name: "hex line",
			line: "\t0x0000:  4500 003c 1c46 4000",
			want: false,
		},
		{
			name: "empty line",
			line: "",
			want: false,
		},
		{
			name: "short line",
			line: "12:34",
			want: false,
		},
		{
			name: "invalid format",
			line: "not a timestamp",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTimestampLine(tt.line)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestProcessPacket(t *testing.T) {
	cfg := &Config{Interface: "en0", ParseAttributes: true}
	sink := &consumertest.LogsSink{}
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, zap.NewNop(), sink, tb)
	require.NoError(t, err)

	lines := []string{
		"12:34:56.789012 IP 192.168.1.100.54321 > 192.168.1.1.443: Flags [S]",
		"\t0x0000:  4500 003c 1c46 4000 4006 b1e6 c0a8 0164",
		"\t0x0010:  c0a8 0101 d431 01bb 4996 02d2 0000 0000",
	}

	ctx := context.Background()
	receiver.processPacket(ctx, lines)

	// Consumer should have received one log
	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 1*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	require.Len(t, logs, 1)
	require.Equal(t, 1, logs[0].LogRecordCount())

	// Verify log structure
	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	// Check body starts with 0x
	body := logRecord.Body().AsString()
	require.True(t, len(body) > 2, "Body should not be empty")
	require.Equal(t, "0x", body[:2], "Body should start with 0x prefix")

	// Check attributes
	attrs := logRecord.Attributes()
	protocol, ok := attrs.Get("network.type")
	require.True(t, ok)
	require.Equal(t, "IP", protocol.AsString())

	interfaceName, ok := attrs.Get("network.interface.name")
	require.True(t, ok)
	require.Equal(t, "en0", interfaceName.AsString())

	transport, ok := attrs.Get("network.transport")
	require.True(t, ok)
	require.Equal(t, "TCP", transport.AsString())

	srcAddr, ok := attrs.Get("source.address")
	require.True(t, ok)
	require.Equal(t, "192.168.1.100", srcAddr.AsString())

	dstAddr, ok := attrs.Get("destination.address")
	require.True(t, ok)
	require.Equal(t, "192.168.1.1", dstAddr.AsString())

	srcPort, ok := attrs.Get("source.port")
	require.True(t, ok)
	require.Equal(t, int64(54321), srcPort.Int())

	dstPort, ok := attrs.Get("destination.port")
	require.True(t, ok)
	require.Equal(t, int64(443), dstPort.Int())

	length, ok := attrs.Get("packet.length")
	require.True(t, ok)
	require.Greater(t, length.Int(), int64(0))
}

func TestProcessPacket_InvalidPacket(t *testing.T) {
	cfg := &Config{Interface: "en0"}
	sink := &consumertest.LogsSink{}
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, zap.NewNop(), sink, tb)
	require.NoError(t, err)

	// Invalid packet lines
	lines := []string{
		"not a valid packet",
		"random text",
	}

	ctx := context.Background()
	receiver.processPacket(ctx, lines)

	// Consumer should not have received any logs
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, 0, sink.LogRecordCount())
}

func TestProcessPacket_ICMPNoPort(t *testing.T) {
	cfg := &Config{Interface: "en0", ParseAttributes: true}
	sink := &consumertest.LogsSink{}
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, zap.NewNop(), sink, tb)
	require.NoError(t, err)

	lines := []string{
		"14:20:30.555555 IP 192.168.1.100 > 192.168.1.1: ICMP echo request",
		"\t0x0000:  4500 0054 0000 4000 4001 b6e8 c0a8 0164",
		"\t0x0010:  c0a8 0101 0800 f7ff 04d2 0001 0001 0203",
	}

	ctx := context.Background()
	receiver.processPacket(ctx, lines)

	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 1*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	// Check transport is ICMP
	attrs := logRecord.Attributes()
	interfaceName, ok := attrs.Get("network.interface.name")
	require.True(t, ok)
	require.Equal(t, "en0", interfaceName.AsString())

	transport, ok := attrs.Get("network.transport")
	require.True(t, ok)
	require.Equal(t, "ICMP", transport.AsString())

	// Verify ports are not set (ICMP doesn't have ports)
	_, srcPortExists := attrs.Get("source.port")
	_, dstPortExists := attrs.Get("destination.port")
	require.False(t, srcPortExists, "ICMP should not have source port")
	require.False(t, dstPortExists, "ICMP should not have destination port")
}

func TestShutdown(t *testing.T) {
	cfg := &Config{Interface: "en0"}
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, zap.NewNop(), consumertest.NewNop(), tb)
	require.NoError(t, err)

	// Test shutdown without starting
	err = receiver.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestProcessPacket_IPv6(t *testing.T) {
	cfg := &Config{Interface: "en0", ParseAttributes: true}
	sink := &consumertest.LogsSink{}
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, zap.NewNop(), sink, tb)
	require.NoError(t, err)

	lines := []string{
		"15:30:45.678901 IP6 2001:db8::1.8080 > 2001:db8::2.80: Flags [S]",
		"\t0x0000:  6000 0000 0014 0640 2001 0db8 0000 0000",
		"\t0x0010:  0000 0000 0000 0001 2001 0db8 0000 0000",
		"\t0x0020:  0000 0000 0000 0002 1f90 0050 3ade 68b1",
	}

	ctx := context.Background()
	receiver.processPacket(ctx, lines)

	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 1*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	attrs := logRecord.Attributes()

	// Check IPv6 protocol
	protocol, ok := attrs.Get("network.type")
	require.True(t, ok)
	require.Equal(t, "IP6", protocol.AsString())

	interfaceName, ok := attrs.Get("network.interface.name")
	require.True(t, ok)
	require.Equal(t, "en0", interfaceName.AsString())

	// Check IPv6 addresses
	srcAddr, ok := attrs.Get("source.address")
	require.True(t, ok)
	require.Contains(t, srcAddr.AsString(), "2001:db8")

	dstAddr, ok := attrs.Get("destination.address")
	require.True(t, ok)
	require.Contains(t, dstAddr.AsString(), "2001:db8")

	// Check ports
	srcPort, ok := attrs.Get("source.port")
	require.True(t, ok)
	require.Equal(t, int64(8080), srcPort.Int())

	dstPort, ok := attrs.Get("destination.port")
	require.True(t, ok)
	require.Equal(t, int64(80), dstPort.Int())
}

func TestDefaultSnapLen(t *testing.T) {
	cfg := &Config{
		Interface: "en0",
		SnapLen:   0, // Not specified, should use default
	}
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, zap.NewNop(), consumertest.NewNop(), tb)
	require.NoError(t, err)

	require.Equal(t, 0, receiver.config.SnapLen, "Config should preserve original 0 value")
	// Default is applied in Start() when building capture command
}

func TestProcessPacket_UDP(t *testing.T) {
	cfg := &Config{Interface: "en0", ParseAttributes: true}
	sink := &consumertest.LogsSink{}
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, zap.NewNop(), sink, tb)
	require.NoError(t, err)

	lines := []string{
		"13:45:22.123456 IP 10.0.0.5.12345 > 8.8.8.8.53: 12345+ A? example.com. (29)",
		"\t0x0000:  4500 0039 0000 4000 4011 3cc6 0a00 0005",
		"\t0x0010:  0808 0808 3039 0035 0025 1234 3039 0100",
	}

	ctx := context.Background()
	receiver.processPacket(ctx, lines)

	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 1*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	attrs := logRecord.Attributes()

	// Check UDP transport
	interfaceName, ok := attrs.Get("network.interface.name")
	require.True(t, ok)
	require.Equal(t, "en0", interfaceName.AsString())

	transport, ok := attrs.Get("network.transport")
	require.True(t, ok)
	require.Equal(t, "UDP", transport.AsString())

	// Check DNS port
	dstPort, ok := attrs.Get("destination.port")
	require.True(t, ok)
	require.Equal(t, int64(53), dstPort.Int())
}

func TestPacketInfo_ToLogAttributes(t *testing.T) {
	// Test that PacketInfo is correctly converted to log attributes
	cfg := &Config{Interface: "en0", ParseAttributes: true}
	sink := &consumertest.LogsSink{}
	settings := receivertest.NewNopSettings(typ)
	tb, err := metadata.NewTelemetryBuilder(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	receiver, err := newReceiver(settings, cfg, zap.NewNop(), sink, tb)
	require.NoError(t, err)

	// Manually construct lines from PacketInfo for testing
	lines := []string{
		"12:34:56.789012 IP 192.168.1.100.54321 > 192.168.1.1.443: Flags [S]",
		"\t0x0000:  4500 003c 1c46 4000 4006 b1e6 c0a8 0164",
	}

	ctx := context.Background()
	receiver.processPacket(ctx, lines)

	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 1*time.Second, 10*time.Millisecond)

	logs := sink.AllLogs()
	logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	// Verify all expected attributes are present
	attrs := logRecord.Attributes()
	require.Equal(t, 8, attrs.Len(), "Should have 8 attributes")

	// Verify attribute types
	protocol, _ := attrs.Get("network.type")
	require.Equal(t, "IP", protocol.AsString())

	interfaceName, _ := attrs.Get("network.interface.name")
	require.Equal(t, "en0", interfaceName.AsString())

	transport, _ := attrs.Get("network.transport")
	require.Equal(t, "TCP", transport.AsString())
}

// TestGolden tests that packet parsing produces the expected log output
// Input files are tcpdump text format, expected files are plog YAML format
func TestGolden(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{name: "tcp_https"},
		{name: "udp_dns"},
		{name: "icmp_echo"},
		{name: "ipv6_tcp"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Interface:       "eth0",
				ParseAttributes: true,
			}
			sink := &consumertest.LogsSink{}
			receiver := newTestReceiver(t, cfg, nil, sink)

			// Read input packet data
			inputPath := filepath.Join("testdata", "golden", "input", tc.name+".txt")
			content, err := os.ReadFile(inputPath)
			require.NoError(t, err)

			// Parse and process the packet (normalize line endings for Windows)
			contentStr := strings.ReplaceAll(string(content), "\r\n", "\n")
			lines := strings.Split(contentStr, "\n")
			ctx := context.Background()
			receiver.processPacket(ctx, lines)

			// Verify log was created
			require.Equal(t, 1, sink.LogRecordCount(), "Expected exactly 1 log record")

			// Read expected output
			expectedPath := filepath.Join("testdata", "golden", "expected", tc.name+".txt.yaml")
			expectedLogs, err := golden.ReadLogs(expectedPath)
			require.NoError(t, err)

			// Compare logs (ignoring timestamps which are dynamic)
			err = plogtest.CompareLogs(expectedLogs, sink.AllLogs()[0],
				plogtest.IgnoreTimestamp(),
				plogtest.IgnoreObservedTimestamp(),
			)
			require.NoError(t, err)
		})
	}
}

// TestGolden_NoAttributes tests packet parsing with ParseAttributes disabled
func TestGolden_NoAttributes(t *testing.T) {
	cfg := &Config{
		Interface:       "eth0",
		ParseAttributes: false,
	}
	sink := &consumertest.LogsSink{}
	receiver := newTestReceiver(t, cfg, nil, sink)

	// Read TCP packet input
	inputPath := filepath.Join("testdata", "golden", "input", "tcp_https.txt")
	content, err := os.ReadFile(inputPath)
	require.NoError(t, err)

	// Parse and process the packet (normalize line endings for Windows)
	contentStr := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(contentStr, "\n")
	ctx := context.Background()
	receiver.processPacket(ctx, lines)

	// Verify log was created
	require.Equal(t, 1, sink.LogRecordCount(), "Expected exactly 1 log record")

	// Verify body is set but no attributes
	logs := sink.AllLogs()[0]
	logRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	// Body should start with 0x
	body := logRecord.Body().AsString()
	require.True(t, strings.HasPrefix(body, "0x"), "Body should start with 0x prefix")

	// No attributes should be set when ParseAttributes is false
	require.Equal(t, 0, logRecord.Attributes().Len(), "No attributes should be set when ParseAttributes is false")
}
