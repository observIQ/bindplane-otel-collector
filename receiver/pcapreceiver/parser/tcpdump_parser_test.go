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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePacket_TCP(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "tcp_packet.txt"))
	require.NoError(t, err)

	lines := strings.Split(string(data), "\n")
	packet, err := ParsePacket(lines)
	require.NoError(t, err)
	require.NotNil(t, packet)

	// Verify timestamp parsing
	require.Equal(t, 12, packet.Timestamp.Hour())
	require.Equal(t, 34, packet.Timestamp.Minute())
	require.Equal(t, 56, packet.Timestamp.Second())

	// Verify protocol information
	require.Equal(t, ProtocolIP, packet.Protocol)
	require.Equal(t, TransportTCP, packet.Transport)

	// Verify addresses
	require.Equal(t, "192.168.1.100", packet.SrcAddress)
	require.Equal(t, "192.168.1.1", packet.DstAddress)

	// Verify ports
	require.Equal(t, 54321, packet.SrcPort)
	require.Equal(t, 443, packet.DstPort)

	// Verify hex data (should be concatenated without spaces/offsets)
	require.NotEmpty(t, packet.HexData)
	require.Contains(t, packet.HexData, "4500003c")
	require.NotContains(t, packet.HexData, "0x0000:")
	require.NotContains(t, packet.HexData, " ")

	// Verify length
	require.Greater(t, packet.Length, 0)
}

func TestParsePacket_UDP(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "udp_packet.txt"))
	require.NoError(t, err)

	lines := strings.Split(string(data), "\n")
	packet, err := ParsePacket(lines)
	require.NoError(t, err)
	require.NotNil(t, packet)

	// Verify protocol information
	require.Equal(t, ProtocolIP, packet.Protocol)
	require.Equal(t, TransportUDP, packet.Transport)

	// Verify addresses
	require.Equal(t, "10.0.0.5", packet.SrcAddress)
	require.Equal(t, "8.8.8.8", packet.DstAddress)

	// Verify ports
	require.Equal(t, 12345, packet.SrcPort)
	require.Equal(t, 53, packet.DstPort)

	// Verify hex data
	require.NotEmpty(t, packet.HexData)
	require.Greater(t, packet.Length, 0)
}

func TestParsePacket_ICMP(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "icmp_packet.txt"))
	require.NoError(t, err)

	lines := strings.Split(string(data), "\n")
	packet, err := ParsePacket(lines)
	require.NoError(t, err)
	require.NotNil(t, packet)

	// Verify protocol information
	require.Equal(t, ProtocolIP, packet.Protocol)
	require.Equal(t, TransportICMP, packet.Transport)

	// Verify addresses
	require.Equal(t, "192.168.1.100", packet.SrcAddress)
	require.Equal(t, "192.168.1.1", packet.DstAddress)

	// ICMP should not have ports
	require.Equal(t, 0, packet.SrcPort)
	require.Equal(t, 0, packet.DstPort)

	// Verify hex data
	require.NotEmpty(t, packet.HexData)
	require.Greater(t, packet.Length, 0)
}

func TestParsePacket_IPv6(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "ipv6_packet.txt"))
	require.NoError(t, err)

	lines := strings.Split(string(data), "\n")
	packet, err := ParsePacket(lines)
	require.NoError(t, err)
	require.NotNil(t, packet)

	// Verify protocol information
	require.Equal(t, ProtocolIP6, packet.Protocol)
	require.Equal(t, TransportTCP, packet.Transport)

	// Verify IPv6 addresses
	require.Contains(t, packet.SrcAddress, "2001:db8")
	require.Contains(t, packet.DstAddress, "2001:db8")

	// Verify ports
	require.Equal(t, 8080, packet.SrcPort)
	require.Equal(t, 80, packet.DstPort)

	// Verify hex data
	require.NotEmpty(t, packet.HexData)
	require.Greater(t, packet.Length, 0)
}

func TestParsePacket_Malformed(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "malformed.txt"))
	require.NoError(t, err)

	lines := strings.Split(string(data), "\n")
	packet, err := ParsePacket(lines)

	// Should return an error for malformed input
	require.Error(t, err)
	require.Nil(t, packet)
}

func TestParsePacket_EmptyInput(t *testing.T) {
	lines := []string{}
	packet, err := ParsePacket(lines)

	require.Error(t, err)
	require.Nil(t, packet)
}

