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
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
)

func TestParseTimestamp(t *testing.T) {
	const epochDifference = 116444736000000000

	tests := []struct {
		name     string
		fileTime uint64
		expected time.Time
	}{
		{
			name:     "before unix epoch returns zero time",
			fileTime: epochDifference - 1,
			expected: time.Time{},
		},
		{
			name:     "unix epoch",
			fileTime: epochDifference,
			expected: time.Unix(0, 0).UTC(),
		},
		{
			// 2024-01-01 00:00:00 UTC = Unix 1704067200s
			// FILETIME = 1704067200 * 10,000,000 + 116444736000000000
			name:     "2024-01-01 00:00:00 UTC",
			fileTime: 133485408000000000,
			expected: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseTimestamp(tt.fileTime))
		})
	}
}

// TestGetEventProperties_StringOnly verifies that STRING_ONLY events are decoded
// from their raw UTF-16 UserData without invoking TDH, which would fail for
// events without a registered schema.
func TestGetEventProperties_StringOnly(t *testing.T) {
	msg, err := syscall.UTF16FromString("hello world")
	require.NoError(t, err)

	r := &advapi32.EventRecord{}
	r.EventHeader.Flags = advapi32.EVENT_HEADER_FLAG_STRING_ONLY
	r.UserData = uintptr(unsafe.Pointer(&msg[0]))
	r.UserDataLength = uint16(len(msg) * 2)

	props, err := GetEventProperties(r, zap.NewNop())
	runtime.KeepAlive(msg) // prevent GC of msg while r.UserData holds a raw uintptr
	require.NoError(t, err)
	require.Equal(t, "hello world", props["UserData"])
}
