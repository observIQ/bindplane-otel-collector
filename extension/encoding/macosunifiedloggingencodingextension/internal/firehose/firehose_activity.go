// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package firehose

import (
	"encoding/binary"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
)

// Activity represents a parsed firehose activity entry
type Activity struct {
	ActivityID         uint32
	Sentinel           uint32
	PID                uint64
	ActivityID2        uint32
	Sentinel2          uint32
	ActivityID3        uint32
	Sentinel3          uint32
	MessageStringRef   uint32
	PCID               uint32
	FirehoseFormatters Formatters
}

// Formatters represents a parsed firehose formatter flags
type Formatters struct {
	MainExe          bool
	SharedCache      bool
	HasLargeOffset   uint16
	LargeSharedCache uint16
	Absolute         bool
	UUIDRelative     string
	MainExeAltIndex  uint16 // If log entry uses an alternative uuid file index (ex: absolute). This value gets prepended to the unknown_pc_id/offset
}

// ParseFirehoseActivity parses a firehose activity entry
func ParseFirehoseActivity(data []byte, flags uint16, firehoseLogType uint8) (Activity, []byte) {
	var activity Activity
	entry := data

	// Useraction activity type does not have first Activity ID or sentinel
	const useraction uint8 = 0x3
	if firehoseLogType != useraction {
		firehoseData, activityID, _ := utils.Take(data, 4)
		firehoseData, sentinel, _ := utils.Take(firehoseData, 4)

		activity.ActivityID = binary.LittleEndian.Uint32(activityID)
		activity.Sentinel = binary.LittleEndian.Uint32(sentinel)
		entry = firehoseData
	}

	const uniquePid uint16 = 0x10 // has_unique_pid flag
	if (flags & uniquePid) != 0 {
		firehoseData, pid, _ := utils.Take(entry, 8)
		activity.PID = binary.LittleEndian.Uint64(pid)
		entry = firehoseData
	}

	const activityIDCurrent uint16 = 0x1 // has_current_aid flag
	if (flags & activityIDCurrent) != 0 {
		firehoseData, activityID, _ := utils.Take(entry, 4)
		firehoseData, sentinel, _ := utils.Take(firehoseData, 4)
		activity.ActivityID2 = binary.LittleEndian.Uint32(activityID)
		activity.Sentinel2 = binary.LittleEndian.Uint32(sentinel)
		entry = firehoseData
	}

	const activityIDOther uint16 = 0x200 // has_other_current_aid flag
	if (flags & activityIDOther) != 0 {
		firehoseData, activityID, _ := utils.Take(entry, 4)
		firehoseData, sentinel, _ := utils.Take(firehoseData, 4)
		activity.ActivityID3 = binary.LittleEndian.Uint32(activityID)
		activity.Sentinel3 = binary.LittleEndian.Uint32(sentinel)
		entry = firehoseData
	}

	firehoseData, pcID, _ := utils.Take(entry, 4)
	activity.PCID = binary.LittleEndian.Uint32(pcID)

	formatters, firehoseData := firehoseFormatterFlags(firehoseData, flags)
	activity.FirehoseFormatters = formatters
	entry = firehoseData

	return activity, entry
}

// firehoseFormatterFlags parses firehose formatter flags
func firehoseFormatterFlags(data []byte, flags uint16) (Formatters, []byte) {
	var formatterFlags Formatters

	var messageStringsUUID uint16 = 0x2 // main_exe flag
	var largeSharedCache uint16 = 0xc   // large_shared_cache flag
	var largeOffset uint16 = 0x20       // has_large_offset flag
	var flagCheck uint16 = 0xe
	input := data

	/*
		0x20 - has_large_offset flag. Offset to format string is larger than normal
		0xc - has_large_shared_cache flag. Offset to format string is larger than normal
		0x8 - absolute flag. The log uses an alterantive index number that points to the UUID file name in the Catalog which contains the format string
		0x2 - main_exe flag. A UUID file contains the format string
		0x4 - shared_cache flag. DSC file contains the format string
		0xa - uuid_relative flag. The UUID file name is in the log data (instead of the Catalog)
	*/
	switch flags & flagCheck {
	case 0x20:
		firehoseData, largeOffsetData, _ := utils.Take(input, 2)
		formatterFlags.HasLargeOffset = binary.LittleEndian.Uint16(largeOffsetData)
		input = firehoseData
		if (flags & largeSharedCache) != 0 {
			firehoseData, largeSharedCacheData, _ := utils.Take(firehoseData, 2)
			formatterFlags.LargeSharedCache = binary.LittleEndian.Uint16(largeSharedCacheData)
			input = firehoseData
		}
	case 0xc:
		if (flags & largeOffset) != 0 {
			firehoseData, largeOffsetData, _ := utils.Take(input, 2)
			formatterFlags.HasLargeOffset = binary.LittleEndian.Uint16(largeOffsetData)
			input = firehoseData
		}
		firehoseData, largeSharedCacheData, _ := utils.Take(input, 2)
		formatterFlags.LargeSharedCache = binary.LittleEndian.Uint16(largeSharedCacheData)
		input = firehoseData
	case 0x8:
		formatterFlags.Absolute = true
		if (flags & messageStringsUUID) == 0 {
			firehoseData, uuidFileIndex, _ := utils.Take(input, 2)
			formatterFlags.MainExeAltIndex = binary.LittleEndian.Uint16(uuidFileIndex)
			input = firehoseData
		}
	case 0x2:
		formatterFlags.MainExe = true
	case 0x4:
		formatterFlags.SharedCache = true
		if (flags & largeOffset) != 0 {
			firehoseData, largeOffsetData, _ := utils.Take(input, 2)
			formatterFlags.HasLargeOffset = binary.LittleEndian.Uint16(largeOffsetData)
			input = firehoseData
		}
	case 0xa:
		firehoseData, uuidRelative, _ := utils.Take(input, 16)
		formatterFlags.UUIDRelative = fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X",
			uuidRelative[0], uuidRelative[1], uuidRelative[2], uuidRelative[3],
			uuidRelative[4], uuidRelative[5], uuidRelative[6], uuidRelative[7],
			uuidRelative[8], uuidRelative[9], uuidRelative[10], uuidRelative[11],
			uuidRelative[12], uuidRelative[13], uuidRelative[14], uuidRelative[15])
		input = firehoseData
	default:
		// TODO: error
	}

	return formatterFlags, input

}
