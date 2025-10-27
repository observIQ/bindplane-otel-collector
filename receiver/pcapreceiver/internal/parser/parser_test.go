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

package parser

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/observiq/bindplane-otel-collector/receiver/pcapreceiver/internal/testutil"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestNewParser(t *testing.T) {
	parser := NewParser("eth0")
	require.NotNil(t, parser, "parser should not be nil")
}

func TestParsePacket_TCP(t *testing.T) {
	parser := NewParser("eth0")

	// Create a test TCP packet
	packet := testutil.CreateTCPPacket("192.168.1.1", "192.168.1.2", 12345, 80)
	require.NotNil(t, packet)

	// Parse to logs
	logs := parser.Parse(packet)
	require.Equal(t, 1, logs.ResourceLogs().Len(), "should have 1 resource log")

	rl := logs.ResourceLogs().At(0)
	require.Equal(t, 1, rl.ScopeLogs().Len(), "should have 1 scope log")

	sl := rl.ScopeLogs().At(0)
	require.Equal(t, 1, sl.LogRecords().Len(), "should have 1 log record")

	lr := sl.LogRecords().At(0)

	// Verify attributes
	attrs := lr.Attributes()

	// Interface name
	val, ok := attrs.Get("interface")
	require.True(t, ok, "interface attribute should exist")
	require.Equal(t, "eth0", val.Str())

	// Packet size
	val, ok = attrs.Get("packet.size")
	require.True(t, ok, "packet.size attribute should exist")
	require.Greater(t, val.Int(), int64(0))

	// Timestamp
	val, ok = attrs.Get("packet.timestamp")
	require.True(t, ok, "packet.timestamp attribute should exist")
	require.NotEmpty(t, val.Str())

	// Network layer
	val, ok = attrs.Get("network.protocol")
	require.True(t, ok, "network.protocol attribute should exist")
	require.Equal(t, "IPv4", val.Str())

	val, ok = attrs.Get("network.src_addr")
	require.True(t, ok, "network.src_addr attribute should exist")
	require.Equal(t, "192.168.1.1", val.Str())

	val, ok = attrs.Get("network.dst_addr")
	require.True(t, ok, "network.dst_addr attribute should exist")
	require.Equal(t, "192.168.1.2", val.Str())

	// Transport layer
	val, ok = attrs.Get("transport.protocol")
	require.True(t, ok, "transport.protocol attribute should exist")
	require.Equal(t, "TCP", val.Str())

	val, ok = attrs.Get("transport.src_port")
	require.True(t, ok, "transport.src_port attribute should exist")
	require.Equal(t, int64(12345), val.Int())

	val, ok = attrs.Get("transport.dst_port")
	require.True(t, ok, "transport.dst_port attribute should exist")
	require.Equal(t, int64(80), val.Int())

	// Application protocol (port-based detection)
	val, ok = attrs.Get("application.protocol")
	require.True(t, ok, "application.protocol attribute should exist")
	require.Equal(t, "HTTP", val.Str())

	// Verify body contains packet data (hex-encoded)
	body := lr.Body().Str()
	require.NotEmpty(t, body, "body should not be empty")
	require.Greater(t, len(body), 0)
}

func TestParsePacket_UDP(t *testing.T) {
	parser := NewParser("lo")

	// Create a test UDP packet
	packet := testutil.CreateUDPPacket("10.0.0.1", "10.0.0.2", 5353, 53)
	require.NotNil(t, packet)

	logs := parser.Parse(packet)
	lr := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := lr.Attributes()

	// Transport protocol should be UDP
	val, ok := attrs.Get("transport.protocol")
	require.True(t, ok)
	require.Equal(t, "UDP", val.Str())

	// Ports
	val, ok = attrs.Get("transport.src_port")
	require.True(t, ok)
	require.Equal(t, int64(5353), val.Int())

	val, ok = attrs.Get("transport.dst_port")
	require.True(t, ok)
	require.Equal(t, int64(53), val.Int())

	// Application protocol (DNS)
	val, ok = attrs.Get("application.protocol")
	require.True(t, ok)
	require.Equal(t, "DNS", val.Str())
}

func TestParsePacket_ICMP(t *testing.T) {
	parser := NewParser("eth1")

	// Create a test ICMP packet
	packet := testutil.CreateICMPPacket("172.16.0.1", "172.16.0.2")
	require.NotNil(t, packet)

	logs := parser.Parse(packet)
	lr := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := lr.Attributes()

	// Transport protocol should be ICMP
	val, ok := attrs.Get("transport.protocol")
	require.True(t, ok)
	require.Equal(t, "ICMP", val.Str())

	// ICMP packets don't have ports
	_, ok = attrs.Get("transport.src_port")
	require.False(t, ok, "ICMP packets should not have src_port")

	_, ok = attrs.Get("transport.dst_port")
	require.False(t, ok, "ICMP packets should not have dst_port")

	// Network addresses should be present
	val, ok = attrs.Get("network.src_addr")
	require.True(t, ok)
	require.Equal(t, "172.16.0.1", val.Str())
}

