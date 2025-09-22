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

package macosunifiedloggingencodingextension // import "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
)

// TimesyncBoot represents a timesync boot record containing time correlation data
// Based on the Rust implementation from mandiant/macos-UnifiedLogs
type TimesyncBoot struct {
	Signature           uint16           // Boot record signature (0xbbb0)
	HeaderSize          uint16           // Size of the header
	Unknown             uint32           // Unknown field
	BootUUID            string           // Boot UUID (formatted)
	TimebaseNumerator   uint32           // Timebase numerator for mach time conversion
	TimebaseDenominator uint32           // Timebase denominator for mach time conversion
	BootTime            int64            // Boot time in nanoseconds since Unix epoch
	TimezoneOffsetMins  uint32           // Timezone offset in minutes
	DaylightSavings     uint32           // Daylight savings flag (0=no DST, 1=DST)
	TimesyncRecords     []TimesyncRecord // Associated timesync records
}

// TimesyncRecord represents a single timesync record correlating mach time to wall time
// Based on the Rust implementation from mandiant/macos-UnifiedLogs
type TimesyncRecord struct {
	Signature       uint32 // Record signature (0x207354)
	UnknownFlags    uint32 // Unknown flags
	KernelTime      uint64 // Mach continuous timestamp (kernel time)
	WallTime        int64  // Wall time in nanoseconds since Unix epoch
	Timezone        uint32 // Timezone value
	DaylightSavings uint32 // Daylight savings flag
}

const expectedTimesyncSignature = 0x207354

// ParseTimesyncData parses timesync files and returns a map of boot UUIDs to TimesyncBoot records
// Based on the Rust implementation logic
func ParseTimesyncData(data []byte) (map[string]*TimesyncBoot, error) {
	timesyncData := make(map[string]*TimesyncBoot)
	offset := 0
	var currentBoot *TimesyncBoot

	for offset < len(data) {
		if offset+4 > len(data) {
			break
		}

		// Check the signature to determine record type
		signature := binary.LittleEndian.Uint32(data[offset:])

		if signature == 0x207354 { // Timesync record signature
			record, err := ParseTimesyncRecord(data)
			if err != nil {
				// Skip this record and continue
				offset += 4
				continue
			}

			if currentBoot != nil {
				currentBoot.TimesyncRecords = append(currentBoot.TimesyncRecords, *record)
			}
			offset = offset + 4

		} else {
			// Try to parse as boot record
			boot, err := ParseTimesyncBoot(data)
			if err != nil {
				// Skip this data and continue
				offset += 4
				continue
			}

			// Save previous boot record if it exists
			if currentBoot != nil {
				if existing, exists := timesyncData[currentBoot.BootUUID]; exists {
					// Merge records if boot UUID already exists
					existing.TimesyncRecords = append(existing.TimesyncRecords, currentBoot.TimesyncRecords...)
				} else {
					timesyncData[currentBoot.BootUUID] = currentBoot
				}
			}

			currentBoot = boot
		}
	}

	// Save the last boot record
	if currentBoot != nil {
		if existing, exists := timesyncData[currentBoot.BootUUID]; exists {
			existing.TimesyncRecords = append(existing.TimesyncRecords, currentBoot.TimesyncRecords...)
		} else {
			timesyncData[currentBoot.BootUUID] = currentBoot
		}
	}

	return timesyncData, nil
}

