//go:build windows

package etw

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw"
	"golang.org/x/sys/windows"
)

// Minimal Windows API constants
const (
	WNODE_FLAG_ALL_DATA              = 0x00000001
	EVENT_TRACE_REAL_TIME_MODE_MIN   = 0x00000100
	EVENT_TRACE_FILE_MODE_SEQUENTIAL = 0x00000001
	EVENT_TRACE_CONTROL_STOP         = 1
)

type MinimalProperties struct {
	Wnode               WnodeHeader
	BufferSize          uint32
	MinimumBuffers      uint32
	MaximumBuffers      uint32
	MaximumFileSize     uint32
	LogFileMode         uint32
	FlushTimer          uint32
	EnableFlags         uint32
	AgeLimit            int32
	NumberOfBuffers     uint32
	FreeBuffers         uint32
	EventsLost          uint32
	BuffersWritten      uint32
	LogBuffersLost      uint32
	RealTimeBuffersLost uint32
	LoggerThreadId      syscall.Handle
	LogFileNameOffset   uint32
	LoggerNameOffset    uint32
}

type WnodeHeader struct {
	BufferSize    uint32
	ProviderId    uint32
	Union1        uint64
	Union2        int64
	Guid          etw.GUID
	ClientContext uint32
	Flags         uint32
}

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

	// Try to verify if the process is running with admin privileges
	isAdmin := checkAdminPrivileges()
	fmt.Printf("Debug: Running as admin (approximate check): %v\n", isAdmin)
	if !isAdmin {
		fmt.Printf("Debug: WARNING - ETW typically requires admin privileges\n")
	}

	// Load advapi32.dll
	advapi32 := windows.NewLazySystemDLL("advapi32.dll")
	startTraceW := advapi32.NewProc("StartTraceW")
	controlTraceW := advapi32.NewProc("ControlTraceW")

	// Report on temp directory access
	tempDir := os.TempDir()
	fmt.Printf("Debug: Using temp directory: %s\n", tempDir)

	if dirInfo, err := os.Stat(tempDir); err == nil {
		fmt.Printf("Debug: Temp directory permissions: %v\n", dirInfo.Mode())
	}

	// Try to see if current user has admin access to important directories
	systemPaths := []string{
		"C:\\Windows\\System32",
		"C:\\Windows",
		"C:\\Program Files",
		os.TempDir(),
	}

	fmt.Printf("Debug: Checking write access to system directories:\n")
	for _, path := range systemPaths {
		testFile := filepath.Join(path, fmt.Sprintf(".test_%d", time.Now().UnixNano()))
		f, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY, 0666)
		if err == nil {
			f.Close()
			os.Remove(testFile)
			fmt.Printf("Debug: - %s: WRITE ACCESS OK\n", path)
		} else {
			fmt.Printf("Debug: - %s: NO WRITE ACCESS (%v)\n", path, err)
		}
	}

	// Convert session name to UTF16 for the API call
	namePtr, err := syscall.UTF16PtrFromString(s.name)
	if err != nil {
		return fmt.Errorf("failed to convert session name to UTF-16: %w", err)
	}

	// Get the full path to the current executable
	exePath, err := os.Executable()
	if err == nil {
		fmt.Printf("Debug: Collector running from: %s\n", exePath)
	}

	// Initialize properties using the helper function
	props, totalSize := NewMinimalSessionProperties(s.name)
	s.buffer = make([]byte, totalSize)
	copy(s.buffer, (*[1 << 30]byte)(unsafe.Pointer(props))[:totalSize:totalSize])
	props = (*MinimalProperties)(unsafe.Pointer(&s.buffer[0]))

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

	// Show detailed debug information
	fmt.Printf("Debug: Properties buffer size=%d bytes\n", totalSize)
	fmt.Printf("Debug: Properties struct size=%d bytes\n", unsafe.Sizeof(MinimalProperties{}))
	fmt.Printf("Debug: LogFileNameOffset=%d, LoggerNameOffset=%d\n", props.LogFileNameOffset, props.LoggerNameOffset)
	fmt.Printf("Debug: SessionName length=%d chars, size=%d bytes\n", len(s.name), (len(s.name)+1)*2)
	fmt.Printf("Debug: WnodeHeader size=%d bytes\n", unsafe.Sizeof(WnodeHeader{}))
	fmt.Printf("Debug: GUID size=%d bytes\n", unsafe.Sizeof(etw.GUID{}))

	// Make the call with extreme care
	r1, _, lastErr := startTraceW.Call(
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
			propsCopy := (*MinimalProperties)(unsafe.Pointer(&propCopy[0]))

			// Stop the existing session
			r1, _, _ = controlTraceW.Call(
				0, // Use 0 for handle since we're stopping by name
				uintptr(unsafe.Pointer(namePtr)),
				uintptr(unsafe.Pointer(propsCopy)),
				uintptr(EVENT_TRACE_CONTROL_STOP),
			)

			// Try to start again
			r1, _, lastErr = startTraceW.Call(
				uintptr(unsafe.Pointer(&s.handle)),
				uintptr(unsafe.Pointer(namePtr)),
				uintptr(unsafe.Pointer(props)),
			)
		}

		if r1 != 0 {
			errCode = uint32(r1)
			// Provide detailed error information
			switch errCode {
			case 5:
				return fmt.Errorf("access denied (code=%d) - MUST run as Administrator", errCode)
			case 183:
				return fmt.Errorf("trace session already exists (code=%d) - try a different session name", errCode)
			case 161:
				return fmt.Errorf("invalid path (code=%d) - this is often a permissions error; run as Administrator. Error details: %v",
					errCode, lastErr)
			default:
				return fmt.Errorf("failed to start trace (code=%d): %v", errCode, lastErr)
			}
		}
	}

	fmt.Printf("Debug: Successfully started trace with handle=%d\n", s.handle)
	return nil
}

// Check if running with admin privileges (simplified approach)
func checkAdminPrivileges() bool {
	// Simple approximate check using file access
	// Try to open a file in System32 for write (typically requires admin)
	systemDir := os.Getenv("SystemRoot") + "\\System32"
	testFile := filepath.Join(systemDir, ".etw_admin_check_temp")

	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		f.Close()
		os.Remove(testFile)
		return true
	}

	return false
}

// EnableDummyProvider enables a dummy provider with trace level verbose
func (s *MinimalSession) EnableDummyProvider() error {
	// Simple no-op implementation for now
	fmt.Printf("Debug: Skipping provider enablement in minimal session\n")
	return nil
}

// Stop stops the ETW session
func (s *MinimalSession) Stop() error {
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

func NewMinimalSessionProperties(logSessionName string) (*MinimalProperties, uint32) {
	sessionNameSize := (len(logSessionName) + 1) * 2
	propSize := unsafe.Sizeof(MinimalProperties{})
	totalSize := propSize + uintptr(sessionNameSize)
	s := make([]byte, totalSize)
	props := (*MinimalProperties)(unsafe.Pointer(&s[0]))

	// Set properties exactly like the original implementation
	props.Wnode.BufferSize = uint32(totalSize)
	props.Wnode.Guid = etw.GUID{}
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
	props.LoggerNameOffset = uint32(propSize)

	return props, uint32(totalSize)
}

// Providers returns the list of providers
func (s *MinimalSession) Providers() []etw.Provider {
	return nil
}

// TraceName returns the name of the trace
func (s *MinimalSession) TraceName() string {
	return s.name
}
