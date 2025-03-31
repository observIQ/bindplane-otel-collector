//go:build windows
// +build windows

package etw

import (
	"syscall"
	"unsafe"
)

/*
StartTraceW API wrapper generated from prototype
EXTERN_C ULONG WMIAPI StartTraceW (
	 PTRACEHANDLE TraceHandle,
	 LPCWSTR InstanceName,
	 PEVENT_TRACE_PROPERTIES Properties);
*/
func StartTrace(traceHandle *syscall.Handle,
	instanceName *uint16,
	properties *EventTraceProperties) error {
	r1, _, _ := startTraceW.Call(
		uintptr(unsafe.Pointer(traceHandle)),
		uintptr(unsafe.Pointer(instanceName)),
		uintptr(unsafe.Pointer(properties)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
StopTrace API wrapper generated from prototype
EXTERN_C ULONG WMIAPI StopTrace(
	  IN TRACEHANDLE TraceHandle,
	 IN LPCWSTR InstanceName OPTIONAL,
	 IN OUT PEVENT_TRACE_PROPERTIES Properties);

NB: This function is obsolete. The ControlTrace function supersedes this function.
*/
func StopTrace(
	traceHandle syscall.Handle,
	instanceName *uint16,
	properties *EventTraceProperties) error {
	r1, _, _ := stopTraceW.Call(
		uintptr(traceHandle),
		uintptr(unsafe.Pointer(instanceName)),
		uintptr(unsafe.Pointer(properties)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
EnableTraceEx2 API wrapper generated from prototype
EXTERN_C ULONG WMIAPI EnableTraceEx2 (
	 TRACEHANDLE TraceHandle,
	 LPCGUID ProviderId,
	 ULONG ControlCode,
	 UCHAR Level,
	 ULONGLONG MatchAnyKeyword,
	 ULONGLONG MatchAllKeyword,
	 ULONG Timeout,
	 PENABLE_TRACE_PARAMETERS EnableParameters);
*/
func EnableTraceEx2(traceHandle syscall.Handle,
	providerId *GUID,
	controlCode uint32,
	level uint8,
	matchAnyKeyword uint64,
	matchAllKeyword uint64,
	timeout uint32,
	enableParameters *EnableTraceParameters) error {
	r1, _, _ := enableTraceEx2.Call(
		uintptr(traceHandle),
		uintptr(unsafe.Pointer(providerId)),
		uintptr(controlCode),
		uintptr(level),
		uintptr(matchAnyKeyword),
		uintptr(matchAllKeyword),
		uintptr(timeout),
		uintptr(unsafe.Pointer(enableParameters)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
ProcessTrace API wrapper generated from prototype
EXTERN_C ULONG WMIAPI ProcessTrace (
	 PTRACEHANDLE HandleArray,
	 ULONG HandleCount,
	 LPFILETIME StartTime,
	 LPFILETIME EndTime);
*/
func ProcessTrace(handleArray *syscall.Handle,
	handleCount uint32,
	startTime *FileTime,
	endTime *FileTime) error {
	r1, _, _ := processTrace.Call(
		uintptr(unsafe.Pointer(handleArray)),
		uintptr(handleCount),
		uintptr(unsafe.Pointer(startTime)),
		uintptr(unsafe.Pointer(endTime)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
OpenTraceW API wrapper generated from prototype
EXTERN_C TRACEHANDLE WMIAPI OpenTraceW (
	 PEVENT_TRACE_LOGFILEW Logfile);
*/
func OpenTrace(logfile *EventTraceLogfile) (syscall.Handle, error) {
	r1, _, err := openTraceW.Call(
		uintptr(unsafe.Pointer(logfile)))
	// This call stores error in lastError so we can keep it like this
	if err.(syscall.Errno) == 0 {
		return syscall.Handle(r1), nil
	}
	return syscall.Handle(r1), err
}

/*
ControlTraceW API wrapper generated from prototype
EXTERN_C ULONG WMIAPI ControlTraceW (
	 TRACEHANDLE TraceHandle,
	 LPCWSTR InstanceName,
	 PEVENT_TRACE_PROPERTIES Properties,
	 ULONG ControlCode);
*/
func ControlTrace(traceHandle syscall.Handle,
	instanceName *uint16,
	properties *EventTraceProperties,
	controlCode uint32) error {
	r1, _, _ := controlTraceW.Call(
		uintptr(traceHandle),
		uintptr(unsafe.Pointer(instanceName)),
		uintptr(unsafe.Pointer(properties)),
		uintptr(controlCode))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
CloseTrace API wrapper generated from prototype
EXTERN_C ULONG WMIAPI CloseTrace (
	 TRACEHANDLE TraceHandle);
*/
func CloseTrace(traceHandle syscall.Handle) error {
	r1, _, _ := closeTrace.Call(
		uintptr(traceHandle))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
EventAccessQuery API wrapper generated from prototype
ULONG EVNTAPI EventAccessQuery (
	 LPGUID Guid,
	 PSECURITY_DESCRIPTOR Buffer,
	 PULONG BufferSize);
*/
func EventAccessQuery(
	guid *GUID,
	buffer *SecurityDescriptor,
	bufferSize *uint32) error {
	r1, _, _ := eventAccessQuery.Call(
		uintptr(unsafe.Pointer(guid)),
		uintptr(unsafe.Pointer(buffer)),
		uintptr(unsafe.Pointer(bufferSize)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
EventAccessControl API wrapper generated from prototype
ULONG EVNTAPI EventAccessControl (
	 LPGUID Guid,
	 ULONG Operation,
	 PSID Sid,
	 ULONG Rights,
	 BOOLEAN AllowOrDeny);
*/
func EventAccessControl(guid *GUID,
	operation uint32,
	sid *SID,
	rights uint32,
	allowOrDeny bool) error {

	// we need a byte not a bool
	var bAllowOrDeny byte

	if allowOrDeny {
		bAllowOrDeny = 1
	}

	r1, _, _ := eventAccessControl.Call(
		uintptr(unsafe.Pointer(guid)),
		uintptr(operation),
		uintptr(unsafe.Pointer(sid)),
		uintptr(rights),
		uintptr(bAllowOrDeny))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
ConvertSecurityDescriptorToStringSecurityDescriptorW API wrapper generated from prototype
WINADVAPI WINBOOL WINAPI ConvertSecurityDescriptorToStringSecurityDescriptorW(
	 PSECURITY_DESCRIPTOR SecurityDescriptor,
	DWORD RequestedStringSDRevision,
	SECURITY_INFORMATION SecurityInformation,
	LPWSTR *StringSecurityDescriptor,
	PULONG StringSecurityDescriptorLen);
*/
func ConvertSecurityDescriptorToStringSecurityDescriptorW(
	securityDescriptor *SecurityDescriptor,
	requestedStringSDRevision uint32,
	securityInformation SecurityInformation,
) (string, error) {
	var stringSecurityDescriptor uint16
	var stringSecurityDescriptorLen uint32

	pStringSecurityDescriptor := &stringSecurityDescriptor

	_, _, err := convertSecurityDescriptorToStringSecurityDescriptorW.Call(
		uintptr(unsafe.Pointer(securityDescriptor)),
		uintptr(requestedStringSDRevision),
		uintptr(securityInformation),
		uintptr(unsafe.Pointer(&pStringSecurityDescriptor)),
		uintptr(unsafe.Pointer(&stringSecurityDescriptorLen)))
	if err == ERROR_SUCCESS {
		s := UTF16PtrToString(pStringSecurityDescriptor)
		if _, err := syscall.LocalFree(syscall.Handle(unsafe.Pointer(pStringSecurityDescriptor))); err != nil {
			return "", err
		}
		return s, nil
	}
	return "", err
}

/*
ConvertStringSidToSidW API wrapper generated from prototype
WINADVAPI WINBOOL WINAPI ConvertStringSidToSidW(
	 LPCWSTR StringSid,
	PSID *Sid);
*/
func ConvertStringSidToSidW(stringSid string) (sid *SID, err error) {

	var rc uintptr
	var utf16Sid *uint16

	if utf16Sid, err = syscall.UTF16PtrFromString(stringSid); err != nil {
		return
	}

	rc, _, err = convertStringSidToSidW.Call(
		uintptr(unsafe.Pointer(utf16Sid)),
		uintptr(unsafe.Pointer(&sid)))

	if rc != 0 {
		// success
		err = nil
	}

	return
}