// ParseTimesyncBoot parses a timesync boot record
func ParseTimesyncBoot(data []byte) (*TimesyncBoot, error) {
	boot := &TimesyncBoot{}
	expectedSignature := uint16(0xbbb0)
	data, signature, _ := utils.Take(data, 2)
	if binary.LittleEndian.Uint16(signature) != expectedSignature {
		return nil, fmt.Errorf("invalid boot signature: expected 0x%x, got 0x%x", expectedSignature, signature)
	}
	boot.Signature = binary.LittleEndian.Uint16(signature)

	data, headerSize, _ := utils.Take(data, 2)
	boot.HeaderSize = binary.LittleEndian.Uint16(headerSize)

	data, unknown, _ := utils.Take(data, 4)
	boot.Unknown = binary.LittleEndian.Uint32(unknown)

	data, bootUUID, _ := utils.Take(data, 16)
	boot.BootUUID = fmt.Sprintf("%08X%04X%04X%04X%012X",
		binary.BigEndian.Uint32(bootUUID[0:4]),
		binary.BigEndian.Uint16(bootUUID[4:6]),
		binary.BigEndian.Uint16(bootUUID[6:8]),
		binary.BigEndian.Uint16(bootUUID[8:10]),
		bootUUID[10:16])

	data, timebaseNumerator, _ := utils.Take(data, 4)
	boot.TimebaseNumerator = binary.LittleEndian.Uint32(timebaseNumerator)

	data, timebaseDenominator, _ := utils.Take(data, 4)
	boot.TimebaseDenominator = binary.LittleEndian.Uint32(timebaseDenominator)

	data, bootTime, _ := utils.Take(data, 8)
	boot.BootTime = int64(binary.LittleEndian.Uint64(bootTime))

	data, timezoneOffsetMins, _ := utils.Take(data, 4)
	boot.TimezoneOffsetMins = binary.LittleEndian.Uint32(timezoneOffsetMins)

	data, daylightSavings, _ := utils.Take(data, 4)
	boot.DaylightSavings = binary.LittleEndian.Uint32(daylightSavings)

	boot.TimesyncRecords = make([]TimesyncRecord, 0)

	return boot, nil
}

// ParseTimesyncRecord parses a timesync record
func ParseTimesyncRecord(data []byte) (*TimesyncRecord, error) {
	record := &TimesyncRecord{}
	data, signature, _ := utils.Take(data, 4)

	if binary.LittleEndian.Uint32(signature) != expectedTimesyncSignature {
		return record, fmt.Errorf("invalid timesync signature: expected 0x%x, got 0x%x", expectedTimesyncSignature, signature)
	}

	data, unknownFlags, _ := utils.Take(data, 4)
	data, kernelTime, _ := utils.Take(data, 8)
	data, wallTime, _ := utils.Take(data, 8)
	data, timezone, _ := utils.Take(data, 4)
	data, daylightSavings, _ := utils.Take(data, 4)

	record.Signature = binary.LittleEndian.Uint32(signature)
	record.UnknownFlags = binary.LittleEndian.Uint32(unknownFlags)
	record.KernelTime = binary.LittleEndian.Uint64(kernelTime)
	record.WallTime = int64(binary.LittleEndian.Uint64(wallTime))
	record.Timezone = binary.LittleEndian.Uint32(timezone)
	record.DaylightSavings = binary.LittleEndian.Uint32(daylightSavings)

	return record, nil
}

