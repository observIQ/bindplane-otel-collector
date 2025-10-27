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

//go:build ignore

package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

func main() {
	// Get the testdata directory path
	testdataDir := filepath.Join("..", "..", "testdata")
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		panic(err)
	}

	pcapPath := filepath.Join(testdataDir, "capture.pcap")
	fmt.Printf("Generating PCAP file: %s\n", pcapPath)

	f, err := os.Create(pcapPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := pcapgo.NewWriter(f)
	if err := w.WriteFileHeader(65536, layers.LinkTypeEthernet); err != nil {
		panic(err)
	}

	// Create test packets
	packets := []gopacket.Packet{
		createTCPPacket("192.168.1.1", "192.168.1.2", 12345, 80),
		createTCPPacket("192.168.1.2", "192.168.1.1", 80, 12345),
		createTCPPacket("10.0.0.1", "10.0.0.2", 54321, 443),
	}

	for _, packet := range packets {
		if err := w.WritePacket(packet.Metadata().CaptureInfo, packet.Data()); err != nil {
			panic(err)
		}
	}

	fmt.Printf("Generated PCAP file successfully with %d packets\n", len(packets))
}

// createTCPPacket creates a simple TCP packet for testing
func createTCPPacket(srcIP, dstIP string, srcPort, dstPort uint16) gopacket.Packet {
	// Ethernet layer
	ethLayer := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x0c, 0x29, 0x3e, 0x7f, 0x8a},
		DstMAC:       net.HardwareAddr{0x00, 0x0c, 0x29, 0x3e, 0x7f, 0x8b},
		EthernetType: layers.EthernetTypeIPv4,
	}

	// IP layer
	ipLayer := &layers.IPv4{
		SrcIP:    net.ParseIP(srcIP),
		DstIP:    net.ParseIP(dstIP),
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolTCP,
	}

	// TCP layer
	tcpLayer := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort),
		DstPort: layers.TCPPort(dstPort),
		Seq:     uint32(1),
		Ack:     uint32(1),
		Window:  14600,
		SYN:     true,
	}
	tcpLayer.SetNetworkLayerForChecksum(ipLayer)

	// Serialize layers
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	err := gopacket.SerializeLayers(buf, opts, ethLayer, ipLayer, tcpLayer)
	if err != nil {
		panic(err)
	}

	packetData := buf.Bytes()
	packet := gopacket.NewPacket(packetData, layers.LinkTypeEthernet, gopacket.Default)

	// Set proper capture info
	md := packet.Metadata()
	md.CaptureLength = len(packetData)
	md.Length = len(packetData)

	return packet
}
