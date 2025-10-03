// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package macosunifiedloggingencodingextension

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
	"github.com/stretchr/testify/require"
)

func TestParseStateDump(t *testing.T) {
	skipIfNoTestdata(t)
	data, err := os.ReadFile(filepath.Join("testdata", "statedump", "statedump_test_plist.bin"))
	require.NoError(t, err)

	statedump, err := ParseStateDump(data)
	require.NoError(t, err)

	// Verify basic structure
	require.Equal(t, uint32(0x6003), statedump.ChunkTag)
	require.Equal(t, uint32(0x0001), statedump.ChunkSubtag)
	require.Equal(t, uint32(1), statedump.UnknownDataType)
	require.Equal(t, "Test Plist", statedump.TitleName)

	require.Equal(t, 50, len(statedump.Data))
	require.Equal(t, byte(0xaa), statedump.Data[0])
	require.Equal(t, byte(0xab), statedump.Data[1])
	require.Equal(t, byte(0xac), statedump.Data[2])

	for i := 0; i < len(statedump.Data); i++ {
		expected := byte(0xaa + (i % 10))
		require.Equal(t, expected, statedump.Data[i], "Data mismatch at index %d", i)
	}
}

func TestParseStateDumpCustomObject(t *testing.T) {
	skipIfNoTestdata(t)
	data, err := os.ReadFile(filepath.Join("testdata", "statedump", "statedump_test_valid.bin"))
	require.NoError(t, err)

	statedump, err := ParseStateDump(data)
	require.NoError(t, err)

	require.Equal(t, uint32(0x6003), statedump.ChunkTag)
	require.Equal(t, uint32(0x0002), statedump.ChunkSubtag)
	require.Equal(t, uint32(3), statedump.UnknownDataType) // Custom object type
	require.Equal(t, "CLClientManagerStateTracker", statedump.TitleName)
	require.Equal(t, "TestLibrary", statedump.DecoderLibrary)
	require.Equal(t, "TestType", statedump.DecoderType)

	require.Equal(t, 100, len(statedump.Data))
	for i := 0; i < len(statedump.Data); i++ {
		expected := byte(i % 256)
		require.Equal(t, expected, statedump.Data[i], "Data mismatch at index %d", i)
	}
}

func TestLoadStatedumpsFromTraceV3(t *testing.T) {
	skipIfNoTestdata(t)
	data, err := os.ReadFile(filepath.Join("testdata", "statedump", "test_tracev3_with_statedumps.tracev3"))
	require.NoError(t, err)

	collection, err := loadStatedumpsFromTracev3(data)
	require.NoError(t, err)

	// Verify basic collection properties
	require.NotNil(t, collection)
	// Note: BootUUID might be empty in test data, that's ok
	require.GreaterOrEqual(t, collection.Timestamp, uint64(0))
}

func TestParseStateDumpErrors(t *testing.T) {
	// Test with data too small
	smallData := make([]byte, 50) // Less than required 72 bytes
	_, err := ParseStateDump(smallData)
	require.Error(t, err)
	require.Contains(t, err.Error(), "statedump data too small")

	// Test with empty data
	_, err = ParseStateDump(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "statedump data too small")
}

func TestExpectedStructsFromTestData(t *testing.T) {
	skipIfNoTestdata(t)
	plistData, err := os.ReadFile(filepath.Join("testdata", "statedump", "statedump_test_plist.bin"))
	require.NoError(t, err)

	plistSD, err := ParseStateDump(plistData)
	require.NoError(t, err)

	require.Equal(t, uint32(0x6003), plistSD.ChunkTag)
	require.Equal(t, uint32(0x0001), plistSD.ChunkSubtag)
	require.Equal(t, uint32(1), plistSD.UnknownDataType)
	require.Equal(t, "Test Plist", plistSD.TitleName)
	require.Equal(t, 50, len(plistSD.Data))

	// Verify data pattern (0xaa + (i % 10))
	for i := 0; i < len(plistSD.Data); i++ {
		expected := byte(0xaa + (i % 10))
		require.Equal(t, expected, plistSD.Data[i])
	}

	// Test custom object statedump parsing
	customData, err := os.ReadFile(filepath.Join("testdata", "statedump", "statedump_test_valid.bin"))
	require.NoError(t, err)

	customSD, err := ParseStateDump(customData)
	require.NoError(t, err)

	// validate the chunk tag, subtag, and unknown data type
	require.Equal(t, uint32(0x6003), customSD.ChunkTag)
	require.Equal(t, uint32(0x0002), customSD.ChunkSubtag)
	require.Equal(t, uint32(3), customSD.UnknownDataType)
	require.Equal(t, "CLClientManagerStateTracker", customSD.TitleName)
	require.Equal(t, "TestLibrary", customSD.DecoderLibrary)
	require.Equal(t, "TestType", customSD.DecoderType)
	require.Equal(t, 100, len(customSD.Data))

	// Verify data pattern (i % 256)
	for i := 0; i < len(customSD.Data); i++ {
		expected := byte(i % 256)
		require.Equal(t, expected, customSD.Data[i])
	}

	// Test TraceV3 collection parsing
	tracev3Data, err := os.ReadFile(filepath.Join("testdata", "statedump", "test_tracev3_with_statedumps.tracev3"))
	require.NoError(t, err)

	collection, err := loadStatedumpsFromTracev3(tracev3Data)
	require.NoError(t, err)

	require.Greater(t, collection.Timestamp, uint64(0))
	require.GreaterOrEqual(t, len(collection.Statedumps), 1)
}