// GetTimestamp calculates a Unix epoch timestamp from mach time using timesync data
// This implements the complex timestamp calculation logic from the Rust parser
func GetTimestamp(
	timesyncData map[string]*TimesyncBoot,
	bootUUID string,
	firehoseLogDeltaTime uint64,
	firehosePreambleTime uint64,
) float64 {
	/*  Timestamp calculation logic (from Rust implementation):
		Firehose Log entry timestamp is calculated by using firehose_preamble_time, firehose.continous_time_delta, and timesync timestamps
		Firehose log header/preample contains a base timestamp
		  Ex: Firehose header base time is 2022-01-01 00:00:00
		All log entries following the header are continous from that base. EXCEPT when the base time is zero. If the base time is zero the TimeSync boot record boot time is used (boot time)
		  Ex: Firehose log entry time is +60 seconds
		Timestamp would be 2022-01-01 00:01:00

		(firehose_log_entry_continous_time = firehose.continous_time_delta | ((firehose.continous_time_delta_upper) << 32))
		firehose_log_delta_time = firehose_preamble_time + firehose_log_entry_continous_time

		Get all timesync boot records if timesync uuid equals boot uuid in tracev3 header data

		Loop through all timesync records from matching boot uuid until timesync cont_time/kernel time is greater than firehose_preamble_time
		If firehose_header_time equals zero. Then the Timesync header walltime is used (the Timesync header cont_time/kernel time is then always zero)
		Subtract timesync_cont_time/kernel time from firehose_log_delta_time
		IF APPLE SILICON (ARM) is the architecture, then we need to mupltiple timesync_cont_time and firehose_log_delta_time by the timebase 125.0/3.0 to get the nanocsecond representation

	   Add results to timesync_walltime (unix epoch in nanoseconds)
	   Final results is unix epoch timestamp in nano seconds
	*/

	var timesyncContinuousTime uint64
	var timesyncWallTime int64

	// Apple Intel uses 1/1 as the timebase
	timebaseAdjustment := 1.0

	// Try different UUID format variations to find a match
	var foundTimesync *TimesyncBoot

	// Try the lookup with different case variations and format variations
	testUUIDs := []string{
		bootUUID,                              // Original format
		strings.ToUpper(bootUUID),             // Uppercase
		strings.ToLower(bootUUID),             // Lowercase
		strings.ReplaceAll(bootUUID, "-", ""), // Remove dashes
		strings.ToUpper(strings.ReplaceAll(bootUUID, "-", "")), // Uppercase without dashes
		strings.ToLower(strings.ReplaceAll(bootUUID, "-", "")), // Lowercase without dashes
	}

	for _, testUUID := range testUUIDs {
		if ts, ok := timesyncData[testUUID]; ok {
			foundTimesync = ts
			break
		}
	}

	// If still no match, try case-insensitive search through all keys
	if foundTimesync == nil {
		for uuid, ts := range timesyncData {
			if strings.EqualFold(uuid, bootUUID) {
				foundTimesync = ts
				break
			}
		}
	}

	if foundTimesync == nil {
		// If no match found, return the raw mach time (which produces 1969/1970 timestamps)
		// This is better than a hardcoded fallback as it preserves relative timing
		return float64(firehoseLogDeltaTime)
	}

	timesync := foundTimesync

	if timesync.TimebaseNumerator == 125 && timesync.TimebaseDenominator == 3 {
		// For Apple Silicon (ARM) we need to adjust the mach time by multiplying by 125.0/3.0 to get the accurate nanosecond count
		timebaseAdjustment = 125.0 / 3.0
	}

	// A preamble time of 0 means we need to find the appropriate timesync correlation point
	// for the firehoseLogDeltaTime (which is a mach absolute time)
	if firehosePreambleTime == 0 {
		// Find the timesync record that correlates with this mach time
		// We'll search through timesync records to find the right correlation point
		timesyncContinuousTime = 0
		timesyncWallTime = 0
	}

	for _, timesyncRecord := range timesync.TimesyncRecords {
		if timesyncRecord.KernelTime > firehoseLogDeltaTime {
			if timesyncContinuousTime == 0 && timesyncWallTime == 0 {
				timesyncContinuousTime = timesyncRecord.KernelTime
				timesyncWallTime = timesyncRecord.WallTime
			}
			break
		}

		timesyncContinuousTime = timesyncRecord.KernelTime
		timesyncWallTime = timesyncRecord.WallTime
	}

	// Calculate the continuous time difference and apply timebase adjustment
	firehoseAdjusted := float64(firehoseLogDeltaTime) * timebaseAdjustment
	timesyncAdjusted := float64(timesyncContinuousTime) * timebaseAdjustment
	continuousTime := firehoseAdjusted - timesyncAdjusted
	result := continuousTime + float64(timesyncWallTime)

	return result
}

// NormalizeBootUUID normalizes boot UUID format to match timesync data format
func NormalizeBootUUID(uuid string) string {
	// Convert to uppercase to match timesync format (preserve hyphens)
	normalized := strings.ToUpper(uuid)
	return normalized
}
