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

package parser

import (
	"encoding/hex"
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// ParsePcapgoPacket parses a packet from binary PCAP data using pcapgo
// data is the raw packet bytes from pcapgo
// ci is the capture info containing timestamp
func ParsePcapgoPacket(data []byte, ci gopacket.CaptureInfo) (*PacketInfo, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty packet data")
	}

	info := &PacketInfo{
		Timestamp: ci.Timestamp,
		Length:    len(data),
	}

	// Convert binary data to hex string
	info.HexData = hex.EncodeToString(data)

	// Parse packet using gopacket
	packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.Default)

	// Extract network layer protocol
	var ipLayer gopacket.Layer
	var ipv4Layer *layers.IPv4
	var ipv6Layer *layers.IPv6
	var arpLayer *layers.ARP

	if ipv4 := packet.Layer(layers.LayerTypeIPv4); ipv4 != nil {
		ipv4Layer = ipv4.(*layers.IPv4)
		ipLayer = ipv4
		info.Protocol = ProtocolIP
		info.SrcAddress = ipv4Layer.SrcIP.String()
		info.DstAddress = ipv4Layer.DstIP.String()
	} else if ipv6 := packet.Layer(layers.LayerTypeIPv6); ipv6 != nil {
		ipv6Layer = ipv6.(*layers.IPv6)
		ipLayer = ipv6
		info.Protocol = ProtocolIPv6
		info.SrcAddress = ipv6Layer.SrcIP.String()
		info.DstAddress = ipv6Layer.DstIP.String()
	} else if arp := packet.Layer(layers.LayerTypeARP); arp != nil {
		arpLayer = arp.(*layers.ARP)
		info.Protocol = ProtocolARP
		// ARP addresses are []byte, convert to IP string
		if len(arpLayer.SourceProtAddress) == 4 {
			info.SrcAddress = fmt.Sprintf("%d.%d.%d.%d",
				arpLayer.SourceProtAddress[0],
				arpLayer.SourceProtAddress[1],
				arpLayer.SourceProtAddress[2],
				arpLayer.SourceProtAddress[3])
		} else {
			info.SrcAddress = fmt.Sprintf("%x", arpLayer.SourceProtAddress)
		}
		if len(arpLayer.DstProtAddress) == 4 {
			info.DstAddress = fmt.Sprintf("%d.%d.%d.%d",
				arpLayer.DstProtAddress[0],
				arpLayer.DstProtAddress[1],
				arpLayer.DstProtAddress[2],
				arpLayer.DstProtAddress[3])
		} else {
			info.DstAddress = fmt.Sprintf("%x", arpLayer.DstProtAddress)
		}
		info.Transport = TransportUnknown
		return info, nil
	} else {
		// Unknown network layer
		info.Protocol = ProtocolUnknown
		info.Transport = TransportUnknown
		return info, nil
	}

	// Extract transport layer protocol
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp := tcpLayer.(*layers.TCP)
		info.Transport = TransportTCP
		info.SrcPort = int(tcp.SrcPort)
		info.DstPort = int(tcp.DstPort)
	} else if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp := udpLayer.(*layers.UDP)
		info.Transport = TransportUDP
		info.SrcPort = int(udp.SrcPort)
		info.DstPort = int(udp.DstPort)
	} else if icmpLayer := packet.Layer(layers.LayerTypeICMPv4); icmpLayer != nil {
		info.Transport = TransportICMP
	} else if icmpv6Layer := packet.Layer(layers.LayerTypeICMPv6); icmpv6Layer != nil {
		info.Transport = TransportICMP
	} else if ipLayer != nil {
		// IP layer found but no transport layer
		info.Transport = TransportUnknown
	}

	return info, nil
}
