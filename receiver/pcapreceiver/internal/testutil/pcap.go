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
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

// CreatePCAPFile writes a PCAP file with test packets
func CreatePCAPFile(w *pcapgo.Writer) error {
	if err := w.WriteFileHeader(65536, layers.LinkTypeEthernet); err != nil {
		return err
	}

	packets := []gopacket.Packet{
		CreateTCPPacket("192.168.1.1", "192.168.1.2", 12345, 80),
		CreateTCPPacket("192.168.1.2", "192.168.1.1", 80, 12345),
		CreateTCPPacket("10.0.0.1", "10.0.0.2", 54321, 443),
		CreateTCPPacket("10.0.0.2", "10.0.0.1", 443, 54321),
		CreateTCPPacket("172.16.0.1", "172.16.0.2", 33333, 8080),
		CreateUDPPacket("192.168.1.1", "8.8.8.8", 55555, 53),
		CreateUDPPacket("8.8.8.8", "192.168.1.1", 53, 55555),
		CreateUDPPacket("10.0.0.1", "10.0.0.254", 12000, 53),
		CreateICMPPacket("192.168.1.1", "192.168.1.254"),
		CreateICMPPacket("192.168.1.254", "192.168.1.1"),
		CreateIPv6TCPPacket("2001:db8::1", "2001:db8::2", 12345, 80),
	}

	for _, packet := range packets {
		captureInfo := gopacket.CaptureInfo{
			Timestamp:     time.Now(),
			CaptureLength: len(packet.Data()),
			Length:        len(packet.Data()),
		}
		if err := w.WritePacket(captureInfo, packet.Data()); err != nil {
			return err
		}
	}

	return nil
}
