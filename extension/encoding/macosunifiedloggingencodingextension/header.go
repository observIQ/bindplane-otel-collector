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

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
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
	var err error

	data, chunkTag, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk tag: %w", err)
	}
	data, chunkSubtag, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk sub tag: %w", err)
	}
	data, chunkDataSize, err = helpers.Take(data, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk data size: %w", err)
	}
	data, machTimeNumerator, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read mach time numerator: %w", err)
	}
	data, machTimeDenominator, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read mach time denominator: %w", err)
	}
	data, continuousTime, err = helpers.Take(data, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to read continuous time: %w", err)
	}
	data, unknownTime, err = helpers.Take(data, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to read unknown time: %w", err)
	}
	data, unknown, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read unknown: %w", err)
	}
	data, biasMin, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read bias min: %w", err)
	}
	data, daylightSavings, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read daylight savings: %w", err)
	}
	data, unknownFlags, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read unknown flags: %w", err)
	}
	data, subChunkTag, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read sub chunk tag: %w", err)
	}
	data, subChunkDataSize, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read sub chunk data size: %w", err)
	}
	data, subChunkContinuousTime, err = helpers.Take(data, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to read sub chunk continuous time: %w", err)
	}
	data, subChunkTag2, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read sub chunk tag 2: %w", err)
	}
	data, subChunkDataSize2, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read sub chunk data size 2: %w", err)
	}
	data, unknown2, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read unknown 2: %w", err)
	}
	data, unknown3, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read unknown 3: %w", err)
	}
	data, buildVersionString, err = helpers.Take(data, 16)
	if err != nil {
		return nil, fmt.Errorf("failed to read build version string: %w", err)
	}
	data, hardwareModelString, err = helpers.Take(data, hardwareModelSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read hardware model string: %w", err)
	}
	data, subChunkTag3, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read sub chunk tag 3: %w", err)
	}
	data, subChunkDataSize3, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read sub chunk data size 3: %w", err)
	}
	data, bootUUID, err = helpers.Take(data, 16)
	if err != nil {
		return nil, fmt.Errorf("failed to read boot UUID: %w", err)
	}
	data, logdPID, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read logd PID: %w", err)
	}
	data, logdExitStatus, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read logd exit status: %w", err)
	}
	data, subChunkTag4, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read sub chunk tag 4: %w", err)
	}
	data, subChunkDataSize4, err = helpers.Take(data, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to read sub chunk data size 4: %w", err)
	}
	data, timezonePath, err = helpers.Take(data, 48)
	if err != nil {
		return nil, fmt.Errorf("failed to read timezone path: %w", err)
	}

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
