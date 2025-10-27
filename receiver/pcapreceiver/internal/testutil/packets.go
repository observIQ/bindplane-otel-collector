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
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// CreateTCPPacket creates a TCP packet for testing
func CreateTCPPacket(srcIP, dstIP string, srcPort, dstPort uint16) gopacket.Packet {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	// Create layers
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       net.HardwareAddr{0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb},
		EthernetType: layers.EthernetTypeIPv4,
	}

	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    net.ParseIP(srcIP).To4(),
		DstIP:    net.ParseIP(dstIP).To4(),
	}

	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort),
		DstPort: layers.TCPPort(dstPort),
		Seq:     1000,
		Ack:     2000,
		Window:  65535,
		SYN:     true,
	}
	tcp.SetNetworkLayerForChecksum(ip)

	payload := gopacket.Payload([]byte("test tcp payload"))

	err := gopacket.SerializeLayers(buf, opts, eth, ip, tcp, payload)
	if err != nil {
		panic(err)
	}

	packet := gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
	packet.Metadata().Timestamp = time.Now()
	packet.Metadata().CaptureLength = len(buf.Bytes())
	packet.Metadata().Length = len(buf.Bytes())

	return packet
}

// CreateUDPPacket creates a UDP packet for testing
func CreateUDPPacket(srcIP, dstIP string, srcPort, dstPort uint16) gopacket.Packet {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       net.HardwareAddr{0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb},
		EthernetType: layers.EthernetTypeIPv4,
	}

	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    net.ParseIP(srcIP).To4(),
		DstIP:    net.ParseIP(dstIP).To4(),
	}

	udp := &layers.UDP{
		SrcPort: layers.UDPPort(srcPort),
		DstPort: layers.UDPPort(dstPort),
	}
	udp.SetNetworkLayerForChecksum(ip)

	payload := gopacket.Payload([]byte("test udp payload"))

	err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, payload)
	if err != nil {
		panic(err)
	}

	packet := gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
	packet.Metadata().Timestamp = time.Now()
	packet.Metadata().CaptureLength = len(buf.Bytes())
	packet.Metadata().Length = len(buf.Bytes())

	return packet
}

// CreateICMPPacket creates an ICMP packet for testing
func CreateICMPPacket(srcIP, dstIP string) gopacket.Packet {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       net.HardwareAddr{0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb},
		EthernetType: layers.EthernetTypeIPv4,
	}

	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    net.ParseIP(srcIP).To4(),
		DstIP:    net.ParseIP(dstIP).To4(),
	}

	icmp := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
		Id:       1,
		Seq:      1,
	}

	payload := gopacket.Payload([]byte("test icmp payload"))

	err := gopacket.SerializeLayers(buf, opts, eth, ip, icmp, payload)
	if err != nil {
		panic(err)
	}

	packet := gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
	packet.Metadata().Timestamp = time.Now()
	packet.Metadata().CaptureLength = len(buf.Bytes())
	packet.Metadata().Length = len(buf.Bytes())

	return packet
}

// CreateIPv6TCPPacket creates an IPv6 TCP packet for testing
func CreateIPv6TCPPacket(srcIP, dstIP string, srcPort, dstPort uint16) gopacket.Packet {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       net.HardwareAddr{0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb},
		EthernetType: layers.EthernetTypeIPv6,
	}

	ip := &layers.IPv6{
		Version:    6,
		HopLimit:   64,
		NextHeader: layers.IPProtocolTCP,
		SrcIP:      net.ParseIP(srcIP),
		DstIP:      net.ParseIP(dstIP),
	}

	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort),
		DstPort: layers.TCPPort(dstPort),
		Seq:     1000,
		Ack:     2000,
		Window:  65535,
		SYN:     true,
	}
	tcp.SetNetworkLayerForChecksum(ip)

	payload := gopacket.Payload([]byte("test ipv6 tcp payload"))

	err := gopacket.SerializeLayers(buf, opts, eth, ip, tcp, payload)
	if err != nil {
		panic(err)
	}

	packet := gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
	packet.Metadata().Timestamp = time.Now()
	packet.Metadata().CaptureLength = len(buf.Bytes())
	packet.Metadata().Length = len(buf.Bytes())

	return packet
}
