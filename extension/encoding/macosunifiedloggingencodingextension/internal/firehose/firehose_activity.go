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

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
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
func ParseFirehoseActivity(data []byte, flags uint16, firehoseLogType uint8) (Activity, []byte, error) {
	var activity Activity
	var activityID, sentinel, pid, pcID []byte
	var err error

	// Useraction activity type does not have first Activity ID or sentinel
	const useraction uint8 = 0x3
	if firehoseLogType != useraction {
		data, activityID, err = helpers.Take(data, 4)
		if err != nil {
			return activity, data, fmt.Errorf("failed to read activity ID: %w", err)
		}
		data, sentinel, err = helpers.Take(data, 4)
		if err != nil {
			return activity, data, fmt.Errorf("failed to read sentinel: %w", err)
		}

		activity.ActivityID = binary.LittleEndian.Uint32(activityID)
		activity.Sentinel = binary.LittleEndian.Uint32(sentinel)
	}

	const uniquePid uint16 = 0x10 // has_unique_pid flag
	if (flags & uniquePid) != 0 {
		data, pid, err = helpers.Take(data, 8)
		if err != nil {
			return activity, data, fmt.Errorf("failed to read pid: %w", err)
		}
		activity.PID = binary.LittleEndian.Uint64(pid)
	}

	const activityIDCurrent uint16 = 0x1 // has_current_aid flag
	if (flags & activityIDCurrent) != 0 {
		data, activityID, err = helpers.Take(data, 4)
		if err != nil {
			return activity, data, fmt.Errorf("failed to read activity ID: %w", err)
		}
		data, sentinel, err = helpers.Take(data, 4)
		if err != nil {
			return activity, data, fmt.Errorf("failed to read sentinel: %w", err)
		}
		activity.ActivityID2 = binary.LittleEndian.Uint32(activityID)
		activity.Sentinel2 = binary.LittleEndian.Uint32(sentinel)
	}

	const activityIDOther uint16 = 0x200 // has_other_current_aid flag
	if (flags & activityIDOther) != 0 {
		data, activityID, err = helpers.Take(data, 4)
		if err != nil {
			return activity, data, fmt.Errorf("failed to read activity ID: %w", err)
		}
		data, sentinel, err = helpers.Take(data, 4)
		if err != nil {
			return activity, data, fmt.Errorf("failed to read sentinel: %w", err)
		}
		activity.ActivityID3 = binary.LittleEndian.Uint32(activityID)
		activity.Sentinel3 = binary.LittleEndian.Uint32(sentinel)
	}

	data, pcID, err = helpers.Take(data, 4)
	if err != nil {
		return activity, data, fmt.Errorf("failed to read pc ID: %w", err)
	}
	activity.PCID = binary.LittleEndian.Uint32(pcID)

	var formatters Formatters
	formatters, data, err = firehoseFormatterFlags(data, flags)
	if err != nil {
		return activity, data, fmt.Errorf("failed to parse firehose formatter flags: %w", err)
	}
	activity.FirehoseFormatters = formatters

	return activity, data, nil
}

// firehoseFormatterFlags parses firehose formatter flags
func firehoseFormatterFlags(data []byte, flags uint16) (Formatters, []byte, error) {
	var formatterFlags Formatters

	var largeOffsetData, largeSharedCacheData, uuidFileIndex, uuidRelative []byte
	var err error

	var messageStringsUUID uint16 = 0x2 // main_exe flag
	var largeSharedCache uint16 = 0xc   // large_shared_cache flag
	var largeOffset uint16 = 0x20       // has_large_offset flag
	var flagCheck uint16 = 0xe

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
		data, largeOffsetData, err = helpers.Take(data, 2)
		if err != nil {
			return formatterFlags, data, fmt.Errorf("failed to read large offset data: %w", err)
		}
		formatterFlags.HasLargeOffset = binary.LittleEndian.Uint16(largeOffsetData)
		if (flags & largeSharedCache) != 0 {
			data, largeSharedCacheData, err = helpers.Take(data, 2)
			if err != nil {
				return formatterFlags, data, fmt.Errorf("failed to read large shared cache data: %w", err)
			}
			formatterFlags.LargeSharedCache = binary.LittleEndian.Uint16(largeSharedCacheData)
		}
	case 0xc:
		if (flags & largeOffset) != 0 {
			data, largeOffsetData, err = helpers.Take(data, 2)
			if err != nil {
				return formatterFlags, data, fmt.Errorf("failed to read large offset data: %w", err)
			}
			formatterFlags.HasLargeOffset = binary.LittleEndian.Uint16(largeOffsetData)
		}
		data, largeSharedCacheData, err = helpers.Take(data, 2)
		if err != nil {
			return formatterFlags, data, fmt.Errorf("failed to read large shared cache data: %w", err)
		}
		formatterFlags.LargeSharedCache = binary.LittleEndian.Uint16(largeSharedCacheData)
	case 0x8:
		formatterFlags.Absolute = true
		if (flags & messageStringsUUID) == 0 {
			data, uuidFileIndex, err = helpers.Take(data, 2)
			if err != nil {
				return formatterFlags, data, fmt.Errorf("failed to read uuid file index: %w", err)
			}
			formatterFlags.MainExeAltIndex = binary.LittleEndian.Uint16(uuidFileIndex)
		}
	case 0x2:
		formatterFlags.MainExe = true
	case 0x4:
		formatterFlags.SharedCache = true
		if (flags & largeOffset) != 0 {
			data, largeOffsetData, err = helpers.Take(data, 2)
			if err != nil {
				return formatterFlags, data, fmt.Errorf("failed to read large offset data: %w", err)
			}
			formatterFlags.HasLargeOffset = binary.LittleEndian.Uint16(largeOffsetData)
		}
	case 0xa:
		data, uuidRelative, err = helpers.Take(data, 16)
		if err != nil {
			return formatterFlags, data, fmt.Errorf("failed to read uuid relative: %w", err)
		}
		formatterFlags.UUIDRelative = fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X%02X",
			uuidRelative[0], uuidRelative[1], uuidRelative[2], uuidRelative[3],
			uuidRelative[4], uuidRelative[5], uuidRelative[6], uuidRelative[7],
			uuidRelative[8], uuidRelative[9], uuidRelative[10], uuidRelative[11],
			uuidRelative[12], uuidRelative[13], uuidRelative[14], uuidRelative[15])
	default:
		return formatterFlags, data, fmt.Errorf("unknown flags: %x", flags&flagCheck)
	}

	return formatterFlags, data, nil
}