// valid test data for the CLDaemonStatusStateTracker
func TestParseStateDumpDaemonStatus(t *testing.T) {
	testData := []byte{
		3, 96, 0, 0, 0, 0, 0, 0, 32, 1, 0, 0, 0, 0, 0, 0, 113, 0, 0, 0, 0, 0, 0, 0, 208, 1, 0,
		0, 14, 0, 0, 0, 13, 179, 213, 232, 0, 0, 0, 0, 118, 4, 0, 0, 0, 0, 0, 128, 92, 216,
		221, 238, 4, 56, 58, 56, 136, 119, 16, 34, 124, 90, 10, 86, 3, 0, 0, 0, 40, 0, 0, 0,
		108, 111, 99, 97, 116, 105, 111, 110, 0, 0, 187, 44, 255, 127, 0, 0, 42, 144, 225, 173,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 83, 78, 41, 126, 255, 127,
		0, 0, 6, 144, 225, 173, 0, 0, 0, 0, 148, 242, 123, 124, 255, 127, 0, 0, 95, 67, 76, 68,
		97, 101, 109, 111, 110, 83, 116, 97, 116, 117, 115, 83, 116, 97, 116, 101, 84, 114, 97,
		99, 107, 101, 114, 83, 116, 97, 116, 101, 0, 144, 225, 173, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 72, 78, 41, 126, 255, 127, 0, 0, 67, 76, 68, 97, 101,
		109, 111, 110, 83, 116, 97, 116, 117, 115, 83, 116, 97, 116, 101, 84, 114, 97, 99, 107,
		101, 114, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 240, 191, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0,
	}

	statedump, err := ParseStateDump(testData)
	require.NoError(t, err)

	require.Equal(t, uint32(24579), statedump.ChunkTag) // 0x6003
	require.Equal(t, uint32(0), statedump.ChunkSubtag)
	require.Equal(t, uint64(288), statedump.ChunkDataSize)
	require.Equal(t, uint64(113), statedump.FirstProcID)
	require.Equal(t, uint32(464), statedump.SecondProcID)
	require.Equal(t, uint8(14), statedump.TTL)
	require.Equal(t, []uint8{0, 0, 0}, statedump.UnknownReserved)
	require.Equal(t, uint64(3906319117), statedump.ContinuousTime)
	require.Equal(t, uint64(9223372036854776950), statedump.ActivityID)
	require.Equal(t, "5CD8DDEE04383A38887710227C5A0A56", statedump.UUID)
	require.Equal(t, uint32(3), statedump.UnknownDataType) // Custom object
	require.Equal(t, uint32(40), statedump.UnknownDataSize)
	require.Equal(t, "location", statedump.DecoderLibrary)
	require.Equal(t, "_CLDaemonStatusStateTrackerState", statedump.DecoderType)
	require.Equal(t, "CLDaemonStatusStateTracker", statedump.TitleName)

	expectedData := []byte{
		0, 0, 0, 0, 0, 0, 240, 191, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0,
		0, 0, 255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	require.Equal(t, expectedData, statedump.Data)
}

// valid test data for the plist statedump
func TestParseStateDumpPlist(t *testing.T) {
	testData := make([]byte, 300)
	binary.LittleEndian.PutUint32(testData[0:4], 0x6003) // ChunkTag
	binary.LittleEndian.PutUint32(testData[4:8], 0)      // ChunkSubtag
	binary.LittleEndian.PutUint64(testData[8:16], 7859)  // ChunkDataSize
	binary.LittleEndian.PutUint64(testData[16:24], 1771) // FirstProcID
	binary.LittleEndian.PutUint32(testData[24:28], 1772) // SecondProcID
	testData[28] = 14                                    // TTL
	testData[29] = 0
	testData[30] = 0
	testData[31] = 0 // UnknownReserved

	binary.LittleEndian.PutUint64(testData[32:40], 10048439123292)      // ContinuousTime
	binary.LittleEndian.PutUint64(testData[40:48], 9223372036854833248) // ActivityID

	uuidBytes := []byte{0x7E, 0x5D, 0x98, 0x55, 0xD1, 0xCC, 0x38, 0x2D, 0xB8, 0xEB, 0x25, 0xA0, 0x30, 0xD4, 0xB2, 0xFA}
	copy(testData[48:64], uuidBytes)

	binary.LittleEndian.PutUint32(testData[64:68], 1)  // UnknownDataType (plist)
	binary.LittleEndian.PutUint32(testData[68:72], 20) // UnknownDataSize (smaller for test)

	titleStart := 72 + 128
	copy(testData[titleStart:titleStart+16], []byte("WebContent state"))

	dataStart := titleStart + 64
	copy(testData[dataStart:dataStart+20], []byte("bplist00testdata123")) // 20 bytes of dummy plist data

	statedump, err := ParseStateDump(testData)
	require.NoError(t, err)

	require.Equal(t, uint32(0x6003), statedump.ChunkTag)
	require.Equal(t, uint32(0), statedump.ChunkSubtag)
	require.Equal(t, uint64(7859), statedump.ChunkDataSize)
	require.Equal(t, uint64(1771), statedump.FirstProcID)
	require.Equal(t, uint32(1772), statedump.SecondProcID)
	require.Equal(t, uint8(14), statedump.TTL)
	require.Equal(t, []uint8{0, 0, 0}, statedump.UnknownReserved)
	require.Equal(t, uint64(10048439123292), statedump.ContinuousTime)
	require.Equal(t, uint64(9223372036854833248), statedump.ActivityID)
	require.Equal(t, "7E5D9855D1CC382DB8EB25A030D4B2FA", statedump.UUID)
	require.Equal(t, uint32(1), statedump.UnknownDataType)
	require.Equal(t, uint32(20), statedump.UnknownDataSize)
	require.Equal(t, "", statedump.DecoderLibrary)
	require.Equal(t, "", statedump.DecoderType)
	require.Contains(t, statedump.TitleName, "WebContent state")
	require.Equal(t, 20, len(statedump.Data))
}

// valid test data for the custom object statedump
func TestParseStateDumpObject(t *testing.T) {
	testData := []byte{
		0, 0, 0, 0, 0, 0, 240, 191, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0,
		255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0,
	}

	result := ParseStateDumpObject(testData, "CLDaemonStatusStateTracker")

	require.Contains(t, result, "CLDaemonStatusStateTracker")
	require.NotEmpty(t, result)

	require.NotContains(t, result, "Unsupported Statedump object")
}

func loadStatedumpsFromTracev3(data []byte) (*StatedumpCollection, error) {

	if len(data) == 0 {
		return nil, fmt.Errorf("empty tracev3 data")
	}

	var headerChunk *HeaderChunk
	var err error

	// Parse the header first to get boot UUID and timing info

	headerChunk, err = ParseHeaderChunk(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tracev3 header: %w", err)
	}

	collection := &StatedumpCollection{
		Statedumps: make([]StatedumpChunk, 0),
		BootUUID:   headerChunk.BootUUID,
		Timestamp:  headerChunk.ContinuousTime,
	}

	// Parse data section for statedump chunks
	remainingData := data[headerChunk.ChunkDataSize:]
	offset := 0
	const chunkPreambleSize = 16

	for offset < len(remainingData) {
		if offset+chunkPreambleSize > len(remainingData) {
			break
		}

		// Parse preamble
		chunkTag := binary.LittleEndian.Uint32(remainingData[offset:])
		chunkDataSize := binary.LittleEndian.Uint64(remainingData[offset+8:])

		// Validate chunk data size
		if chunkDataSize == 0 || chunkDataSize > uint64(len(remainingData)) {
			offset += 4
			continue
		}

		totalChunkSize := chunkPreambleSize + int(chunkDataSize)
		if offset+totalChunkSize > len(remainingData) {
			break
		}

		if chunkTag == 0x6003 {
			// Extract statedump data (skip preamble)
			statedumpData := remainingData[offset+chunkPreambleSize : offset+totalChunkSize]

			statedump, err := ParseStateDump(statedumpData)
			if err == nil {
				collection.Statedumps = append(collection.Statedumps, *statedump)
			}
		}

		// Move to next chunk with 8-byte alignment padding
		offset += totalChunkSize + int(helpers.PaddingSize(chunkDataSize, 8))

		// Safety limit
		if len(collection.Statedumps) >= 1000 {
			break
		}
	}

	return collection, nil
}
