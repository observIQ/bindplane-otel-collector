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
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// Regular expressions for parsing tcpdump output
	timestampRegex = regexp.MustCompile(`^(\d{2}:\d{2}:\d{2}\.\d{6})`)
	hexLineRegex   = regexp.MustCompile(`^\s+0x[0-9a-f]+:\s+([0-9a-f\s]+)`)

	// Error definitions
	errEmptyInput    = errors.New("empty input")
	errNoHexData     = errors.New("no hex data found")
	errInvalidFormat = errors.New("invalid tcpdump format")
)

// ParseTcpdumpPacket parses a complete packet from tcpdump -xx output
// The input should be multiple lines: first line is the header, subsequent lines are hex data
func ParseTcpdumpPacket(lines []string) (*PacketInfo, error) {
	if len(lines) == 0 {
		return nil, errEmptyInput
	}

	// Filter out empty lines
	var nonEmptyLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	if len(nonEmptyLines) == 0 {
		return nil, errEmptyInput
	}

	// First line is the header
	headerLine := nonEmptyLines[0]
	packetInfo, err := parseHeaderLine(headerLine)
	if err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}

	// Remaining lines should be hex data
	if len(nonEmptyLines) < 2 {
		return nil, errNoHexData
	}

	hexLines := nonEmptyLines[1:]
	hexData, length, err := parseHexLines(hexLines)
	if err != nil {
		return nil, fmt.Errorf("parse hex data: %w", err)
	}

	packetInfo.HexData = hexData
	packetInfo.Length = length

	return packetInfo, nil
}

// parseHeaderLine parses the first line of tcpdump output
// Example: "12:34:56.789012 IP 192.168.1.100.54321 > 192.168.1.1.443: Flags [S], seq 1234567890"
func parseHeaderLine(line string) (*PacketInfo, error) {
	info := &PacketInfo{}

	// Extract timestamp
	timestampMatch := timestampRegex.FindStringSubmatch(line)
	if len(timestampMatch) < 2 {
		return nil, fmt.Errorf("%w: no timestamp found", errInvalidFormat)
	}

	ts, err := parseTimestamp(timestampMatch[1])
	if err != nil {
		return nil, fmt.Errorf("parse timestamp: %w", err)
	}
	info.Timestamp = ts

	// Remove timestamp from line for easier parsing
	line = strings.TrimSpace(line[len(timestampMatch[0]):])

	// Determine protocol (IP, IP6, ARP, etc.)
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return nil, fmt.Errorf("%w: not enough fields", errInvalidFormat)
	}

	info.Protocol = parts[0]

	// Parse addresses and ports
	// Format: "192.168.1.100.54321 > 192.168.1.1.443:"
	// or for ICMP: "192.168.1.100 > 192.168.1.1:"
	srcPart := parts[1]
	if len(parts) < 3 || parts[2] != ">" {
		return nil, fmt.Errorf("%w: missing direction indicator", errInvalidFormat)
	}
	dstPart := strings.TrimSuffix(parts[3], ":")

	// Parse source address and port
	info.SrcAddress, info.SrcPort = parseAddressPort(srcPart)
	info.DstAddress, info.DstPort = parseAddressPort(dstPart)

	// Determine transport protocol from the rest of the line
	restOfLine := strings.Join(parts[4:], " ")
	info.Transport = determineTransport(restOfLine, info.Protocol)

	return info, nil
}

