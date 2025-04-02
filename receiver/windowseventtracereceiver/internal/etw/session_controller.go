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
	"context"
	"fmt"
	"syscall"
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/sys/windows"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
	windows_ "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/windows"
)

// Minimal Windows API constants
const (
	WNODE_FLAG_ALL_DATA              = 0x00000001
	EVENT_TRACE_REAL_TIME_MODE_MIN   = 0x00000100
	EVENT_TRACE_FILE_MODE_SEQUENTIAL = 0x00000001
	EVENT_TRACE_CONTROL_STOP         = 1
)

const (
	PROCESS_TRACE_MODE_REAL_TIME     = 0x00000100
	PROCESS_TRACE_MODE_RAW_TIMESTAMP = 0x00001000
	PROCESS_TRACE_MODE_EVENT_RECORD  = 0x10000000
)

// SessionController implements the absolute minimum needed to start an ETW session
type SessionController struct {
	handle     syscall.Handle
	name       string
	logger     *zap.Logger
	bufferSize int
	properties *advapi32.EventTraceProperties
}

// newSessionController creates a new minimal session
func newSessionController(sessionName string, bufferSize int, logger *zap.Logger) *SessionController {
	return &SessionController{
		name:       sessionName,
		handle:     0,
		logger:     logger,
		bufferSize: bufferSize,
	}
}

// Start starts the ETW session with hardcoded values for simplicity
func (s *SessionController) Start(_ context.Context) error {
	s.logger.Debug("Starting session", zap.String("name", s.name))

	var namePtrU16 *uint16
	var err error

	if namePtrU16, err = syscall.UTF16PtrFromString(s.name); err != nil {
		return fmt.Errorf("failed to convert session name to UTF-16: %w", err)
	}

	s.initProperties(s.name)

	r1, err := advapi32.StartTrace(
		&s.handle,
		namePtrU16,
		s.properties,
	)

	if r1 != 0 {
		errCode := uint32(r1)
		switch r1 {
		case windows.ERROR_BAD_ARGUMENTS: // ERROR_BAD_ARGUMENTS / ERROR_INVALID_PARAMETER
			return fmt.Errorf("invalid parameters for StartTraceW(%d): %v", errCode, err)
		case windows.ERROR_ALREADY_EXISTS: // ERROR_ALREADY_EXISTS
			s.logger.Debug("Session already exists, attempting to stop and restart")

			propsCopy := *s.properties
			r1, err := advapi32.ControlTrace(
				nil,
				advapi32.EVENT_TRACE_CONTROL_STOP,
				namePtrU16,
				&propsCopy,
				0,
			)
			if r1 != 0 {
				return fmt.Errorf("failed to close trace(%d): %w", r1, err)
			}

			r1, err = advapi32.StartTrace(
				&s.handle,
				namePtrU16,
				s.properties,
			)
			if r1 != 0 {
				return fmt.Errorf("failed to restart trace after stopping(%d): %w", r1, err)
			}
		default:
			return fmt.Errorf("unexpected error starting trace: error code %d: %v", errCode, err)
		}
	}
	s.logger.Debug("Started session", zap.String("name", s.name), zap.Uintptr("handle", uintptr(s.handle)))
	return nil
}

func (s *SessionController) Stop(ctx context.Context) error {
	if s.handle == 0 {
		return nil
	}

	var namePtrU16 *uint16
	var err error

	if namePtrU16, err = syscall.UTF16PtrFromString(s.name); err != nil {
		return fmt.Errorf("failed to convert session name to UTF-16: %w", err)
	}

	propsCopy := *s.properties
	r1, err := advapi32.ControlTrace(
		nil,
		advapi32.EVENT_TRACE_CONTROL_STOP,
		namePtrU16,
		&propsCopy,
		0,
	)

	if r1 != 0 {
		return fmt.Errorf("failed to stop trace(%d): %w", r1, err)
	}

	s.handle = 0
	return nil
}

func (s *SessionController) initProperties(logSessionName string) {
	props, totalSize := allocBuffer(logSessionName)

	props.Wnode.BufferSize = totalSize
	props.Wnode.Guid = windows_.GUID{}
	props.Wnode.ClientContext = 1
	props.Wnode.Flags = advapi32.WNODE_FLAG_ALL_DATA
	props.LogFileMode = advapi32.EVENT_TRACE_REAL_TIME_MODE
	props.LogFileNameOffset = 0
	props.BufferSize = uint32(s.bufferSize)
	props.FlushTimer = 1
	props.LoggerNameOffset = uint32(unsafe.Sizeof(advapi32.EventTraceProperties{}))

	s.properties = props
}

func allocBuffer(logSessionName string) (propertyBuffer *advapi32.EventTraceProperties, totalSize uint32) {
	sessionNameSize := (len(logSessionName) + 1) * 2
	propSize := unsafe.Sizeof(advapi32.EventTraceProperties{})
	size := int(propSize) + sessionNameSize

	s := make([]byte, size)
	return (*advapi32.EventTraceProperties)(unsafe.Pointer(&s[0])), uint32(size)
}

// TraceName returns the name of the trace
func (s *SessionController) TraceName() string {
	return s.name
}

// Handle returns the session handle
func (s *SessionController) Handle() syscall.Handle {
	return s.handle
}