func TestParsePacket_LinkLayer(t *testing.T) {
	parser := NewParser("eth0")

	packet := testutil.CreateTCPPacket("192.168.1.1", "192.168.1.2", 443, 443)
	require.NotNil(t, packet)

	logs := parser.Parse(packet)
	lr := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := lr.Attributes()

	// Link layer MAC addresses
	val, ok := attrs.Get("link.src_addr")
	require.True(t, ok, "link.src_addr attribute should exist")
	require.NotEmpty(t, val.Str())

	val, ok = attrs.Get("link.dst_addr")
	require.True(t, ok, "link.dst_addr attribute should exist")
	require.NotEmpty(t, val.Str())
}

func TestParsePacket_FromPCAP(t *testing.T) {
	// Read packets from the test PCAP file
	pcapPath := filepath.Join("..", "..", "testdata", "capture.pcap")
	_, err := os.Stat(pcapPath)
	if os.IsNotExist(err) {
		t.Skipf("PCAP file not found at %s", pcapPath)
	}

	handle, err := pcap.OpenOffline(pcapPath)
	require.NoError(t, err)
	defer handle.Close()

	parser := NewParser("test")
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	packetCount := 0
	for packet := range packetSource.Packets() {
		logs := parser.Parse(packet)
		require.Equal(t, 1, logs.ResourceLogs().Len())

		lr := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
		attrs := lr.Attributes()

		// All packets should have basic attributes
		_, ok := attrs.Get("interface")
		require.True(t, ok, "packet %d should have interface attribute", packetCount)

		_, ok = attrs.Get("packet.size")
		require.True(t, ok, "packet %d should have packet.size attribute", packetCount)

		_, ok = attrs.Get("packet.timestamp")
		require.True(t, ok, "packet %d should have packet.timestamp attribute", packetCount)

		// Body should not be empty
		require.NotEmpty(t, lr.Body().Str(), "packet %d should have non-empty body", packetCount)

		packetCount++
	}

	require.Greater(t, packetCount, 0, "should have parsed at least one packet")
}

func TestParsePacket_Timestamp(t *testing.T) {
	parser := NewParser("eth0")
	packet := testutil.CreateTCPPacket("192.168.1.1", "192.168.1.2", 8080, 80)

	logs := parser.Parse(packet)
	lr := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	// Verify timestamp is set on the log record
	ts := lr.Timestamp()
	require.NotEqual(t, pcommon.Timestamp(0), ts, "timestamp should be set")

	// Convert to time.Time and verify it's recent (within last minute)
	tsTime := time.Unix(0, int64(ts))
	require.WithinDuration(t, time.Now(), tsTime, 1*time.Minute)
}

func TestParsePacket_NilPacket(t *testing.T) {
	parser := NewParser("eth0")

	// Should handle nil packet gracefully
	logs := parser.Parse(nil)
	require.NotNil(t, logs)
	require.Equal(t, 0, logs.ResourceLogs().Len(), "nil packet should result in empty logs")
}

func TestParsePacket_PartialLayers(t *testing.T) {
	parser := NewParser("eth0")

	// Create a packet with only Ethernet layer (no IP)
	// Use ARP EthernetType to avoid gopacket trying to decode IP
	ethLayer := &layers.Ethernet{
		SrcMAC:       []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := gopacket.SerializeLayers(buf, opts, ethLayer)
	require.NoError(t, err)

	packet := gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.NoCopy)

	logs := parser.Parse(packet)
	require.Equal(t, 1, logs.ResourceLogs().Len())

	lr := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := lr.Attributes()

	// Should have link layer attributes
	_, ok := attrs.Get("link.src_addr")
	require.True(t, ok)

	// Should not have network layer attributes
	_, ok = attrs.Get("network.src_addr")
	require.False(t, ok, "should not have network layer when only Ethernet is present")
}

func TestDetectApplicationProtocol(t *testing.T) {
	testCases := []struct {
		port     uint16
		protocol string
		expected string
	}{
		{80, "TCP", "HTTP"},
		{443, "TCP", "HTTPS"},
		{53, "UDP", "DNS"},
		{22, "TCP", "SSH"},
		{21, "TCP", "FTP"},
		{25, "TCP", "SMTP"},
		{3306, "TCP", "MySQL"},
		{5432, "TCP", "PostgreSQL"},
		{6379, "TCP", "Redis"},
		{9999, "TCP", "Unknown"}, // Non-standard port
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			detected := detectApplicationProtocol(tc.port, tc.protocol)
			require.Equal(t, tc.expected, detected)
		})
	}
}
