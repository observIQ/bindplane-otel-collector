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
	"strings"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/sys/windows"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/tdh"
)

type Session struct {
	name        string
	providerMap map[string]*Provider
	logger      *zap.Logger
	bufferSize  int

	properties *advapi32.EventTraceProperties
	handle     syscall.Handle

	// Add a field for the minimal session implementation
	controller *SessionController
}

func NewRealTimeSession(name string, logger *zap.Logger, bufferSize int) *Session {
	return &Session{
		name:        name,
		handle:      0,
		providerMap: make(map[string]*Provider),
		logger:      logger,
		bufferSize:  bufferSize,
	}
}

func (s *Session) Start(ctx context.Context) error {
	if s.handle != 0 {
		return fmt.Errorf("session already started with handle %d", s.handle)
	}

	c, err := createSessionController(s.name, s.logger, s.bufferSize)
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}

	s.controller = c
	s.handle = c.handle

	err = s.initializeProviderMap()
	if err != nil {
		return fmt.Errorf("failed to initialize provider map: %w", err)
	}
	return nil
}

func (s *Session) EnableProvider(nameOrGuid string, traceLevel advapi32.TraceLevel, matchAnyKeyword uint64, matchAllKeyword uint64) error {
	// Make sure the session is started
	if s.handle == 0 {
		return fmt.Errorf("session must be started before enabling providers")
	}

	provider, ok := s.providerMap[nameOrGuid]
	if !ok {
		providerNames := make([]string, 0, len(s.providerMap))
		for name := range s.providerMap {
			providerNames = append(providerNames, name)
		}
		return fmt.Errorf("provider %s not found, available providers: %v", nameOrGuid, strings.Join(providerNames, ", "))
	}

	guid, err := windows.GUIDFromString(provider.GUID)
	if err != nil {
		return fmt.Errorf("failed to parse provider GUID: %w", err)
	}
	if err := s.controller.enableProvider(s.handle, &guid, provider, traceLevel, matchAnyKeyword, matchAllKeyword); err != nil {
		return fmt.Errorf("failed to enable provider %s: %w", provider.Name, err)
	}
	return nil

}

var providerMapOnce sync.Once
var providers map[string]*Provider
var providerErr error

func (s *Session) initializeProviderMap() error {
	providerMapOnce.Do(func() {
		providers = make(map[string]*Provider)

		// Start with a small buffer and let Windows tell us the required size we need for the buffer for providers
		bufferSize := uint32(1)
		buffer := make([]byte, bufferSize)

		enumInfo := (*tdh.ProviderEnumerationInfo)(unsafe.Pointer(&buffer[0]))
		err := tdh.EnumerateProviders(enumInfo, &bufferSize)
		if err != nil {
			if err != windows.ERROR_INSUFFICIENT_BUFFER {
				providerErr = fmt.Errorf("failed to get required buffer size: %w", err)
				return
			}
		}

		// Create buffer with the required size
		buffer = make([]byte, bufferSize)

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

		providerInfoSize := unsafe.Sizeof(tdh.TraceProviderInfo{})
		providerInfoOffset := unsafe.Sizeof(uint32(0)) * 2

		for i := uint32(0); i < numProviders; i++ {
			offset := providerInfoOffset + (uintptr(i) * providerInfoSize)
			if offset+providerInfoSize > uintptr(len(buffer)) {
				s.logger.Warn("Buffer overflow prevented, stopping at provider", zap.Uint32("provider", i))
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

func utf16StringAtOffset(pstruct uintptr, offset uintptr) string {
	stringBuffer := make([]uint16, 0, 64)

	windowChar := (*uint16)(unsafe.Pointer(pstruct + offset))

	for i := uintptr(2); *windowChar != 0; i += 2 {
		stringBuffer = append(stringBuffer, *windowChar)
		windowChar = (*uint16)(unsafe.Pointer(pstruct + offset + i))
	}

	return syscall.UTF16ToString(stringBuffer)
}

// createSessionController creates a very basic ETW session
func createSessionController(sessionName string, logger *zap.Logger, bufferSize int) (*SessionController, error) {
	session := newSessionController(sessionName, bufferSize, logger)

	// Start the session
	if err := session.Start(); err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	return session, nil
}
