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
	"fmt"
	"time"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/firehose"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/models"
)

const (
	chunkPreambleSize = 16
	headerChunk       = 0x1000
	catalogChunk      = 0x600b
	chunksetChunk     = 0x600d
)

type EventType uint8

const (
	EventTypeUnknown EventType = iota
	EventTypeLog
	EventTypeActivity
	EventTypeTrace
	EventTypeSignpost
	EventTypeSimpledump
	EventTypeStatedump
	EventTypeLoss
)

func GetEventType(eventType uint8) EventType {
	switch eventType {
	case 0x4:
		return EventTypeLog
	case 0x2:
		return EventTypeActivity
	case 0x3:
		return EventTypeTrace
	case 0x6:
		return EventTypeSignpost
	case 0x7:
		return EventTypeLoss
	default:
		return EventTypeUnknown
	}
}

// LogType represents the type of log entry
type LogType uint8

const (
	LogTypeDebug LogType = iota
	LogTypeInfo
	LogTypeDefault
	LogTypeError
	LogTypeFault
	LogTypeCreate
	LogTypeUseraction
	LogTypeProcessSignpostEvent
	LogTypeProcessSignpostStart
	LogTypeProcessSignpostEnd
	LogTypeSystemSignpostEvent
	LogTypeSystemSignpostStart
	LogTypeSystemSignpostEnd
	LogTypeThreadSignpostEvent
	LogTypeThreadSignpostStart
	LogTypeThreadSignpostEnd
	LogTypeSimpledump
	LogTypeStatedump
	LogTypeLoss
)

// String returns the string representation of the LogType
func (lt LogType) String() string {
	switch lt {
	case LogTypeDebug:
		return "Debug"
	case LogTypeInfo:
		return "Info"
	case LogTypeDefault:
		return "Default"
	case LogTypeError:
		return "Error"
	case LogTypeFault:
		return "Fault"
	case LogTypeCreate:
		return "Create"
	case LogTypeUseraction:
		return "Useraction"
	case LogTypeProcessSignpostEvent:
		return "ProcessSignpostEvent"
	case LogTypeProcessSignpostStart:
		return "ProcessSignpostStart"
	case LogTypeProcessSignpostEnd:
		return "ProcessSignpostEnd"
	case LogTypeSystemSignpostEvent:
		return "SystemSignpostEvent"
	case LogTypeSystemSignpostStart:
		return "SystemSignpostStart"
	case LogTypeSystemSignpostEnd:
		return "SystemSignpostEnd"
	case LogTypeThreadSignpostEvent:
		return "ThreadSignpostEvent"
	case LogTypeThreadSignpostStart:
		return "ThreadSignpostStart"
	case LogTypeThreadSignpostEnd:
		return "ThreadSignpostEnd"
	case LogTypeSimpledump:
		return "Simpledump"
	case LogTypeStatedump:
		return "Statedump"
	case LogTypeLoss:
		return "Loss"
	default:
		return "Unknown"
	}
}

// GetLogType converts raw log type and activity type values to LogType enum
// This is equivalent to the Rust LogData::get_log_type function
func GetLogType(logType uint8, activityType uint8) LogType {
	switch logType {
	case 0x1:
		if activityType == 0x2 {
			return LogTypeCreate
		}
		return LogTypeUseraction
	case 0x2:
		return LogTypeDebug
	case 0x3:
		return LogTypeError
	case 0x4:
		return LogTypeFault
	case 0x5:
		return LogTypeDefault
	case 0x6:
		return LogTypeInfo
	case 0x7:
		return LogTypeLoss
	default:
		return LogTypeDefault
	}
}

// LogEntry represents a processed unified log entry (equivalent to Rust LogData)
type LogEntry struct {
	Subsystem      string
	ThreadID       uint64
	PID            uint64
	EUID           uint32
	Library        string
	LibraryUUID    string
	ActivityID     uint64
	Time           float64 // Unix timestamp as float
	Category       string
	EventType      EventType // EventType enum value
	LogType        LogType   // LogType enum value
	Process        string
	ProcessUUID    string
	Message        string // Formatted final message
	RawMessage     string // Format string template
	BootUUID       string
	TimezoneName   string
	MessageEntries []firehose.ItemInfo
	Timestamp      time.Time // ISO formatted timestamp
}

