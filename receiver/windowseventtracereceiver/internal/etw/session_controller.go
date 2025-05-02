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
	"time"
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/sys/windows"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
	advapi32pkg "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
)

// SessionController implements the absolute minimum needed to start an ETW session
type SessionController struct {
	handle       syscall.Handle
	name         string
	logger       *zap.Logger
	bufferSize   int
	properties   *advapi32.EventTraceProperties
	isAttached   bool
	enableTrace  func(handle syscall.Handle, providerGUID *windows.GUID, controlCode uint32, level advapi32pkg.TraceLevel, matchAnyKeyword uint64, matchAllKeyword uint64, timeout uint32, enableParameters *advapi32.EnableTraceParameters) (errorCode syscall.Errno, err error)
	stopTrace    func(sessionName string) error
	controlTrace func(handle *syscall.Handle, control uint32, instanceName *uint16, properties *advapi32pkg.EventTraceProperties) (errorCode syscall.Errno, err error)
}

// newSessionController creates a new minimal session
func newSessionController(sessionName string, bufferSize int, logger *zap.Logger) *SessionController {
	return &SessionController{
		name:         sessionName,
		handle:       0,
		logger:       logger,
		bufferSize:   bufferSize,
		enableTrace:  advapi32.EnableTrace,
		stopTrace:    advapi32.StopTrace,
		controlTrace: advapi32.ControlTrace,
	}
}

// Start starts the ETW session with hardcoded values for simplicity
func (s *SessionController) Start() error {
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

	switch r1 {
	case 0:
		s.logger.Debug("Successfully started session", zap.String("name", s.name), zap.Uintptr("handle", uintptr(s.handle)))
	case windows.ERROR_INVALID_PARAMETER:
		return fmt.Errorf("invalid parameters for starting session trace(%d): %v", r1, err)
	case windows.ERROR_ALREADY_EXISTS:
		s.logger.Debug("Session already exists, attempting to stop and restart")

		// We need to aggressively close and reopen the session
		propsCopy := *s.properties
		r1, err := advapi32.ControlTrace(
			nil,
			advapi32.EVENT_TRACE_CONTROL_STOP,
			namePtrU16,
			&propsCopy,
		)
		if r1 != 0 {
			s.logger.Warn("Failed to close existing trace, trying to continue anyway",
				zap.Uint64("error_code", uint64(r1)),
				zap.Error(err))
		} else {
			s.logger.Debug("Successfully stopped existing session", zap.String("name", s.name))
		}
		r1, err = advapi32.StartTrace(
			&s.handle,
			namePtrU16,
			s.properties,
		)
		if r1 != 0 {
			return fmt.Errorf("failed to restart trace after stopping(%d): %w", r1, err)
		}
		s.logger.Debug("Successfully restarted session", zap.String("name", s.name))
	default:
		return fmt.Errorf("unexpected error starting trace: error code %d: %v", r1, err)
	}
	s.logger.Debug("Started session", zap.String("name", s.name), zap.Uintptr("handle", uintptr(s.handle)))
	return nil
}

