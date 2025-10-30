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

// Package parser provides functions to parse network packets from tcpdump output.
package parser

import "time"

// PacketInfo contains parsed information from a network packet
type PacketInfo struct {
	Timestamp  time.Time // Timestamp from tcpdump output
	Protocol   string    // Network layer protocol: "IP", "IPv6", "ARP", etc.
	Transport  string    // Transport layer protocol: "TCP", "UDP", "ICMP", etc.
	SrcAddress string    // Source IP address
	DstAddress string    // Destination IP address
	SrcPort    int       // Source port (0 if not applicable)
	DstPort    int       // Destination port (0 if not applicable)
	HexData    string    // Full packet as hex string (without 0x prefix or spaces)
	Length     int       // Total packet length in bytes
}
