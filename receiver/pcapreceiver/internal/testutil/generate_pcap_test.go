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

package testutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/stretchr/testify/require"
)

// TestGeneratePCAP generates a test PCAP file
// This is not run automatically but can be used to create test data
func TestGeneratePCAP(t *testing.T) {
	t.Skip("Utility test to generate PCAP file, run manually when needed")

	pcapPath := filepath.Join("..", "..", "testdata", "capture.pcap")

	f, err := os.Create(pcapPath)
	require.NoError(t, err)
	defer f.Close()

	w := pcapgo.NewWriter(f)
	require.NoError(t, w.WriteFileHeader(65536, layers.LinkTypeEthernet))

	// Generate test packets
	packets := []gopacket.Packet{
		// TCP packets on port 80
		CreateTCPPacket("192.168.1.1", "192.168.1.2", 12345, 80),
		CreateTCPPacket("192.168.1.2", "192.168.1.1", 80, 12345),
		CreateTCPPacket("10.0.0.1", "10.0.0.2", 54321, 80),

		// TCP packets on port 443
		CreateTCPPacket("10.0.0.2", "10.0.0.1", 443, 54321),
		CreateTCPPacket("172.16.0.1", "172.16.0.2", 33333, 443),

		// UDP packets on port 53 (DNS)
		CreateUDPPacket("192.168.1.1", "8.8.8.8", 55555, 53),
		CreateUDPPacket("8.8.8.8", "192.168.1.1", 53, 55555),
		CreateUDPPacket("10.0.0.1", "10.0.0.254", 12000, 53),

		// ICMP packets
		CreateICMPPacket("192.168.1.1", "192.168.1.254"),
		CreateICMPPacket("192.168.1.254", "192.168.1.1"),

		// IPv6 TCP packet
		CreateIPv6TCPPacket("2001:db8::1", "2001:db8::2", 12345, 80),
	}

	for _, packet := range packets {
		captureInfo := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(packet.Data()),
			Length:        len(packet.Data()),
		}
		require.NoError(t, w.WritePacket(captureInfo, packet.Data()))
	}

	t.Logf("Generated PCAP file: %s with %d packets", pcapPath, len(packets))
}
