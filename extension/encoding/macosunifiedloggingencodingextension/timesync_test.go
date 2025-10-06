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

package macosunifiedloggingencodingextension

import (
	"os"
	"path/filepath"
	"testing"

	testutil "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestParseTimesyncData_DataFromArchive(t *testing.T) {
	// testutil.SkipIfNoReceiverTestdata(t)
	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "system_logs_big_sur.logarchive", "timesync", "0000000000000002.timesync")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := ParseTimesyncData(data)
	require.NoError(t, err)

	require.Equal(t, 5, len(results))
	require.Equal(t, len(results["9A6A3124274A44B29ABF2BC9E4599B3B"].TimesyncRecords), 5)
}

func TestParseTimesyncData_BadBootHeader(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)
	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "Bad Data", "Timesync", "Bad_Boot_header_0000000000000002.timesync")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := ParseTimesyncData(data)
	require.Contains(t, err.Error(), "invalid boot signature")
	require.Empty(t, results)
}

func TestParseTimesyncData_BadRecordHeader(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)
	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "Bad Data", "Timesync", "Bad_Record_header_0000000000000002.timesync")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := ParseTimesyncData(data)
	require.Contains(t, err.Error(), "invalid boot signature")
	require.Equal(t, 1, len(results))
}

func TestParseTimesyncData_BadContent(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)
	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "Bad Data", "Timesync", "Bad_content_0000000000000002.timesync")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := ParseTimesyncData(data)
	require.Contains(t, err.Error(), "invalid boot signature")
	require.Equal(t, 2, len(results))
}

func TestParseTimesyncData_BadFile(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)
	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "Bad Data", "Timesync", "BadFile.timesync")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := ParseTimesyncData(data)
	require.Contains(t, err.Error(), "invalid boot signature")
	require.Equal(t, 0, len(results))
}

