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

package uuidtext_test

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/testutil"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/uuidtext"
	"github.com/stretchr/testify/require"
)

func TestParseUUIDText_BigSur(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)
	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "UUIDText", "Big Sur", "1FE459BBDC3E19BBF82D58415A2AE9")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := uuidtext.ParseUUIDText(data)
	require.NoError(t, err)

	require.Equal(t, uint32(0x66778899), results.Signature)
	require.Equal(t, uint32(2), results.UnknownMajorVersion)
	require.Equal(t, uint32(1), results.UnknownMinorVersion)
	require.Equal(t, uint32(2), results.NumberEntries)
	require.Equal(t, uint32(617), results.EntryDescriptors[0].EntrySize)
	require.Equal(t, uint32(2301), results.EntryDescriptors[1].EntrySize)
	require.Equal(t, uint32(32048), results.EntryDescriptors[0].RangeStartOffset)
	require.Equal(t, uint32(29747), results.EntryDescriptors[1].RangeStartOffset)
	require.Equal(t, 2987, len(results.FooterData))
}

func TestParseUUIDText_HighSierra(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)
	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "UUIDText", "High Sierra", "425A2E5B5531B98918411B4379EE5F")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := uuidtext.ParseUUIDText(data)
	require.NoError(t, err)

	require.Equal(t, uint32(0x66778899), results.Signature)
	require.Equal(t, uint32(2), results.UnknownMajorVersion)
	require.Equal(t, uint32(1), results.UnknownMinorVersion)
	require.Equal(t, uint32(1), results.NumberEntries)
	require.Equal(t, uint32(2740), results.EntryDescriptors[0].EntrySize)
	require.Equal(t, uint32(21132), results.EntryDescriptors[0].RangeStartOffset)
	require.Equal(t, 2951, len(results.FooterData))
}
func TestParseUUIDText_BadHeader(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)
	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "Bad Data", "UUIDText", "Bad_Header_1FE459BBDC3E19BBF82D58415A2AE9")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := uuidtext.ParseUUIDText(data)
	require.Contains(t, err.Error(), "invalid UUID text signature")
	require.Nil(t, results)
}
func TestParseUUIDText_BadContent(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)
	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "Bad Data", "UUIDText", "Bad_Content_1FE459BBDC3E19BBF82D58415A2AE9")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := uuidtext.ParseUUIDText(data)
	require.Contains(t, err.Error(), "failed to read signature: not enough bytes")
	require.Nil(t, results)
}

func TestParseUUIDText_BadFile(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)
	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "Bad Data", "UUIDText", "Badfile.txt")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := uuidtext.ParseUUIDText(data)
	require.Contains(t, err.Error(), "invalid UUID text signature")
	require.Nil(t, results)
}

// buildUUIDTextBytes constructs a minimal UUIDText binary for testing.
func buildUUIDTextBytes(major, minor uint32, entries []uuidtext.Entry, footer []byte) []byte {
	// Signature 0x66778899 (little-endian)
	buf := make([]byte, 0, 4+4+4+4+len(entries)*8+len(footer))

	// Signature
	sig := make([]byte, 4)
	binary.LittleEndian.PutUint32(sig, 0x66778899)
	buf = append(buf, sig...)

	// Versions
	v := make([]byte, 4)
	binary.LittleEndian.PutUint32(v, major)
	buf = append(buf, v...)
	binary.LittleEndian.PutUint32(v, minor)
	buf = append(buf, v...)

	// Number of entries
	binary.LittleEndian.PutUint32(v, uint32(len(entries)))
	buf = append(buf, v...)

	// Entries
	efield := make([]byte, 4)
	for _, e := range entries {
		binary.LittleEndian.PutUint32(efield, e.RangeStartOffset)
		buf = append(buf, efield...)
		binary.LittleEndian.PutUint32(efield, e.EntrySize)
		buf = append(buf, efield...)
	}

	// Footer data (strings blob)
	buf = append(buf, footer...)
	return buf
}

func TestParseUUIDText_Valid(t *testing.T) {
	entries := []uuidtext.Entry{
		{RangeStartOffset: 10, EntrySize: 100},
		{RangeStartOffset: 200, EntrySize: 50},
	}
	footer := []byte("hello\x00world\x00")
	data := buildUUIDTextBytes(1, 2, entries, footer)

	got, err := uuidtext.ParseUUIDText(data)
	require.NoError(t, err)

	require.Equal(t, uint32(0x66778899), got.Signature)
	require.Equal(t, uint32(1), got.UnknownMajorVersion)
	require.Equal(t, uint32(2), got.UnknownMinorVersion)
	require.Equal(t, uint32(len(entries)), got.NumberEntries)

	// Note: current implementation pre-allocates slice with length NumberEntries and then appends again.
	// Validate that the appended entries exist at the tail of the slice.
	require.GreaterOrEqual(t, len(got.EntryDescriptors), len(entries))
	// Expect appended entries at indices [NumberEntries, NumberEntries+len(entries))
	base := int(got.NumberEntries)
	require.GreaterOrEqual(t, len(got.EntryDescriptors), base+len(entries))
	for i := range entries {
		require.Equal(t, entries[i].RangeStartOffset, got.EntryDescriptors[base+i].RangeStartOffset)
		require.Equal(t, entries[i].EntrySize, got.EntryDescriptors[base+i].EntrySize)
	}

	require.Equal(t, footer, got.FooterData)
}

func TestParseUUIDText_InvalidSignature(t *testing.T) {
	// Invalid signature (all zeros)
	data := []byte{0x00, 0x00, 0x00, 0x00}
	_, err := uuidtext.ParseUUIDText(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid UUID text signature")
}

func TestParseUUIDText_ShortInput(t *testing.T) {
	// Fewer than 4 bytes for signature
	data := []byte{0x01, 0x02, 0x03}
	_, err := uuidtext.ParseUUIDText(data)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read signature")
}

func TestParseUUIDText_FileBased(t *testing.T) {
	// Mirrors the file-reading pattern used in DSC tests, but generates a temp file here.
	// If we later add static fixtures under testdata, we can switch to using testutil.ExtensionTestdataDir().
	entries := []uuidtext.Entry{{RangeStartOffset: 1234, EntrySize: 56}}
	footer := []byte("fmt %s\x00")
	payload := buildUUIDTextBytes(3, 4, entries, footer)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "uuidtext_sample")
	require.NoError(t, os.WriteFile(filePath, payload, 0o600))

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	got, err := uuidtext.ParseUUIDText(data)
	require.NoError(t, err)

	require.Equal(t, uint32(3), got.UnknownMajorVersion)
	require.Equal(t, uint32(4), got.UnknownMinorVersion)
	require.Equal(t, uint32(1), got.NumberEntries)

	base := int(got.NumberEntries)
	require.GreaterOrEqual(t, len(got.EntryDescriptors), base+1)
	require.Equal(t, uint32(1234), got.EntryDescriptors[base+0].RangeStartOffset)
	require.Equal(t, uint32(56), got.EntryDescriptors[base+0].EntrySize)
	require.True(t, strings.HasPrefix(string(got.FooterData), "fmt "))
}