func (s *SessionController) Stop(ctx context.Context) error {
	if s.handle == 0 || s.isAttached {
		s.logger.Debug("Session already stopped or not attached", zap.String("name", s.name), zap.Uintptr("handle", uintptr(s.handle)))
		return nil
	}

	var namePtrU16 *uint16
	var err error

	if namePtrU16, err = syscall.UTF16PtrFromString(s.name); err != nil {
		return fmt.Errorf("failed to convert session name to UTF-16: %w", err)
	}

	propsCopy := *s.properties
	r1, err := advapi32.ControlTrace(
		&s.handle,
		advapi32.EVENT_TRACE_CONTROL_STOP,
		namePtrU16,
		&propsCopy,
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
	props.Wnode.Guid = windows.GUID{}
	props.Wnode.ClientContext = 1
	props.Wnode.Flags = advapi32.WNODE_FLAG_TRACED_GUID
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

// enableProvider enables a provider with the given trace level and match any/all keywords.
// we will attempt to enable the provider up to maxAttempts times if we encounter an error. We do this because sometimes the system does not have resources
// or the provider may not be enabled yet
// if the provider is already enabled, we will attempt to attach to it.
// if we encounter an error, we will return an error.
// if we successfully enable the provider, we will return nil.
func (s *SessionController) enableProvider(handle syscall.Handle, providerGUID *windows.GUID, provider *Provider, traceLevel advapi32.TraceLevel, matchAnyKeyword uint64, matchAllKeyword uint64) error {
	params := advapi32.EnableTraceParameters{
		Version:        2,
		EnableProperty: advapi32.EVENT_ENABLE_PROPERTY_SID,
	}

	const maxAttempts = 5

	for attempts := 0; attempts < maxAttempts; attempts++ {
		if attempts > 0 {
			delay := time.Duration(50*attempts) * time.Millisecond
			s.logger.Debug("Sleeping before provider enable attempt",
				zap.String("provider", provider.Name),
				zap.Duration("delay", delay),
				zap.Int("attempt", attempts+1))
			time.Sleep(delay)
		}

		var r1 syscall.Errno
		var err error
		var panicErr error

		func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Warn("Recovered from EnableTrace panic",
						zap.String("provider", provider.Name),
						zap.Any("recovered", r))
					panicErr = fmt.Errorf("recovered from panic in EnableTrace: %v", r)
				}
			}()
			// timeout is 0 for async enablement
			const timeout = 0
			r1, err = s.enableTrace(
				s.handle,
				providerGUID,
				advapi32.EVENT_CONTROL_CODE_ENABLE_PROVIDER,
				traceLevel,
				provider.MatchAnyKeyword,
				provider.MatchAllKeyword,
				timeout,
				&params,
			)
		}()

		if panicErr != nil {
			return panicErr
		}

		switch r1 {
		case 0:
			s.logger.Debug("Successfully enabled with attempt",
				zap.String("provider", provider.Name),
				zap.Int("attempt", attempts+1))
			return nil
		case windows.ERROR_ALREADY_EXISTS:
			s.logger.Info("provider already enabled, will attempt attaching to it",
				zap.String("provider", provider.Name))
			return s.attach()
		case windows.ERROR_NO_SYSTEM_RESOURCES:
			s.logger.Error("no system resources when enabling provider", zap.Error(err))
			return fmt.Errorf("no system resources when enabling provider: %w", err)
		case windows.ERROR_INVALID_PARAMETER:
			s.logger.Debug("invalid parameters when enabling session trace",
				zap.String("provider", provider.Name),
				zap.Uint64("error_code", uint64(r1)),
				zap.Error(err),
				zap.Int("attempt", attempts+1),
				zap.Int("maxAttempts", maxAttempts))
		case windows.ERROR_TIMEOUT:
			s.logger.Debug("timeout when enabling provider",
				zap.String("provider", provider.Name),
				zap.Int("attempt", attempts+1))
		}

		if attempts == maxAttempts-1 {
			return fmt.Errorf("EnableTrace failed(%d): %w", r1, err)
		}
	}

	return fmt.Errorf("failed to enable provider after %d attempts", maxAttempts)
}

func (s *SessionController) attach() error {
	sessionNamePtr, err := syscall.UTF16PtrFromString(s.name)
	if err != nil {
		return fmt.Errorf("failed to convert session name to UTF16: %w", err)
	}

	temporaryHandle := syscall.Handle(0)
	r1, err := advapi32.ControlTrace(&temporaryHandle, advapi32.EVENT_TRACE_CONTROL_QUERY, sessionNamePtr, s.properties)
	if err != nil {
		return fmt.Errorf("failed to attach to provider: %w", err)
	}

	if r1 != 0 {
		return fmt.Errorf("failed to attach to provider: %w", err)
	}

	s.isAttached = true
	s.handle = syscall.Handle(s.properties.Wnode.Union1)
	return nil
}