// parseAddressPort splits an address:port combination
// For IPv4: "192.168.1.100.54321" -> "192.168.1.100", 54321
// For IPv6: "2001:db8::1.8080" -> "2001:db8::1", 8080
// For ICMP (no port): "192.168.1.100" -> "192.168.1.100", 0
func parseAddressPort(addrPort string) (string, int) {
	// Check if this is an IPv6 address (contains colons)
	if strings.Contains(addrPort, ":") {
		// IPv6 address - port is after the last dot
		lastDot := strings.LastIndex(addrPort, ".")
		if lastDot == -1 {
			// No port
			return addrPort, 0
		}

		portStr := addrPort[lastDot+1:]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			// Not a valid port
			return addrPort, 0
		}

		// Valid port for IPv6
		addr := addrPort[:lastDot]
		return addr, port
	}

	// IPv4 address handling
	// Find the last dot
	lastDot := strings.LastIndex(addrPort, ".")
	if lastDot == -1 {
		// No port
		return addrPort, 0
	}

	// Try to parse the part after the last dot as a port
	portStr := addrPort[lastDot+1:]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		// Not a valid port, treat the whole thing as address
		return addrPort, 0
	}

	// Check if this looks like a valid port (1-65535)
	// If it's > 255, it's definitely a port, not part of an IP
	// If it's <= 255, we need to check if the address part looks like a complete IPv4
	if port > 255 {
		// Definitely a port
		addr := addrPort[:lastDot]
		return addr, port
	}

	// port <= 255, could be last octet of IP or an actual port
	// Check if what's before the last dot looks like a complete IPv4 (has 3 dots total)
	addrPart := addrPort[:lastDot]
	dotCount := strings.Count(addrPart, ".")
	if dotCount == 3 {
		// This looks like IP.port (e.g., "192.168.1.1.53")
		return addrPart, port
	}

	// Otherwise, it's part of the IP address (e.g., "192.168.1.100")
	return addrPort, 0
}

// determineTransport determines the transport protocol from the packet description
func determineTransport(description, protocol string) string {
	descLower := strings.ToLower(description)

	// Check for explicit protocol mentions
	if strings.Contains(descLower, "tcp") || strings.Contains(descLower, "flags [") {
		return "TCP"
	}
	if strings.Contains(descLower, "udp") {
		return "UDP"
	}
	if strings.Contains(descLower, "icmp") {
		return "ICMP"
	}

	// Check for DNS patterns (usually UDP)
	// Example: "12345+ A? example.com."
	if strings.Contains(description, "+") && (strings.Contains(descLower, "a?") || strings.Contains(descLower, "aaaa?")) {
		return "UDP"
	}

	// Default based on protocol
	if protocol == "IP" || protocol == "IP6" {
		// If we have flags, it's likely TCP
		if strings.Contains(descLower, "flags") {
			return "TCP"
		}
		// Otherwise unknown
		return "Unknown"
	}

	return "Unknown"
}

// parseTimestamp parses a timestamp string like "12:34:56.789012"
func parseTimestamp(ts string) (time.Time, error) {
	// Parse time in HH:MM:SS.microseconds format
	// We'll use today's date as base
	now := time.Now()
	parsed, err := time.Parse("15:04:05.000000", ts)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp format: %w", err)
	}

	// Combine with today's date
	result := time.Date(
		now.Year(), now.Month(), now.Day(),
		parsed.Hour(), parsed.Minute(), parsed.Second(), parsed.Nanosecond(),
		now.Location(),
	)

	return result, nil
}

// parseHexLines parses the hex dump lines from tcpdump output
// Example:
//
//	0x0000:  4500 003c 1c46 4000 4006 b1e6 c0a8 0164
//	0x0010:  c0a8 0101 d431 01bb 4996 02d2 0000 0000
//
// Returns concatenated hex string (without spaces) and total byte count
func parseHexLines(lines []string) (string, int, error) {
	var hexBuilder strings.Builder
	byteCount := 0

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Extract hex data from line
		match := hexLineRegex.FindStringSubmatch(line)
		if len(match) < 2 {
			// Not a hex line, skip
			continue
		}

		hexData := match[1]
		// Remove all spaces
		hexData = strings.ReplaceAll(hexData, " ", "")
		hexBuilder.WriteString(hexData)

		// Count bytes (2 hex chars = 1 byte)
		byteCount += len(hexData) / 2
	}

	if hexBuilder.Len() == 0 {
		return "", 0, errNoHexData
	}

	return hexBuilder.String(), byteCount, nil
}

