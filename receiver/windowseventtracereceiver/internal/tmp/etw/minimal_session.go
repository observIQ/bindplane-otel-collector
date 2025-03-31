//go:build windows

package etw

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
)

// Minimal Windows API constants
const (
	WNODE_FLAG_ALL_DATA              = 0x00000001
	EVENT_TRACE_REAL_TIME_MODE_MIN   = 0x00000100
	EVENT_TRACE_FILE_MODE_SEQUENTIAL = 0x00000001
	EVENT_TRACE_CONTROL_STOP         = 1
	PROCESS_TRACE_MODE_REAL_TIME     = 0x00000002
)

// MinimalSession implements the absolute minimum needed to start an ETW session
type MinimalSession struct {
	handle syscall.Handle
	buffer []byte
	name   string
}

// NewMinimalSession creates a new minimal session
func NewMinimalSession(sessionName string) *MinimalSession {
	// Use a simple name without timestamps
	return &MinimalSession{
		name: sessionName,
	}
}

// Start starts the ETW session with hardcoded values for simplicity
func (s *MinimalSession) Start() error {
	fmt.Printf("Debug: Starting minimal session with name '%s'\n", s.name)
	namePtr, err := syscall.UTF16PtrFromString(s.name)
	if err != nil {
		return fmt.Errorf("failed to convert session name to UTF-16: %w", err)
	}

	aa32 := windows.NewLazySystemDLL("advapi32.dll")
	startTraceW := aa32.NewProc("StartTraceW")
	controlTraceW := aa32.NewProc("ControlTraceW")

	// Initialize properties using the helper function
	props, totalSize := NewMinimalSessionProperties(s.name)
	s.buffer = make([]byte, totalSize)
	copy(s.buffer, (*[1 << 16]byte)(unsafe.Pointer(props))[:totalSize:totalSize])
	props = (*advapi32.EventTraceProperties)(unsafe.Pointer(&s.buffer[0]))

	// Copy the session name to the buffer
	sessionNamePtr, err := syscall.UTF16FromString(s.name)
	if err != nil {
		return fmt.Errorf("failed to convert session name to UTF-16: %w", err)
	}

	// Copy each UTF-16 character (2 bytes) to the buffer
	for i, ch := range sessionNamePtr {
		offset := props.LoggerNameOffset + uint32(i*2)
		if int(offset+1) >= len(s.buffer) {
			break
		}
		s.buffer[offset] = byte(ch)
		s.buffer[offset+1] = byte(ch >> 8)
	}

	// TODO(schmikei): migrate over to use advapi32.StartTrace
	// err = advapi32.StartTrace(
	// 	s.handle,
	// 	namePtr,
	// 	(*advapi32.EventTraceProperties)(unsafe.Pointer(&s.buffer[0])),
	// )
	// if err != nil {
	// 	return fmt.Errorf("failed to start trace: %w", err)
	// }

	// Make the call with extreme care
	r1, _, _ := startTraceW.Call(
		uintptr(unsafe.Pointer(&s.handle)),
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(unsafe.Pointer(props)),
	)

	if r1 != 0 {
		errCode := uint32(r1)

		// Handle the case where the trace already exists
		if errCode == 183 { // ERROR_ALREADY_EXISTS
			fmt.Printf("Debug: Session already exists, attempting to stop and restart\n")

			// Create a copy of properties for ControlTrace
			propCopy := make([]byte, len(s.buffer))
			copy(propCopy, s.buffer)
			propsCopy := (*advapi32.EventTraceProperties)(unsafe.Pointer(&propCopy[0]))

			// Stop the existing session
			r1, _, _ = controlTraceW.Call(
				0, // Use 0 for handle since we're stopping by name
				uintptr(unsafe.Pointer(namePtr)),
				uintptr(unsafe.Pointer(propsCopy)),
				uintptr(EVENT_TRACE_CONTROL_STOP),
			)

			// Try to start again
			r1, _, _ = startTraceW.Call(
				uintptr(unsafe.Pointer(&s.handle)),
				uintptr(unsafe.Pointer(namePtr)),
				uintptr(unsafe.Pointer(props)),
			)
		}
	}
	return nil
}

// EnableDummyProvider enables a dummy provider with trace level verbose
func (s *MinimalSession) EnableDummyProvider() error {
	// Simple no-op implementation for now
	fmt.Printf("Debug: Skipping provider enablement in minimal session\n")
	return nil
}

// Stop stops the ETW session
func (s *MinimalSession) Stop(ctx context.Context) error {
	if s.handle == 0 {
		return nil
	}

	// Load advapi32.dll
	advapi32 := windows.NewLazySystemDLL("advapi32.dll")
	controlTraceW := advapi32.NewProc("ControlTraceW")

	// We don't need properties for stopping by handle
	r1, _, _ := controlTraceW.Call(
		uintptr(s.handle),
		0, // Use 0 for instance name since we're stopping by handle
		0, // Use 0 for properties since we're stopping by handle
		uintptr(EVENT_TRACE_CONTROL_STOP),
	)

	if r1 != 0 {
		fmt.Printf("Debug: Failed to stop trace (code=%d)\n", r1)
	}

	s.handle = 0
	return nil
}

func NewMinimalSessionProperties(logSessionName string) (*advapi32.EventTraceProperties, uint32) {
	sessionNameSize := (len(logSessionName) + 1) * 2
	propSize := unsafe.Sizeof(advapi32.EventTraceProperties{})
	totalSize := propSize + uintptr(sessionNameSize)
	s := make([]byte, totalSize)
	props := (*advapi32.EventTraceProperties)(unsafe.Pointer(&s[0]))

	// Set properties exactly like the original implementation
	props.Wnode.BufferSize = uint32(totalSize)
	props.Wnode.Guid = advapi32.GUID{}
	props.Wnode.ClientContext = 1 // QPC
	props.Wnode.Flags = WNODE_FLAG_ALL_DATA
	props.LogFileMode = EVENT_TRACE_REAL_TIME_MODE_MIN
	props.LogFileNameOffset = 0
	props.BufferSize = 64
	props.MinimumBuffers = 2
	props.MaximumBuffers = 4
	props.EnableFlags = 0
	props.FlushTimer = 1
	props.AgeLimit = 15
	props.MaximumFileSize = 0

	return props, uint32(totalSize)
}

// Providers returns the list of providers
func (s *MinimalSession) Providers() []Provider {
	return nil
}

// TraceName returns the name of the trace
func (s *MinimalSession) TraceName() string {
	return s.name
}
