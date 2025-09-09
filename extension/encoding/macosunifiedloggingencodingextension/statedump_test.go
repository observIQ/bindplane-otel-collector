// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package macosunifiedloggingencodingextension

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStateDump(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "statedump", "statedump_test_plist.bin"))
	require.NoError(t, err)

	statedump, err := ParseStateDump(data)
	require.NoError(t, err)

	// Verify basic structure
	require.Equal(t, uint32(0x6003), statedump.ChunkTag)
	require.Equal(t, uint32(0x0001), statedump.ChunkSubtag)
	require.Equal(t, uint32(1), statedump.UnknownDataType) // Plist type
	require.Equal(t, "Test Plist", statedump.TitleName)

	// Verify the actual data pattern (0xaa, 0xab, 0xac... repeating)
	require.Equal(t, 50, len(statedump.Data))
	require.Equal(t, byte(0xaa), statedump.Data[0])
	require.Equal(t, byte(0xab), statedump.Data[1])
	require.Equal(t, byte(0xac), statedump.Data[2])

	// Verify the repeating pattern
	for i := 0; i < len(statedump.Data); i++ {
		expected := byte(0xaa + (i % 10))
		require.Equal(t, expected, statedump.Data[i], "Data mismatch at index %d", i)
	}
}

func TestParseStateDumpCustomObject(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "statedump", "statedump_test_valid.bin"))
	require.NoError(t, err)

	statedump, err := ParseStateDump(data)
	require.NoError(t, err)

	// Verify basic structure
	require.Equal(t, uint32(0x6003), statedump.ChunkTag)
	require.Equal(t, uint32(0x0002), statedump.ChunkSubtag)
	require.Equal(t, uint32(3), statedump.UnknownDataType) // Custom object type
	require.Equal(t, "CLClientManagerStateTracker", statedump.TitleName)
	require.Equal(t, "TestLibrary", statedump.DecoderLibrary)
	require.Equal(t, "TestType", statedump.DecoderType)

	// Verify the actual data pattern (0, 1, 2, 3... repeating)
	require.Equal(t, 100, len(statedump.Data))
	for i := 0; i < len(statedump.Data); i++ {
		expected := byte(i % 256)
		require.Equal(t, expected, statedump.Data[i], "Data mismatch at index %d", i)
	}
}

func TestLoadStatedumpsFromTraceV3(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "statedump", "test_tracev3_with_statedumps.tracev3"))
	require.NoError(t, err)

	collection, err := LoadStatedumpsFromTraceV3(data)
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
	// Test plist statedump parsing
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

	collection, err := LoadStatedumpsFromTraceV3(tracev3Data)
	require.NoError(t, err)

	require.Greater(t, collection.Timestamp, uint64(0))
	require.GreaterOrEqual(t, len(collection.Statedumps), 1)
}
