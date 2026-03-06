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

//go:build windows

package etw

import (
	"encoding/binary"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
)

// winAlloc allocates size bytes of memory outside the Go heap using VirtualAlloc.
// DataPtr fields in EventHeaderExtendedDataItem must not point into Go heap memory:
// the unsafe uintptr→unsafe.Pointer conversion in collectExtendedData would be
// flagged by the checkptr instrumentation that runs during race-enabled test builds.
// Windows-allocated memory is outside the Go heap and is not checked by checkptr.
func winAlloc(t *testing.T, size int) (uintptr, func()) {
	t.Helper()
	if size < 1 {
		size = 1
	}
	p, err := windows.VirtualAlloc(0, uintptr(size), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	require.NoError(t, err)
	return p, func() { windows.VirtualFree(p, 0, windows.MEM_RELEASE) } //nolint:errcheck
}

// makeExtRecord builds an EventRecord with a single extended data item.
// data is copied into Windows-allocated memory so DataPtr is outside the Go heap.
func makeExtRecord(t *testing.T, extType uint16, data []byte) (*advapi32.EventRecord, func()) {
	t.Helper()
	dataPtr, free := winAlloc(t, len(data))
	if len(data) > 0 {
		dst := unsafe.Slice((*byte)(unsafe.Pointer(dataPtr)), len(data))
		copy(dst, data)
	}
	item := &advapi32.EventHeaderExtendedDataItem{
		ExtType:  extType,
		DataSize: uint16(len(data)),
		DataPtr:  dataPtr,
	}
	r := &advapi32.EventRecord{ExtendedDataCount: 1, ExtendedData: item}
	return r, free
}

// guidBytes serializes g into a 16-byte slice matching the in-memory layout of
// windows.GUID so the bytes can be copied into Windows-allocated test buffers.
func guidBytes(g windows.GUID) []byte {
	b := make([]byte, 16)
	binary.LittleEndian.PutUint32(b[0:], g.Data1)
	binary.LittleEndian.PutUint16(b[4:], g.Data2)
	binary.LittleEndian.PutUint16(b[6:], g.Data3)
	copy(b[8:], g.Data4[:])
	return b
}

func TestCollectExtendedData_Empty(t *testing.T) {
	assert.Nil(t, collectExtendedData(&advapi32.EventRecord{}))
}

func TestCollectExtendedData_TerminalSessionID(t *testing.T) {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, 42)
	r, cleanup := makeExtRecord(t, advapi32.EventHeaderExtTypeTerminalSessionID, data)
	defer cleanup()
	assert.Equal(t, "42", collectExtendedData(r)["terminal_session_id"])
}

func TestCollectExtendedData_TerminalSessionID_TooSmall(t *testing.T) {
	r, cleanup := makeExtRecord(t, advapi32.EventHeaderExtTypeTerminalSessionID, make([]byte, 3))
	defer cleanup()
	assert.Empty(t, collectExtendedData(r))
}

func TestCollectExtendedData_EventKey(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, 0x0000aabbccddeeff)
	r, cleanup := makeExtRecord(t, advapi32.EventHeaderExtTypeEventKey, data)
	defer cleanup()
	assert.Equal(t, "0x0000aabbccddeeff", collectExtendedData(r)["event_key"])
}

func TestCollectExtendedData_ProcessStartKey(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, 0x0000000012345678)
	r, cleanup := makeExtRecord(t, advapi32.EventHeaderExtTypeProcessStartKey, data)
	defer cleanup()
	assert.Equal(t, "0x0000000012345678", collectExtendedData(r)["process_start_key"])
}

func TestCollectExtendedData_PMCCounters(t *testing.T) {
	data := make([]byte, 16)
	binary.LittleEndian.PutUint64(data[0:], 100)
	binary.LittleEndian.PutUint64(data[8:], 200)
	r, cleanup := makeExtRecord(t, advapi32.EventHeaderExtTypePMCCounters, data)
	defer cleanup()
	assert.Equal(t, []any{"100", "200"}, collectExtendedData(r)["pmc_counters"])
}

func TestCollectExtendedData_StackTrace32(t *testing.T) {
	// struct: uint64 MatchId + uint32 Address[]
	data := make([]byte, 16) // MatchId + two 32-bit addresses
	binary.LittleEndian.PutUint64(data[0:], 0) // MatchId (ignored)
	binary.LittleEndian.PutUint32(data[8:], 0xdeadbeef)
	binary.LittleEndian.PutUint32(data[12:], 0xcafe1234)
	r, cleanup := makeExtRecord(t, advapi32.EventHeaderExtTypeStackTrace32, data)
	defer cleanup()
	assert.Equal(t, []any{"0xdeadbeef", "0xcafe1234"}, collectExtendedData(r)["stack_trace_32"])
}

