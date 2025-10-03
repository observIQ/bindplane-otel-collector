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

package sharedcache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// skipIfNoTestdata skips the test if the testdata directory doesn't exist
func skipIfNoTestdata(t *testing.T) {
	testdataPath := filepath.Join("..", "..", "..", "..", "..", "receiver", "macosunifiedloggingreceiver", "testdata")
	if _, err := os.Stat(testdataPath); os.IsNotExist(err) {
		t.Skip("Skipping test: testdata directory not found")
	}
}

func TestParseDSC_VersionOne(t *testing.T) {
	skipIfNoTestdata(t)
	filePath := filepath.Join("..", "..", "..", "..", "..", "receiver", "macosunifiedloggingreceiver", "testdata", "DSC Tests", "big_sur_version_1_522F6217CB113F8FB845C2A1B784C7C2")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := ParseDSC(data)
	require.NoError(t, err)

	require.Equal(t, 532, len(results.UUIDs))
	require.Equal(t, "4DF6D8F5D9C23A968DE45E99D6B73DC8", results.UUIDs[0].UUID)
	require.Equal(t, uint32(19919502), results.UUIDs[0].PathOffset)
	require.Equal(t, uint32(8192), results.UUIDs[0].TextSize)
	require.Equal(t, uint64(73728), results.UUIDs[0].TextOffset)
	require.Equal(t, "/usr/lib/system/libsystem_blocks.dylib", results.UUIDs[0].PathString)

	require.Equal(t, 788, len(results.Ranges))
	require.Equal(t, []byte{0}, results.Ranges[0].Strings)
	require.Equal(t, uint64(0), results.Ranges[0].UnknownUUIDIndex)
	require.Equal(t, uint64(80296), results.Ranges[0].RangeOffset)
	require.Equal(t, uint32(1), results.Ranges[0].RangeSize)

	require.Equal(t, uint32(1685283688), results.Signature)
	require.Equal(t, uint16(1), results.MajorVersion)
	require.Equal(t, uint16(0), results.MinorVersion)
	require.Equal(t, "", results.DSCUUID)
	require.Equal(t, uint32(788), results.NumberRanges)
	require.Equal(t, uint32(532), results.NumberUUIDs)
}

func TestParseDSC_InvalidSignature(t *testing.T) {
	// 4 bytes signature that does NOT match 0x64736368 ("dsch")
	data := []byte{0x00, 0x00, 0x00, 0x00}

	_, err := ParseDSC(data)
	if err == nil {
		t.Fatalf("expected error for invalid signature, got nil")
	}
	if !strings.Contains(err.Error(), "invalid DSC signature") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseDSC_ShortInputSignature(t *testing.T) {
	// Fewer than 4 bytes should fail reading signature
	data := []byte{0x01, 0x02, 0x03}

	_, err := ParseDSC(data)
	if err == nil {
		t.Fatalf("expected error for short input, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read signature") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractSharedString_DynamicFormatter(t *testing.T) {
	s := &Strings{
		UUIDs: []UUIDDescriptor{{
			PathString: "libExample.dylib",
			UUID:       "ABCDEF0123456789ABCDEF0123456789",
		}},
	}

	// High bit set indicates dynamic formatter ("%s")
	msg, err := s.ExtractSharedString(0x80000001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.FormatString != "%s" {
		t.Fatalf("expected format '%%s', got %q", msg.FormatString)
	}
	if msg.Library != "libExample.dylib" || msg.LibraryUUID != "ABCDEF0123456789ABCDEF0123456789" {
		t.Fatalf("unexpected library fields: %+v", msg)
	}
}

func TestExtractSharedString_NotFound(t *testing.T) {
	s := &Strings{}
	_, err := s.ExtractSharedString(12345)
	if err == nil {
		t.Fatalf("expected not found error, got nil")
	}
	if !strings.Contains(err.Error(), "shared string not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractSharedString_Found(t *testing.T) {
	// Create a range where the string starts 5 bytes after RangeOffset
	base := uint64(100)
	raw := make([]byte, 5)
	raw = append(raw, []byte("hello\x00trailing")...)

	s := &Strings{
		Ranges: []RangeDescriptor{{
			RangeOffset:      base,
			DataOffset:       0,  // not used by ExtractSharedString
			RangeSize:        64, // large enough to include offset and string
			UnknownUUIDIndex: 0,
			Strings:          raw,
		}},
		UUIDs: []UUIDDescriptor{{
			PathString: "libStrings.dylib",
			UUID:       "00112233445566778899AABBCCDDEEFF",
		}},
	}

	msg, err := s.ExtractSharedString(base + 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.FormatString != "hello" {
		t.Fatalf("expected 'hello', got %q", msg.FormatString)
	}
	if msg.Library != "libStrings.dylib" || msg.LibraryUUID != "00112233445566778899AABBCCDDEEFF" {
		t.Fatalf("unexpected library fields: %+v", msg)
	}
}