// UnifiedLogData aggregates parsed data from a unified logging archive,
// including header, catalog, oversize, and related chunk data.
type UnifiedLogData struct {
	HeaderData   []HeaderChunk
	CatalogData  []UnifiedLogCatalogData
	OversizeData []OversizeChunk
}

// UnifiedLogCatalogData represents the complete unified log catalog data
type UnifiedLogCatalogData struct {
	CatalogData    models.CatalogChunk
	FirehoseData   []firehose.Preamble
	OversizeData   []OversizeChunk
	StatedumpData  []StatedumpChunk
	SimpledumpData []SimpledumpChunk
}

// ParseUnifiedLog parses a unified logging byte stream and returns a
// fully-populated UnifiedLogData structure containing discovered chunks.
func ParseUnifiedLog(data []byte) (*UnifiedLogData, error) {
	unifiedLogData := &UnifiedLogData{}
	catalogData := &UnifiedLogCatalogData{}
	var chunkData []byte

	for len(data) > 0 {

		if len(data) < chunkPreambleSize {
			break
		}

		preamble, err := DetectPreamble(data)
		if err != nil {
			return unifiedLogData, err
		}
		chunkDataSize := preamble.ChunkDataSize
		if chunkDataSize > uint64(^uint(0)>>1) {
			return unifiedLogData, fmt.Errorf("failed to extract string size: u64 is bigger than system usize")
		}
		data, chunkData, _ = helpers.Take(data, int(chunkDataSize)+chunkPreambleSize)

		switch preamble.ChunkTag {
		case headerChunk:
			err := getULHeaderData(chunkData, unifiedLogData)
			if err != nil {
				return unifiedLogData, fmt.Errorf("failed to get header data: %w", err)
			}
		case catalogChunk:
			if catalogData.CatalogData.ChunkTag != 0 {
				unifiedLogData.CatalogData = append(unifiedLogData.CatalogData, *catalogData)
			}
			catalogData = &UnifiedLogCatalogData{}
			err := getULCatalogData(chunkData, catalogData)
			if err != nil {
				return unifiedLogData, fmt.Errorf("failed to get catalog data: %w", err)
			}
		case chunksetChunk:
			err := getULChunksetData(chunkData, catalogData, unifiedLogData)
			if err != nil {
				return unifiedLogData, fmt.Errorf("failed to get chunkset data: %w", err)
			}
		default:
			return unifiedLogData, fmt.Errorf("unknown chunk tag: %x", preamble.ChunkTag)
		}

		// Calculate and consume padding
		paddingSize := helpers.PaddingSize(chunkDataSize, 8)
		if len(data) < int(paddingSize) {
			break
		}
		if paddingSize > uint64(^uint(0)>>1) {
			return unifiedLogData, fmt.Errorf("failed to extract string size: u64 is bigger than system usize")
		}
		data, _, _ = helpers.Take(data, int(paddingSize))
		// Check if we have enough data for a preamble
		if len(data) < chunkPreambleSize {
			// TODO: Log this warning
			// fmt.Printf("Not enough data for preamble header, needed 16 bytes. Got: %d\n", len(data))
			break
		}
	}
	if catalogData.CatalogData.ChunkTag != 0 {
		unifiedLogData.CatalogData = append(unifiedLogData.CatalogData, *catalogData)
	}
	return unifiedLogData, nil
}

func getULHeaderData(data []byte, unifiedLogData *UnifiedLogData) error {
	headerResults, err := ParseHeaderChunk(data)
	if err != nil {
		return err
	}
	unifiedLogData.HeaderData = append(unifiedLogData.HeaderData, *headerResults)
	return nil
}

func getULCatalogData(data []byte, catalogData *UnifiedLogCatalogData) error {
	catalogResults, _, err := ParseCatalogChunk(data)
	if err != nil {
		return err
	}
	catalogData.CatalogData = catalogResults
	return nil
}

func getULChunksetData(chunkData []byte, catalogData *UnifiedLogCatalogData, unifiedLogData *UnifiedLogData) error {
	chunksetResults, _, err := ParseChunkset(chunkData)
	if err != nil {
		return err
	}
	if err := ParseChunksetData(chunksetResults.DecompressedData, catalogData); err != nil {
		return err
	}
	unifiedLogData.OversizeData = append(unifiedLogData.OversizeData, catalogData.OversizeData...)

	return nil
}
