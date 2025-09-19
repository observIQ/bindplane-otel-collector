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

// HeaderChunk represents a parsed header chunk
type HeaderChunk struct {
	ChunkTag               uint32
	ChunkSubTag            uint32
	ChunkDataSize          uint64
	MachTimeNumerator      uint32
	MachTimeDenominator    uint32
	ContinuousTime         uint64
	UnknownTime            uint64
	Unknown                uint32
	BiasMin                uint32
	DaylightSavings        uint32
	UnknownFlags           uint32
	SubChunkTag            uint32
	SubChunkDataSize       uint32
	SubChunkContinuousTime uint64
	SubChunkTag2           uint32
	SubChunkDataSize2      uint32
	Unknown2               uint32
	Unknown3               uint32
	BuildVersionString     string
	HardwareModelString    string
	SubChunkTag3           uint32
	SubChunkDataSize3      uint32
	BootUUID               string
	LogdPID                uint32
	LogdExitStatus         uint32
	SubChunkTag4           uint32
	SubChunkDataSize4      uint32
	TimezonePath           string
}

var (
	hardwareModelSize = 32
	timezonePathSize  = 48
)

func ParseHeaderChunk(data []byte) (*HeaderChunk, error) {
	header := &HeaderChunk{}
	var chunkTag, chunkSubtag, chunkDataSize, machTimeNumerator, machTimeDenominator, continuousTime, unknownTime, unknown, biasMin, daylightSavings, unknownFlags, subChunkTag, subChunkDataSize, subChunkContinuousTime, subChunkTag2, subChunkDataSize2, unknown2, unknown3, buildVersionString, hardwareModelString, timezonePath, subChunkTag3, subChunkDataSize3, bootUUID, logdPID, logdExitStatus, subChunkTag4, subChunkDataSize4 []byte

	data, chunkTag, _ = utils.Take(data, 4)
	data, chunkSubtag, _ = utils.Take(data, 4)
	data, chunkDataSize, _ = utils.Take(data, 8)
	data, machTimeNumerator, _ = utils.Take(data, 4)
	data, machTimeDenominator, _ = utils.Take(data, 4)
	data, continuousTime, _ = utils.Take(data, 8)
	data, unknownTime, _ = utils.Take(data, 8)
	data, unknown, _ = utils.Take(data, 4)
	data, biasMin, _ = utils.Take(data, 4)
	data, daylightSavings, _ = utils.Take(data, 4)
	data, unknownFlags, _ = utils.Take(data, 4)
	data, subChunkTag, _ = utils.Take(data, 4)
	data, subChunkDataSize, _ = utils.Take(data, 4)
	data, subChunkContinuousTime, _ = utils.Take(data, 8)
	data, subChunkTag2, _ = utils.Take(data, 4)
	data, subChunkDataSize2, _ = utils.Take(data, 4)
	data, unknown2, _ = utils.Take(data, 4)
	data, unknown3, _ = utils.Take(data, 4)
	data, buildVersionString, _ = utils.Take(data, 16)
	data, hardwareModelString, _ = utils.Take(data, hardwareModelSize)
	data, subChunkTag3, _ = utils.Take(data, 4)
	data, subChunkDataSize3, _ = utils.Take(data, 4)
	data, bootUUID, _ = utils.Take(data, 16)
	data, logdPID, _ = utils.Take(data, 4)
	data, logdExitStatus, _ = utils.Take(data, 4)
	data, subChunkTag4, _ = utils.Take(data, 4)
	data, subChunkDataSize4, _ = utils.Take(data, 4)
	data, timezonePath, _ = utils.Take(data, 48)

	header.ChunkTag = binary.LittleEndian.Uint32(chunkTag)
	header.ChunkSubTag = binary.LittleEndian.Uint32(chunkSubtag)
	header.ChunkDataSize = binary.LittleEndian.Uint64(chunkDataSize)
	header.MachTimeNumerator = binary.LittleEndian.Uint32(machTimeNumerator)
	header.MachTimeDenominator = binary.LittleEndian.Uint32(machTimeDenominator)
	header.ContinuousTime = binary.LittleEndian.Uint64(continuousTime)
	header.UnknownTime = binary.LittleEndian.Uint64(unknownTime)
	header.Unknown = binary.LittleEndian.Uint32(unknown)
	header.BiasMin = binary.LittleEndian.Uint32(biasMin)
	header.DaylightSavings = binary.LittleEndian.Uint32(daylightSavings)
	header.UnknownFlags = binary.LittleEndian.Uint32(unknownFlags)
	header.SubChunkTag = binary.LittleEndian.Uint32(subChunkTag)
	header.SubChunkDataSize = binary.LittleEndian.Uint32(subChunkDataSize)
	header.SubChunkContinuousTime = binary.LittleEndian.Uint64(subChunkContinuousTime)
	header.SubChunkTag2 = binary.LittleEndian.Uint32(subChunkTag2)
	header.SubChunkDataSize2 = binary.LittleEndian.Uint32(subChunkDataSize2)
	header.Unknown2 = binary.LittleEndian.Uint32(unknown2)
	header.Unknown3 = binary.LittleEndian.Uint32(unknown3)
	header.SubChunkTag3 = binary.LittleEndian.Uint32(subChunkTag3)
	header.SubChunkDataSize3 = binary.LittleEndian.Uint32(subChunkDataSize3)
	header.LogdPID = binary.LittleEndian.Uint32(logdPID)
	header.LogdExitStatus = binary.LittleEndian.Uint32(logdExitStatus)
	header.SubChunkTag4 = binary.LittleEndian.Uint32(subChunkTag4)
	header.SubChunkDataSize4 = binary.LittleEndian.Uint32(subChunkDataSize4)

	// Convert byte arrays to strings, trimming null terminators
	header.TimezonePath = strings.TrimRight(string(timezonePath), "\x00")
	header.BuildVersionString = strings.TrimRight(string(buildVersionString), "\x00")
	header.HardwareModelString = strings.TrimRight(string(hardwareModelString), "\x00")

	// Convert UUID from big-endian bytes to formatted hex string
	if len(bootUUID) >= 16 {
		upper := binary.BigEndian.Uint64(bootUUID[:8])
		lower := binary.BigEndian.Uint64(bootUUID[8:16])
		header.BootUUID = fmt.Sprintf("%016X%016X", upper, lower)
	}

	return header, nil
}
