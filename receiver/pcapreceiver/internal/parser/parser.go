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

// Package parser converts gopacket.Packet to OpenTelemetry logs.
package parser

import (
	"encoding/hex"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

// Parser converts gopacket.Packet to OpenTelemetry logs
type Parser struct {
	interfaceName string
}

// NewParser creates a new packet parser
func NewParser(interfaceName string) *Parser {
	return &Parser{
		interfaceName: interfaceName,
	}
}

// Parse converts a packet to plog.Logs
func (p *Parser) Parse(packet gopacket.Packet) plog.Logs {
	logs := plog.NewLogs()

	// Handle nil packet
	if packet == nil {
		return logs
	}

	// Create log record
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()

	// Set timestamp from packet metadata
	timestamp := packet.Metadata().Timestamp
	lr.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))

	// Set body as hex-encoded packet data
	lr.Body().SetStr(hex.EncodeToString(packet.Data()))

	// Extract attributes from packet layers
	attrs := lr.Attributes()

	// Basic packet attributes
	attrs.PutStr("interface", p.interfaceName)
	attrs.PutInt("packet.size", int64(packet.Metadata().Length))
	attrs.PutStr("packet.timestamp", timestamp.Format("2006-01-02T15:04:05.999999999Z07:00"))

	// Extract link layer (Ethernet)
	p.parseEthernetLayer(packet, attrs)

	// Extract network layer (IPv4/IPv6)
	p.parseNetworkLayer(packet, attrs)

	// Extract transport layer (TCP/UDP/ICMP)
	p.parseTransportLayer(packet, attrs)

	// Detect application protocol based on ports
	p.detectApplicationLayer(attrs)

	return logs
}

// parseEthernetLayer extracts Ethernet layer attributes
func (p *Parser) parseEthernetLayer(packet gopacket.Packet, attrs pcommon.Map) {
	if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
		eth, _ := ethLayer.(*layers.Ethernet)
		if eth != nil {
			attrs.PutStr("link.src_addr", eth.SrcMAC.String())
			attrs.PutStr("link.dst_addr", eth.DstMAC.String())
		}
	}
}

// parseNetworkLayer extracts network layer attributes (IPv4 or IPv6)
func (p *Parser) parseNetworkLayer(packet gopacket.Packet, attrs pcommon.Map) {
	// Try IPv4 first
	if ipv4Layer := packet.Layer(layers.LayerTypeIPv4); ipv4Layer != nil {
		ipv4, _ := ipv4Layer.(*layers.IPv4)
		if ipv4 != nil {
			attrs.PutStr("network.protocol", "IPv4")
			attrs.PutStr("network.src_addr", ipv4.SrcIP.String())
			attrs.PutStr("network.dst_addr", ipv4.DstIP.String())
			attrs.PutInt("network.ttl", int64(ipv4.TTL))
			return
		}
	}

	// Try IPv6
	if ipv6Layer := packet.Layer(layers.LayerTypeIPv6); ipv6Layer != nil {
		ipv6, _ := ipv6Layer.(*layers.IPv6)
		if ipv6 != nil {
			attrs.PutStr("network.protocol", "IPv6")
			attrs.PutStr("network.src_addr", ipv6.SrcIP.String())
			attrs.PutStr("network.dst_addr", ipv6.DstIP.String())
			attrs.PutInt("network.ttl", int64(ipv6.HopLimit))
			return
		}
	}
}