func TestCollectExtendedData_StackTrace64(t *testing.T) {
	// struct: uint64 MatchId + uint64 Address[]
	data := make([]byte, 24) // MatchId + two 64-bit addresses
	binary.LittleEndian.PutUint64(data[0:], 0) // MatchId (ignored)
	binary.LittleEndian.PutUint64(data[8:], 0x00007ff812345678)
	binary.LittleEndian.PutUint64(data[16:], 0x00007ff8deadbeef)
	r, cleanup := makeExtRecord(t, advapi32.EventHeaderExtTypeStackTrace64, data)
	defer cleanup()
	assert.Equal(t, []any{"0x00007ff812345678", "0x00007ff8deadbeef"}, collectExtendedData(r)["stack_trace_64"])
}

func TestCollectExtendedData_InstanceInfo(t *testing.T) {
	// struct: uint32 InstanceId + uint32 ParentInstanceId + GUID ParentGuid
	parentGUID := windows.GUID{Data1: 0xaabbccdd, Data2: 0x1122, Data3: 0x3344}
	data := make([]byte, 24)
	binary.LittleEndian.PutUint32(data[0:], 10)
	binary.LittleEndian.PutUint32(data[4:], 20)
	copy(data[8:], guidBytes(parentGUID))
	r, cleanup := makeExtRecord(t, advapi32.EventHeaderExtTypeInstanceInfo, data)
	defer cleanup()

	result := collectExtendedData(r)
	info, ok := result["instance_info"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "10", info["instance_id"])
	assert.Equal(t, "20", info["parent_instance_id"])
	assert.Equal(t, parentGUID.String(), info["parent_guid"])
}

func TestCollectExtendedData_StackKey32(t *testing.T) {
	// struct: uint64 MatchId + uint32 StackKey + uint32 Padding
	data := make([]byte, 16)
	binary.LittleEndian.PutUint64(data[0:], 0x0000000011223344) // MatchId
	binary.LittleEndian.PutUint32(data[8:], 0xaabbccdd)         // StackKey
	r, cleanup := makeExtRecord(t, advapi32.EventHeaderExtTypeStackKey32, data)
	defer cleanup()

	result := collectExtendedData(r)
	sk, ok := result["stack_key_32"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "0x0000000011223344", sk["match_id"])
	assert.Equal(t, "0xaabbccdd", sk["key"])
}

func TestCollectExtendedData_StackKey64(t *testing.T) {
	// struct: uint64 MatchId + uint64 StackKey
	data := make([]byte, 16)
	binary.LittleEndian.PutUint64(data[0:], 0x0000000055667788) // MatchId
	binary.LittleEndian.PutUint64(data[8:], 0x000000009900aabb) // StackKey
	r, cleanup := makeExtRecord(t, advapi32.EventHeaderExtTypeStackKey64, data)
	defer cleanup()

	result := collectExtendedData(r)
	sk, ok := result["stack_key_64"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "0x0000000055667788", sk["match_id"])
	assert.Equal(t, "0x000000009900aabb", sk["key"])
}

func TestCollectExtendedData_ContainerID(t *testing.T) {
	g := windows.GUID{Data1: 0x12345678, Data2: 0xabcd, Data3: 0xef01}
	r, cleanup := makeExtRecord(t, advapi32.EventHeaderExtTypeContainerID, guidBytes(g))
	defer cleanup()
	assert.Equal(t, g.String(), collectExtendedData(r)["container_id"])
}

func TestCollectExtendedData_ControlGUID(t *testing.T) {
	g := windows.GUID{Data1: 0xdeadbeef, Data2: 0x1234, Data3: 0x5678}
	r, cleanup := makeExtRecord(t, advapi32.EventHeaderExtTypeControlGUID, guidBytes(g))
	defer cleanup()
	assert.Equal(t, g.String(), collectExtendedData(r)["control_guid"])
}

func TestCollectExtendedData_UnknownType(t *testing.T) {
	r, cleanup := makeExtRecord(t, 0x00FE, []byte{0xDE, 0xAD})
	defer cleanup()
	assert.Equal(t, "dead", collectExtendedData(r)["ext_0x00fe"])
}

func TestCollectExtendedData_UnknownType_Truncated(t *testing.T) {
	data := make([]byte, 80)
	for i := range data {
		data[i] = byte(i)
	}
	r, cleanup := makeExtRecord(t, 0x00FE, data)
	defer cleanup()
	val, ok := collectExtendedData(r)["ext_0x00fe"].(string)
	require.True(t, ok)
	assert.True(t, len(val) > 0)
	assert.Equal(t, "...", val[len(val)-3:], "truncated output should end with ...")
}
