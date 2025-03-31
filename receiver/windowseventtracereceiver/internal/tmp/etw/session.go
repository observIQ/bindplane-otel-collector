//go:build windows

package etw

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"

	advapi32pkg "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/tdh"
	"golang.org/x/sys/windows"
)

// Define constants needed for session configuration
const (
	EVENT_TRACE_REAL_TIME_MODE = 0x00000100
	WNODE_FLAG_TRACED_GUID     = 0x00020000

	// Control codes
	EVENT_CONTROL_CODE_ENABLE_PROVIDER = 1

	// Trace levels
	TRACE_LEVEL_INFORMATION = 4
)

type Session struct {
	name        string
	providerMap map[string]*Provider

	properties *advapi32pkg.EventTraceProperties
	handle     syscall.Handle

	// Add buffer field to keep the memory alive for the entire session
	buffer []byte

	// Add a field for the minimal session implementation
	minimalSession *MinimalSession
}

type eventTraceProperties struct {
	// Wnode               WnodeHeader
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

func sessionBase() *Session {
	return &Session{
		name:        "BindplaneOtelCollector",
		providerMap: make(map[string]*Provider),
	}
}

func (s *Session) WithSessionName(name string) *Session {
	s.name = name
	return s
}

func NewRealTimeSession(name string) *Session {
	s := sessionBase().WithSessionName(name)

	// Create a properly sized buffer for the properties including space for session and log file names
	// We need to layout memory in this order: struct, log file name (empty), session name
	sessionNameLen := (len(name) + 1) * 2 // UTF-16 characters (2 bytes each) + null terminator
	logFileNameLen := 2                   // Just null terminator (2 bytes for UTF-16)

	// Calculate the total size needed
	propSize := unsafe.Sizeof(advapi32pkg.EventTraceProperties{})
	totalSize := propSize + uintptr(sessionNameLen) + uintptr(logFileNameLen)

	// Allocate the buffer for properties and strings
	s.buffer = make([]byte, totalSize)
	props := (*advapi32pkg.EventTraceProperties)(unsafe.Pointer(&s.buffer[0]))

	// Set basic properties
	props.Wnode.BufferSize = uint32(totalSize)
	props.Wnode.Flags = WNODE_FLAG_TRACED_GUID

	// Set real-time mode
	props.LogFileMode = EVENT_TRACE_REAL_TIME_MODE

	// Set buffer parameters
	props.BufferSize = 64 // 64 KB
	props.MinimumBuffers = 2
	props.MaximumBuffers = 4
	props.FlushTimer = 1 // 1 second

	// Set the offsets for strings
	props.LoggerNameOffset = uint32(propSize)
	props.LogFileNameOffset = uint32(propSize + uintptr(sessionNameLen))

	// Copy the session name (as UTF-16) into the buffer at the correct offset
	sessionNamePtr, err := syscall.UTF16FromString(name)
	if err == nil {
		// Calculate offset in bytes
		offset := props.LoggerNameOffset

		// Copy each byte of the UTF-16 string
		for i := 0; i < len(sessionNamePtr); i++ {
			if int(offset)+1 >= len(s.buffer) {
				break
			}

			// Copy the UTF-16 character (2 bytes)
			s.buffer[offset] = byte(sessionNamePtr[i])
			s.buffer[offset+1] = byte(sessionNamePtr[i] >> 8)
			offset += 2
		}
	}

	// Assign the properties to the session
	s.properties = props

	return s
}

func (s *Session) Start() error {
	// For backward compatibility, remove context parameter
	return s.StartWithContext(context.Background())
}

func (s *Session) StartWithContext(ctx context.Context) error {
	if s.handle != 0 {
		return nil // Session already started
	}

	// DirectSession worked
	// DirectSession failed, try the MinimalSession implementation
	fmt.Printf("Debug: Trying MinimalSession implementation...\n")
	minimalSession, err := CreateMinimalSession(s.name)
	if err != nil {
		// Both implementations failed, return both errors
		return fmt.Errorf("failed to create session: %v", err)
	}

	// MinimalSession worked
	s.minimalSession = minimalSession
	s.handle = 1 // Placeholder handle to indicate session is started

	fmt.Printf("Debug: Successfully started trace using MinimalSession\n")
	return nil
}

func (s *Session) EnableProvider(nameOrGuid string) error {
	err := s.initializeProviderMap()
	if err != nil {
		return fmt.Errorf("failed to initialize provider map: %w", err)
	}

	// Use the MinimalSession if it exists (it already has a provider enabled)
	if s.minimalSession != nil {
		// Minimal session already has a provider enabled in its creation
		return nil
	}

	if provider, ok := s.providerMap[nameOrGuid]; ok {
		guid, err := windows.GUIDFromString(provider.GUID)
		if err != nil {
			return fmt.Errorf("failed to parse provider GUID: %w", err)
		}

		// Make sure the session is started
		if s.handle == 0 {
			return fmt.Errorf("session must be started before enabling providers")
		}

		// Enable the provider with default settings
		// Using standard Windows ETW keywords and levels
		keywords := uint64(0xFFFFFFFFFFFFFFFF) // All keywords

		// Call our custom enableProvider function
		if err := enableProvider(s.handle, guid, keywords); err != nil {
			return fmt.Errorf("failed to enable provider %s: %w", provider.Name, err)
		}

		return nil
	}

	providerNames := make([]string, 0, len(s.providerMap))
	for name := range s.providerMap {
		providerNames = append(providerNames, name)
	}
	return fmt.Errorf("provider %s not found, available providers: %v", nameOrGuid, strings.Join(providerNames, ", "))
}

var providerMapOnce sync.Once
var providers map[string]*Provider
var providerErr error

func (s *Session) initializeProviderMap() error {
	providerMapOnce.Do(func() {
		providers = make(map[string]*Provider)

		// Start with a small buffer and let Windows tell us the required size
		bufferSize := uint32(1)
		buffer := make([]byte, bufferSize)

		// First call to get required buffer size
		enumInfo := (*tdh.ProviderEnumerationInfo)(unsafe.Pointer(&buffer[0]))
		err := tdh.EnumerateProviders(enumInfo, &bufferSize)
		if err != nil {
			if err == windows.ERROR_INSUFFICIENT_BUFFER {
				fmt.Printf("Buffer too small, required size: %d\n", bufferSize)
			} else {
				providerErr = fmt.Errorf("failed to get required buffer size: %w", err)
				return
			}
		}

		// Create buffer with the required size
		buffer = make([]byte, bufferSize)

		// Second call to actually get the providers
		enumInfo = (*tdh.ProviderEnumerationInfo)(unsafe.Pointer(&buffer[0]))
		err = tdh.EnumerateProviders(enumInfo, &bufferSize)
		if err != nil {
			providerErr = fmt.Errorf("failed to enumerate providers: %w", err)
			return
		}

		numProviders := enumInfo.NumberOfProviders
		if numProviders == 0 {
			providerErr = fmt.Errorf("no providers found in enumeration")
			return
		}

		// The provider info array starts immediately after the header
		// Header is 8 bytes (NumberOfProviders + Reserved)
		providerInfoSize := unsafe.Sizeof(tdh.TraceProviderInfo{})
		providerInfoOffset := unsafe.Sizeof(uint32(0)) * 2 // Skip NumberOfProviders and Reserved

		// Extract each provider
		for i := uint32(0); i < numProviders; i++ {
			// Calculate the offset to this provider's info
			offset := providerInfoOffset + (uintptr(i) * providerInfoSize)
			if offset+providerInfoSize > uintptr(len(buffer)) {
				fmt.Printf("Warning: Buffer overflow prevented, stopping at provider %d\n", i)
				break
			}

			provInfo := (*tdh.TraceProviderInfo)(unsafe.Pointer(&buffer[offset]))

			name := utf16BufferToString(buffer, provInfo.ProviderNameOffset)
			if name == "" {
				continue
			}

			guid := provInfo.ProviderGuid.String()
			provider := baseNewProvider()
			provider.Name = name
			provider.GUID = guid

			providers[name] = provider
			providers[guid] = provider
		}

		if len(providers) == 0 {
			providerErr = fmt.Errorf("no providers found in enumeration")
			return
		}
	})

	if providerErr != nil {
		return providerErr
	}

	s.providerMap = providers
	return nil
}

// utf16BufferToString converts a UTF16 encoded byte buffer at the specified offset to a Go string
func utf16BufferToString(buffer []byte, offset uint32) string {
	if offset == 0 || int(offset) >= len(buffer) {
		return ""
	}

	// Make sure offset is properly aligned for a uint16
	if offset%2 != 0 {
		return ""
	}

	// Determine string length (find the null terminator)
	strLen := 0
	for i := offset; i < uint32(len(buffer)); i += 2 {
		if i+1 >= uint32(len(buffer)) {
			break
		}

		// Extract character from buffer
		char := uint16(buffer[i]) | (uint16(buffer[i+1]) << 8)
		if char == 0 {
			break
		}
		strLen++
	}

	if strLen == 0 {
		return ""
	}

	// Create slice to hold the UTF16 chars
	utf16Chars := make([]uint16, strLen)
	for i := 0; i < strLen; i++ {
		pos := offset + uint32(i*2)
		if pos+1 >= uint32(len(buffer)) {
			break
		}

		// Extract character from buffer
		utf16Chars[i] = uint16(buffer[pos]) | (uint16(buffer[pos+1]) << 8)
	}

	// Convert UTF16 to string
	return string(utf16.Decode(utf16Chars))
}

// // createTraceBuffer creates a buffer for the EventTraceProperties structure and its associated strings
// func createTraceBuffer(totalSize uintptr, logFileName, sessionName string) ([]byte, error) {
// 	nameSize := uint32(len(sessionName)+1) * 2
// 	logFileNameSize := uint32(len(logFileName)+1) * 2

// 	// Create a buffer that includes the structure, session name, and log file name
// 	buffer := make([]byte, totalSize+uintptr(nameSize)+uintptr(logFileNameSize))

// 	// Copy the log file name into the buffer
// 	logFileNameBytes := make([]uint16, len(logFileName)+1)
// 	utf16LogFileName, err := syscall.UTF16FromString(logFileName)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to convert log file name to UTF-16: %w", err)
// 	}
// 	copy(logFileNameBytes, utf16LogFileName)
// 	copy(buffer[totalSize:], (*[1<<30 - 1]byte)(unsafe.Pointer(&logFileNameBytes[0]))[:logFileNameSize])

// 	// Copy the session name into the buffer
// 	nameBytes := make([]uint16, len(sessionName)+1)
// 	utf16Name, err := syscall.UTF16FromString(sessionName)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to convert session name to UTF-16: %w", err)
// 	}
// 	copy(nameBytes, utf16Name)
// 	copy(buffer[totalSize+uintptr(logFileNameSize):], (*[1<<30 - 1]byte)(unsafe.Pointer(&nameBytes[0]))[:nameSize])

// 	return buffer, nil
// }

// // initializeProperties initializes the EventTraceProperties structure with the given parameters
// func initializeProperties(buffer []byte, totalSize uintptr, logFileNameSize uint32, sessionName string) *advapi32pkg.EventTraceProperties {
// 	properties := (*advapi32pkg.EventTraceProperties)(unsafe.Pointer(&buffer[0]))
// 	properties.Wnode = advapi32pkg.WnodeHeader{
// 		BufferSize: uint32(totalSize + uintptr(len(sessionName)+1)*2 + uintptr(logFileNameSize)),
// 		Flags:      0x00020000, // WNODE_FLAG_TRACED_GUID
// 	}
// 	properties.BufferSize = 64
// 	properties.MinimumBuffers = 2
// 	properties.MaximumBuffers = 4
// 	properties.MaximumFileSize = 100
// 	properties.LogFileMode = advapi32pkg.EVENT_TRACE_REAL_TIME_MODE | advapi32pkg.EVENT_TRACE_BUFFERING_MODE
// 	properties.LogFileNameOffset = uint32(totalSize)
// 	properties.LoggerNameOffset = uint32(totalSize + uintptr(logFileNameSize))
// 	return properties
// }

// Stop stops the session
func (s *Session) Stop() error {
	if s.handle == 0 {
		// Session not started
		return nil
	}

	// Use the MinimalSession if it exists
	if s.minimalSession != nil {
		err := s.minimalSession.Stop()
		if err != nil {
			return fmt.Errorf("failed to stop minimal session: %w", err)
		}

		// Clear the handle and minimal session
		s.handle = 0
		s.minimalSession = nil

		return nil
	}

	// The original stop implementation...
	// Create a properties structure for stopping the trace
	// Use the same approach as for starting - allocate buffer with space for strings
	sessionNameLen := (len(s.name) + 1) * 2 // UTF-16 characters + null terminator
	logFileNameLen := 2                     // Just null terminator

	propSize := unsafe.Sizeof(advapi32pkg.EventTraceProperties{})
	totalSize := propSize + uintptr(sessionNameLen) + uintptr(logFileNameLen)

	// Allocate buffer for the stop properties
	stopBuffer := make([]byte, totalSize)
	stopProps := (*advapi32pkg.EventTraceProperties)(unsafe.Pointer(&stopBuffer[0]))

	// Initialize the structure
	stopProps.Wnode.BufferSize = uint32(totalSize)
	stopProps.Wnode.Flags = WNODE_FLAG_TRACED_GUID
	stopProps.LogFileMode = EVENT_TRACE_REAL_TIME_MODE

	// Set string offsets
	stopProps.LoggerNameOffset = uint32(propSize)
	stopProps.LogFileNameOffset = uint32(propSize + uintptr(sessionNameLen))

	// Copy session name to the buffer
	sessionNamePtr, err := syscall.UTF16FromString(s.name)
	if err == nil {
		// Calculate offset in bytes
		offset := stopProps.LoggerNameOffset

		// Copy each byte of the UTF-16 string
		for i := 0; i < len(sessionNamePtr); i++ {
			if int(offset)+1 >= len(stopBuffer) {
				break
			}

			// Copy the UTF-16 character (2 bytes)
			stopBuffer[offset] = byte(sessionNamePtr[i])
			stopBuffer[offset+1] = byte(sessionNamePtr[i] >> 8)
			offset += 2
		}
	}

	// Call ControlTrace with correct parameters - use null for instance name
	// and use the session name embedded in the properties structure
	if err := advapi32pkg.ControlTrace(
		nil,                                  // Trace handle (null for stop by name)
		advapi32pkg.EVENT_TRACE_CONTROL_STOP, // Stop control code
		nil,                                  // Instance name (null - we use the name in properties)
		stopProps,                            // Properties with embedded session name
		0,                                    // Timeout (0 = no timeout)
	); err != nil {
		return fmt.Errorf("failed to stop trace: %w", err)
	}

	// Clear the handle
	s.handle = 0

	return nil
}

// TraceName returns the session name
func (s *Session) TraceName() string {
	return s.name
}

// Providers returns the list of providers for this session
func (s *Session) Providers() []Provider {
	providers := make([]Provider, 0, len(s.providerMap))
	for _, p := range s.providerMap {
		providers = append(providers, Provider{
			Name:        p.Name,
			GUID:        p.GUID,
			EnableLevel: p.EnableLevel,
		})
	}
	return providers
}