// parseTransportLayer extracts transport layer attributes (TCP, UDP, or ICMP)
func (p *Parser) parseTransportLayer(packet gopacket.Packet, attrs pcommon.Map) {
	// Try TCP
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		if tcp != nil {
			attrs.PutStr("transport.protocol", "TCP")
			attrs.PutInt("transport.src_port", int64(tcp.SrcPort))
			attrs.PutInt("transport.dst_port", int64(tcp.DstPort))

			// TCP flags
			if tcp.SYN {
				attrs.PutBool("transport.tcp.syn", true)
			}
			if tcp.ACK {
				attrs.PutBool("transport.tcp.ack", true)
			}
			if tcp.FIN {
				attrs.PutBool("transport.tcp.fin", true)
			}
			if tcp.RST {
				attrs.PutBool("transport.tcp.rst", true)
			}
			if tcp.PSH {
				attrs.PutBool("transport.tcp.psh", true)
			}
			return
		}
	}

	// Try UDP
	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		if udp != nil {
			attrs.PutStr("transport.protocol", "UDP")
			attrs.PutInt("transport.src_port", int64(udp.SrcPort))
			attrs.PutInt("transport.dst_port", int64(udp.DstPort))
			return
		}
	}

	// Try ICMP (IPv4)
	if icmpLayer := packet.Layer(layers.LayerTypeICMPv4); icmpLayer != nil {
		icmp, _ := icmpLayer.(*layers.ICMPv4)
		if icmp != nil {
			attrs.PutStr("transport.protocol", "ICMP")
			attrs.PutInt("transport.icmp.type", int64(icmp.TypeCode.Type()))
			attrs.PutInt("transport.icmp.code", int64(icmp.TypeCode.Code()))
			return
		}
	}

	// Try ICMPv6
	if icmpv6Layer := packet.Layer(layers.LayerTypeICMPv6); icmpv6Layer != nil {
		icmpv6, _ := icmpv6Layer.(*layers.ICMPv6)
		if icmpv6 != nil {
			attrs.PutStr("transport.protocol", "ICMPv6")
			attrs.PutInt("transport.icmp.type", int64(icmpv6.TypeCode.Type()))
			attrs.PutInt("transport.icmp.code", int64(icmpv6.TypeCode.Code()))
			return
		}
	}
}

// detectApplicationLayer detects application protocol based on port numbers
func (p *Parser) detectApplicationLayer(attrs pcommon.Map) {
	// Get transport protocol and ports
	transportProto, ok := attrs.Get("transport.protocol")
	if !ok {
		return
	}

	srcPortVal, hasSrcPort := attrs.Get("transport.src_port")
	dstPortVal, hasDstPort := attrs.Get("transport.dst_port")

	if !hasSrcPort || !hasDstPort {
		return // No ports for ICMP
	}

	// Safely convert int64 to uint16 with bounds checking
	srcPortInt := srcPortVal.Int()
	dstPortInt := dstPortVal.Int()

	if srcPortInt < 0 || srcPortInt > 65535 {
		return // Invalid port
	}
	if dstPortInt < 0 || dstPortInt > 65535 {
		return // Invalid port
	}

	srcPort := uint16(srcPortInt)
	dstPort := uint16(dstPortInt)

	// Check both source and destination ports
	protocol := detectApplicationProtocol(dstPort, transportProto.Str())
	if protocol == "Unknown" {
		protocol = detectApplicationProtocol(srcPort, transportProto.Str())
	}

	attrs.PutStr("application.protocol", protocol)
}

// detectApplicationProtocol maps port numbers to application protocols
func detectApplicationProtocol(port uint16, transportProtocol string) string {
	// Common TCP ports
	if transportProtocol == "TCP" {
		switch port {
		case 20, 21:
			return "FTP"
		case 22:
			return "SSH"
		case 23:
			return "Telnet"
		case 25:
			return "SMTP"
		case 80:
			return "HTTP"
		case 110:
			return "POP3"
		case 143:
			return "IMAP"
		case 443:
			return "HTTPS"
		case 3306:
			return "MySQL"
		case 5432:
			return "PostgreSQL"
		case 6379:
			return "Redis"
		case 8080, 8081, 8443:
			return "HTTP"
		case 27017:
			return "MongoDB"
		}
	}

	// Common UDP ports
	if transportProtocol == "UDP" {
		switch port {
		case 53:
			return "DNS"
		case 67, 68:
			return "DHCP"
		case 123:
			return "NTP"
		case 161, 162:
			return "SNMP"
		case 514:
			return "Syslog"
		}
	}

	return "Unknown"
}