func TestParseTimesyncRecord(t *testing.T) {
	testData := []byte{
		84, 115, 32, 0, 0, 0, 0, 0, 165, 196, 104, 252, 1, 0, 0, 0, 216, 189, 100, 108, 116,
		158, 131, 22, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	record, remainingData, err := ParseTimesyncRecord(testData)
	require.NoError(t, err)
	require.Equal(t, uint32(0x207354), record.Signature)
	require.Equal(t, uint32(0), record.UnknownFlags)
	require.Equal(t, uint64(8529691813), record.KernelTime)
	require.Equal(t, int64(1622314513655447000), record.WallTime)
	require.Equal(t, uint32(0), record.Timezone)
	require.Equal(t, uint32(0), record.DaylightSavings)
	require.Equal(t, len(remainingData), 0)
}

func TestParseTimesyncBoot(t *testing.T) {
	testData := []byte{
		176, 187, 48, 0, 0, 0, 0, 0, 132, 91, 13, 213, 1, 96, 69, 62, 172, 224, 56, 118, 12,
		123, 92, 29, 1, 0, 0, 0, 1, 0, 0, 0, 168, 167, 19, 176, 114, 158, 131, 22, 0, 0, 0, 0,
		0, 0, 0, 0,
	}
	boot, _, err := ParseTimesyncBoot(testData)
	require.NoError(t, err)
	require.Equal(t, uint16(0xbbb0), boot.Signature)
	require.Equal(t, uint16(48), boot.HeaderSize)
	require.Equal(t, uint32(0), boot.Unknown)
	require.Equal(t, "845B0DD50160453EACE038760C7B5C1D", boot.BootUUID)
	require.Equal(t, uint32(1), boot.TimebaseNumerator)
	require.Equal(t, uint32(1), boot.TimebaseDenominator)
	require.Equal(t, int64(1622314506201049000), boot.BootTime)
	require.Equal(t, uint32(0), boot.TimezoneOffsetMins)
	require.Equal(t, uint32(0), boot.DaylightSavings)
}

// TestParseTimesyncData tests parsing of a complete timesync data structure
// containing both boot records and timesync records
func TestParseTimesyncData(t *testing.T) {
	// Create test data with one boot record followed by two timesync records
	// This mimics the structure found in actual macOS timesync files

	// Boot record data (Intel timebase 1/1)
	bootRecord := []byte{
		// Boot signature (0xbbb0) + header size (48) + unknown (0)
		176, 187, 48, 0, 0, 0, 0, 0,
		// Boot UUID: 845B0DD50160453EACE038760C7B5C1D (big-endian)
		132, 91, 13, 213, 1, 96, 69, 62, 172, 224, 56, 118, 12, 123, 92, 29,
		// Timebase numerator (1) + denominator (1)
		1, 0, 0, 0, 1, 0, 0, 0,
		// Boot time: 1622314506201049000 ns since Unix epoch
		168, 167, 19, 176, 114, 158, 131, 22,
		// Timezone offset (0) + daylight savings (0)
		0, 0, 0, 0, 0, 0, 0, 0,
	}

	// First timesync record
	timesyncRecord1 := []byte{
		// Timesync signature (0x207354) + unknown flags (0)
		84, 115, 32, 0, 0, 0, 0, 0,
		// Kernel time: 8529691813
		165, 196, 104, 252, 1, 0, 0, 0,
		// Wall time: 1622314513655447000 ns since Unix epoch
		216, 189, 100, 108, 116, 158, 131, 22,
		// Timezone (0) + daylight savings (0)
		0, 0, 0, 0, 0, 0, 0, 0,
	}

	// Second timesync record with different values
	timesyncRecord2 := []byte{
		// Timesync signature (0x207354) + unknown flags (0)
		84, 115, 32, 0, 0, 0, 0, 0,
		// Kernel time: 8529691900 (slightly later)
		// 8529691900 = 0x1FC68C4FC in little-endian: [252, 196, 104, 252, 1, 0, 0, 0]
		252, 196, 104, 252, 1, 0, 0, 0,
		// Wall time: 1622314513655534000 ns since Unix epoch (87μs later)
		176, 17, 102, 108, 116, 158, 131, 22,
		// Timezone (0) + daylight savings (0)
		0, 0, 0, 0, 0, 0, 0, 0,
	}

	// Combine all data
	testData := append(bootRecord, timesyncRecord1...)
	testData = append(testData, timesyncRecord2...)

	timesyncData, err := ParseTimesyncData(testData)
	require.NoError(t, err)
	require.Len(t, timesyncData, 1)

	boot := timesyncData["845B0DD50160453EACE038760C7B5C1D"]
	require.NotNil(t, boot)
	require.Equal(t, "845B0DD50160453EACE038760C7B5C1D", boot.BootUUID)
	require.Equal(t, uint32(1), boot.TimebaseNumerator)
	require.Equal(t, uint32(1), boot.TimebaseDenominator)
	require.Equal(t, int64(1622314506201049000), boot.BootTime)
	require.Len(t, boot.TimesyncRecords, 2)

	// Verify first timesync record
	require.Equal(t, uint64(8529691813), boot.TimesyncRecords[0].KernelTime)
	require.Equal(t, int64(1622314513655447000), boot.TimesyncRecords[0].WallTime)

	// Verify second timesync record
	require.Equal(t, uint64(8529691900), boot.TimesyncRecords[1].KernelTime)
	require.Equal(t, int64(1622314513655534000), boot.TimesyncRecords[1].WallTime)
}

// TestParseTimesyncDataARMTimebase tests parsing with Apple Silicon (ARM) timebase
func TestParseTimesyncDataARMTimebase(t *testing.T) {
	// ARM boot record with 125/3 timebase
	bootRecord := []byte{
		// Boot signature (0xbbb0) + header size (48) + unknown (0)
		176, 187, 48, 0, 0, 0, 0, 0,
		// Boot UUID: 3E12B435814B4C62918CEBC0826F06B8 (big-endian)
		62, 18, 180, 53, 129, 75, 76, 98, 145, 140, 235, 192, 130, 111, 6, 184,
		// Timebase numerator (125) + denominator (3) - Apple Silicon
		125, 0, 0, 0, 3, 0, 0, 0,
		// Boot time: 1650767513342574600 ns since Unix epoch
		8, 0, 138, 167, 78, 180, 232, 22,
		// Timezone offset (0) + daylight savings (0)
		0, 0, 0, 0, 0, 0, 0, 0,
	}

	// Timesync record for ARM
	timesyncRecord := []byte{
		// Timesync signature (0x207354) + unknown flags (0)
		84, 115, 32, 0, 0, 0, 0, 0,
		// Kernel time: 2818326118 (raw mach time before timebase adjustment)
		102, 62, 252, 167, 0, 0, 0, 0,
		// Wall time: 1650767519086487000 ns since Unix epoch
		216, 37, 231, 253, 79, 180, 232, 22,
		// Timezone (0) + daylight savings (0)
		0, 0, 0, 0, 0, 0, 0, 0,
	}

	testData := append(bootRecord, timesyncRecord...)

	timesyncData, err := ParseTimesyncData(testData)
	require.NoError(t, err)
	require.Len(t, timesyncData, 1)

	boot := timesyncData["3E12B435814B4C62918CEBC0826F06B8"]
	require.NotNil(t, boot)
	require.Equal(t, uint32(125), boot.TimebaseNumerator)
	require.Equal(t, uint32(3), boot.TimebaseDenominator)
	require.Equal(t, int64(1650767513342574600), boot.BootTime)
	require.Len(t, boot.TimesyncRecords, 1)
	require.Equal(t, uint64(2818326118), boot.TimesyncRecords[0].KernelTime)
	require.Equal(t, int64(1650767519086487000), boot.TimesyncRecords[0].WallTime)
}

// TestGetTimestamp tests timestamp calculation for Intel (1/1 timebase)
func TestGetTimestamp(t *testing.T) {
	// Create test timesync data with Intel timebase
	timesyncData := map[string]*TimesyncBoot{
		"845B0DD50160453EACE038760C7B5C1D": {
			BootUUID:            "845B0DD50160453EACE038760C7B5C1D",
			TimebaseNumerator:   1,
			TimebaseDenominator: 1,
			BootTime:            1622314506201049000,
			TimesyncRecords: []TimesyncRecord{
				{
					KernelTime: 8529691813,
					WallTime:   1622314513655447000,
				},
				{
					KernelTime: 8529791813,          // 100ms later in mach time
					WallTime:   1622314513755447000, // 100ms later in wall time
				},
			},
		},
	}

	// Test case 1: Normal timestamp calculation
	// firehoseLogDeltaTime falls between the two timesync records
	bootUUID := "845B0DD50160453EACE038760C7B5C1D"
	firehoseLogDeltaTime := uint64(8529741813) // 50ms after first record
	firehosePreambleTime := uint64(1)

	timestamp := GetTimestamp(timesyncData, bootUUID, firehoseLogDeltaTime, firehosePreambleTime)

	// Should use first timesync record as reference
	// Expected: (8529741813 - 8529691813) + 1622314513655447000 = 1622314513655497000
	expected := float64(1622314513655497000)
	require.Equal(t, expected, timestamp)

	// Test case 2: firehoseLogDeltaTime before any timesync records
	firehoseLogDeltaTime = uint64(8529691800) // Before first record
	timestamp = GetTimestamp(timesyncData, bootUUID, firehoseLogDeltaTime, firehosePreambleTime)

	// Should use first timesync record as fallback
	// Expected: (8529691800 - 8529691813) + 1622314513655447000 = 1622314513655446987
	expected = float64(1622314513655446987)
	require.Equal(t, expected, timestamp)
}

// TestGetTimestampZeroPreamble tests timestamp calculation when preamble time is zero
func TestGetTimestampZeroPreamble(t *testing.T) {
	timesyncData := map[string]*TimesyncBoot{
		"845B0DD50160453EACE038760C7B5C1D": {
			BootUUID:            "845B0DD50160453EACE038760C7B5C1D",
			TimebaseNumerator:   1,
			TimebaseDenominator: 1,
			BootTime:            1622314506201049000,
			TimesyncRecords: []TimesyncRecord{
				{
					KernelTime: 8529691813,
					WallTime:   1622314513655447000,
				},
			},
		},
	}

	bootUUID := "845B0DD50160453EACE038760C7B5C1D"
	firehoseLogDeltaTime := uint64(8529741813)
	firehosePreambleTime := uint64(0) // Zero preamble time

	timestamp := GetTimestamp(timesyncData, bootUUID, firehoseLogDeltaTime, firehosePreambleTime)

	// When preamble time is 0, we set initial values to boot time, but still process timesync records
	// The loop finds timesync record with KernelTime=8529691813, so that becomes the reference
	// Expected: (8529741813 - 8529691813) + 1622314513655447000 = 1622314513655497000
	expected := float64(1622314513655497000)
	require.Equal(t, expected, timestamp)
}

// TestGetTimestampARMTimebase tests timestamp calculation for Apple Silicon (125/3 timebase)
func TestGetTimestampARMTimebase(t *testing.T) {
	// Create test timesync data with ARM timebase
	timesyncData := map[string]*TimesyncBoot{
		"3E12B435814B4C62918CEBC0826F06B8": {
			BootUUID:            "3E12B435814B4C62918CEBC0826F06B8",
			TimebaseNumerator:   125,
			TimebaseDenominator: 3,
			BootTime:            1650767513342574600,
			TimesyncRecords: []TimesyncRecord{
				{
					KernelTime: 2818326118, // Raw mach time before timebase adjustment
					WallTime:   1650767519086487000,
				},
			},
		},
	}

	bootUUID := "3E12B435814B4C62918CEBC0826F06B8"
	firehoseLogDeltaTime := uint64(2818326118)
	firehosePreambleTime := uint64(1)

	timestamp := GetTimestamp(timesyncData, bootUUID, firehoseLogDeltaTime, firehosePreambleTime)

	// For ARM, timebase adjustment is 125.0/3.0 ≈ 41.666...
	// Expected: (2818326118 * 41.666... - 2818326118 * 41.666...) + 1650767519086487000
	// = 0 + 1650767519086487000 = 1650767519086487000
	expected := float64(1650767519086487000)
	require.Equal(t, expected, timestamp)
}

// TestGetTimestampMissingBootUUID tests behavior when boot UUID is not found
func TestGetTimestampMissingBootUUID(t *testing.T) {
	timesyncData := map[string]*TimesyncBoot{}

	bootUUID := "NONEXISTENT-BOOT-UUID"
	firehoseLogDeltaTime := uint64(8529741813)
	firehosePreambleTime := uint64(1)

	timestamp := GetTimestamp(timesyncData, bootUUID, firehoseLogDeltaTime, firehosePreambleTime)

	// When boot UUID is not found, should return the raw delta time
	// Expected: (8529741813 * 1.0 - 0 * 1.0) + 0 = 8529741813
	expected := float64(8529741813)
	require.Equal(t, expected, timestamp)
}
