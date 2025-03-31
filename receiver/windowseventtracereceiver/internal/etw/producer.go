//go:build windows
// +build windows

package etw

import (
	"syscall"
	"unsafe"
)

const (
	NtKernelLogger = "NT Kernel Logger"
	//  0x9e814aad, 0x3204, 0x11d2, 0x9a, 0x82, 0x00, 0x60, 0x08, 0xa8, 0x69, 0x39
)

var (
	systemTraceControlGuid = MustParseGUIDFromString("{9E814AAD-3204-11D2-9A82-006008A86939}")
)

type Session interface {
	TraceName() string
	Providers() []Provider
}

type RealTimeSession struct {
	properties    *EventTraceProperties
	sessionHandle syscall.Handle

	traceName string
	providers []Provider
}

// NewRealTimeSession creates a new ETW session to receive events
// in real time
func NewRealTimeSession(name string) (p *RealTimeSession) {
	p = &RealTimeSession{}
	p.properties = NewRealTimeEventTraceSessionProperties(name)
	p.traceName = name
	p.providers = make([]Provider, 0)
	return
}

// NewKernelRealTimeSession creates a new ETW session to receive
// NT Kernel Logger events in real time
func NewKernelRealTimeSession(flags ...uint32) (p *RealTimeSession) {
	p = NewRealTimeSession(NtKernelLogger)
	// guid must be set for Kernel Session
	p.properties.Wnode.Guid = *systemTraceControlGuid
	for _, flag := range flags {
		p.properties.EnableFlags |= flag
	}
	return
}

// IsStarted returns true if the session is already started
func (p *RealTimeSession) IsStarted() bool {
	return p.sessionHandle != 0
}

// Start starts the session
func (p *RealTimeSession) Start() (err error) {
	var u16TraceName *uint16

	if u16TraceName, err = syscall.UTF16PtrFromString(p.traceName); err != nil {
		return err
	}

	if !p.IsStarted() {
		if err = StartTrace(&p.sessionHandle, u16TraceName, p.properties); err != nil {
			// we handle the case where the trace already exists
			if err == ERROR_ALREADY_EXISTS {
				// we have to use a copy of properties as ControlTrace modifies
				// the structure and if we don't do that we cannot StartTrace later
				prop := *p.properties
				// we close the trace first
				ControlTrace(0, u16TraceName, &prop, EVENT_TRACE_CONTROL_STOP)
				return StartTrace(&p.sessionHandle, u16TraceName, p.properties)
			}
			return
		}
	}

	return
}

// EnableProvider enables the session to receive events from a given provider
func (p *RealTimeSession) EnableProvider(prov Provider) (err error) {
	var guid *GUID

	// If the trace is not started yet we have to start it
	// otherwise we cannot enable provider
	if !p.IsStarted() {
		if err = p.Start(); err != nil {
			return
		}
	}

	if guid, err = ParseGUID(prov.GUID); err != nil {
		return
	}

	params := EnableTraceParameters{
		Version: 2,
		// Does not seem to bring valuable information
		//EnableProperty: EVENT_ENABLE_PROPERTY_PROCESS_START_KEY,
	}

	if len(prov.Filter) > 0 {
		fds := prov.BuildFilterDesc()
		if len(fds) > 0 {
			params.EnableFilterDesc = (*EventFilterDescriptor)(unsafe.Pointer(&fds[0]))
			params.FilterDescCount = uint32(len(fds))
		}
	}

	if err = EnableTraceEx2(
		p.sessionHandle,
		guid,
		EVENT_CONTROL_CODE_ENABLE_PROVIDER,
		prov.EnableLevel,
		prov.MatchAnyKeyword,
		prov.MatchAllKeyword,
		0,
		&params,
	); err != nil {
		return
	}

	p.providers = append(p.providers, prov)

	return
}

// TraceName implements Session interface
func (p *RealTimeSession) TraceName() string {
	return p.traceName
}

// Providers implements Session interface
func (p *RealTimeSession) Providers() []Provider {
	return p.providers
}

// Stop stops the session
func (p *RealTimeSession) Stop() error {
	return ControlTrace(p.sessionHandle, nil, p.properties, EVENT_TRACE_CONTROL_STOP)
}