func TestParsePacket_OnlyHeaderLine(t *testing.T) {
	lines := []string{
		"12:34:56.789012 IP 192.168.1.100.54321 > 192.168.1.1.443: Flags [S], seq 1234567890",
	}
	packet, err := ParsePacket(lines)

	// Should error because there's no hex data
	require.Error(t, err)
	require.Nil(t, packet)
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHour int
		wantMin  int
		wantSec  int
		wantErr  bool
	}{
		{
			name:     "valid timestamp",
			input:    "12:34:56.789012",
			wantHour: 12,
			wantMin:  34,
			wantSec:  56,
			wantErr:  false,
		},
		{
			name:     "midnight",
			input:    "00:00:00.000000",
			wantHour: 0,
			wantMin:  0,
			wantSec:  0,
			wantErr:  false,
		},
		{
			name:     "end of day",
			input:    "23:59:59.999999",
			wantHour: 23,
			wantMin:  59,
			wantSec:  59,
			wantErr:  false,
		},
		{
			name:    "invalid format",
			input:   "not a timestamp",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := parseTimestamp(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantHour, ts.Hour())
				require.Equal(t, tt.wantMin, ts.Minute())
				require.Equal(t, tt.wantSec, ts.Second())
			}
		})
	}
}

func TestParseHeaderLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantProto   string
		wantTransp  string
		wantSrcAddr string
		wantDstAddr string
		wantSrcPort int
		wantDstPort int
		wantErr     bool
	}{
		{
			name:        "TCP with ports",
			line:        "12:34:56.789012 IP 192.168.1.100.54321 > 192.168.1.1.443: Flags [S]",
			wantProto:   ProtocolIP,
			wantTransp:  TransportTCP,
			wantSrcAddr: "192.168.1.100",
			wantDstAddr: "192.168.1.1",
			wantSrcPort: 54321,
			wantDstPort: 443,
			wantErr:     false,
		},
		{
			name:        "UDP with ports",
			line:        "13:45:22.123456 IP 10.0.0.5.12345 > 8.8.8.8.53: UDP",
			wantProto:   ProtocolIP,
			wantTransp:  TransportUDP,
			wantSrcAddr: "10.0.0.5",
			wantDstAddr: "8.8.8.8",
			wantSrcPort: 12345,
			wantDstPort: 53,
			wantErr:     false,
		},
		{
			name:        "ICMP without ports",
			line:        "14:20:30.555555 IP 192.168.1.100 > 192.168.1.1: ICMP echo request",
			wantProto:   ProtocolIP,
			wantTransp:  TransportICMP,
			wantSrcAddr: "192.168.1.100",
			wantDstAddr: "192.168.1.1",
			wantSrcPort: 0,
			wantDstPort: 0,
			wantErr:     false,
		},
		{
			name:        "IPv6 with ports",
			line:        "15:30:45.678901 IP6 2001:db8::1.8080 > 2001:db8::2.80: Flags [S]",
			wantProto:   ProtocolIP6,
			wantTransp:  TransportTCP,
			wantSrcAddr: "2001:db8::1",
			wantDstAddr: "2001:db8::2",
			wantSrcPort: 8080,
			wantDstPort: 80,
			wantErr:     false,
		},
		{
			name:    "malformed line",
			line:    "not a valid tcpdump line",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseHeaderLine(tt.line)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantProto, info.Protocol)
				require.Equal(t, tt.wantTransp, info.Transport)
				require.Equal(t, tt.wantSrcAddr, info.SrcAddress)
				require.Equal(t, tt.wantDstAddr, info.DstAddress)
				require.Equal(t, tt.wantSrcPort, info.SrcPort)
				require.Equal(t, tt.wantDstPort, info.DstPort)
			}
		})
	}
}

func TestParseHexLines(t *testing.T) {
	tests := []struct {
		name    string
		lines   []string
		wantHex string
		wantLen int
		wantErr bool
	}{
		{
			name: "valid hex lines",
			lines: []string{
				"\t0x0000:  4500 003c 1c46 4000 4006 b1e6 c0a8 0164",
				"\t0x0010:  c0a8 0101 d431 01bb 4996 02d2 0000 0000",
			},
			wantHex: "4500003c1c4640004006b1e6c0a80164c0a80101d43101bb499602d200000000",
			wantLen: 32,
			wantErr: false,
		},
		{
			name: "hex line with varying spacing",
			lines: []string{
				"\t0x0000:  4500 003c 1c46 4000",
				"\t0x0008:  4006 b1e6",
			},
			wantHex: "4500003c1c4640004006b1e6",
			wantLen: 12,
			wantErr: false,
		},
		{
			name:    "no hex lines",
			lines:   []string{},
			wantHex: "",
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hex, length, err := parseHexLines(tt.lines)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantHex, hex)
				require.Equal(t, tt.wantLen, length)
			}
		})
	}
}
