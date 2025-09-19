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
	"regexp"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/firehose"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/types"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/uuidtext"
)

const (
	chunkPreambleSize = 16
	headerChunk       = 0x1000
	catalogChunk      = 0x600b
	chunksetChunk     = 0x600d
)

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
	EventType      string // "logEvent", "activityEvent", etc.
	LogType        string // "Info", "Debug", "Error", etc.
	Process        string
	ProcessUUID    string
	Message        string // Formatted final message
	RawMessage     string // Format string template
	BootUUID       string
	TimezoneName   string
	MessageEntries []interface{} // Raw message data
	Timestamp      string        // ISO formatted timestamp
}

// LogProcessor processes unified log data into individual log entries
type LogProcessor struct {
	cacheProvider *uuidtext.CacheProvider
	timesyncData  map[string]*TimesyncBoot
	messageRegex  *regexp.Regexp
	debugMode     bool
}
type UnifiedLogData struct {
	HeaderData   []HeaderChunk
	CatalogData  []UnifiedLogCatalogData
	OversizeData []OversizeChunk
}

// UnifiedLogCatalogData represents the complete unified log data
type UnifiedLogCatalogData struct {
	CatalogData    types.CatalogChunk
	FirehoseData   []firehose.Preamble
	OversizeData   []OversizeChunk
	StatedumpData  []StatedumpChunk
	SimpledumpData []SimpledumpChunk
}

func ParseUnifiedLog(data []byte) (*UnifiedLogData, error) {
	unifiedLogData := &UnifiedLogData{}
	catalogData := &UnifiedLogCatalogData{}
	var chunkData []byte

	for len(data) > 0 {
		preamble, err := DetectPreamble(data)
		if err != nil {
			return nil, err
		}
		chunkDataSize := preamble.ChunkDataSize
		if chunkDataSize > uint64(^uint(0)>>1) {
			return nil, fmt.Errorf("failed to extract string size: u64 is bigger than system usize")
		}
		data, chunkData, _ = utils.Take(data, int(chunkDataSize))

		switch preamble.ChunkTag {
		case headerChunk:
			err := getULHeaderData(chunkData, unifiedLogData)
			if err != nil {
				return nil, fmt.Errorf("failed to get header data: %w", err)
			}
		case catalogChunk:
			if catalogData.CatalogData.ChunkTag != 0 {
				unifiedLogData.CatalogData = append(unifiedLogData.CatalogData, *catalogData)
			}
			catalogData = &UnifiedLogCatalogData{}
			err := getULCatalogData(chunkData, catalogData)
			if err != nil {
				return nil, fmt.Errorf("failed to get catalog data: %w", err)
			}
		case chunksetChunk:
			err := getULChunksetData(chunkData, catalogData, unifiedLogData)
			if err != nil {
				return nil, fmt.Errorf("failed to get chunkset data: %w", err)
			}
		default:
			return nil, fmt.Errorf("unknown chunk tag: %x", preamble.ChunkTag)
		}

		paddingSize := utils.PaddingSize(chunkDataSize, 8)
		if len(data) < int(paddingSize) {
			break
		}
		if paddingSize > uint64(^uint(0)>>1) {
			return nil, fmt.Errorf("failed to extract string size: u64 is bigger than system usize")
		}
		data, _, _ = utils.Take(data, int(paddingSize))
		if len(data) == 0 {
			break
		}
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
	// TODO: make sure this is correct once the catalog chunk PR is merged
	// catalogResults, err := ParseCatalogChunk(data, catalogData)
	// if err != nil {
	// 	return err
	// }
	// catalogData.CatalogData = *catalogResults
	return nil
}

func getULChunksetData(data []byte, catalogData *UnifiedLogCatalogData, unifiedLogData *UnifiedLogData) error {
	chunksetResults, _, err := ParseChunkset(data)
	if err != nil {
		return err
	}
	if err := ParseChunksetData(chunksetResults.DecompressedData, catalogData); err != nil {
		return err
	}
	unifiedLogData.OversizeData = append(unifiedLogData.OversizeData, catalogData.OversizeData...)

	return nil
}
