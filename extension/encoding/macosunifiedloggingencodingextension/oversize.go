// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package macosunifiedloggingencodingextension // import "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"

import (
	"encoding/binary"
	"fmt"
)

// ParseOversizeChunk parses an Oversize chunk (0x6002) containing large log entries
func ParseOversizeChunk(data []byte, entry *TraceV3Entry, header *TraceV3Header, timesyncData map[string]*TimesyncBoot) {
	if len(data) < 48 { // Need at least 48 bytes for oversize header
		entry.Message = fmt.Sprintf("Oversize chunk too small: %d bytes", len(data))
		return
	}

	// Parse oversize chunk header (based on rust implementation)
	firstProcID := binary.LittleEndian.Uint64(data[16:24])
	secondProcID := binary.LittleEndian.Uint32(data[24:28])
	ttl := data[28]
	continuousTime := binary.LittleEndian.Uint64(data[32:40])
	dataRefIndex := binary.LittleEndian.Uint32(data[40:44])
	publicDataSize := binary.LittleEndian.Uint16(data[44:46])
	privateDataSize := binary.LittleEndian.Uint16(data[46:48])

	entry.ThreadID = firstProcID
	entry.ProcessID = secondProcID

	// Calculate timestamp using oversize's continuous time with timesync conversion
	if timesyncData != nil && header != nil {
		// Use oversize's continuous time directly for timestamp calculation
		machTime := continuousTime
		// Convert using timesync conversion with preambleTime=0 for oversize entries
		entry.Timestamp = convertMachTimeToUnixNanosWithTimesync(machTime, header.BootUUID, 0, timesyncData)
	} else {
		// Fallback to raw continuous time if no timesync data available
		entry.Timestamp = continuousTime
	}

	entry.Message = fmt.Sprintf("Oversize entry: ttl=%d dataRef=%d publicSize=%d privateSize=%d (large log data)",
		ttl, dataRefIndex, publicDataSize, privateDataSize)
}
