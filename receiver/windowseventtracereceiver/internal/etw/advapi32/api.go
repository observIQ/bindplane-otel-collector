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

package advapi32

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	windows_ "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/windows"
)

// DLL and function references
var (
	advapi32       = windows.NewLazySystemDLL("advapi32.dll")
	startTraceW    = advapi32.NewProc("StartTraceW")
	controlTraceW  = advapi32.NewProc("ControlTraceW")
	openTraceW     = advapi32.NewProc("OpenTraceW")
	enableTraceEx2 = advapi32.NewProc("EnableTraceEx2")
	processTraceW  = advapi32.NewProc("ProcessTrace")
	closeTraceW    = advapi32.NewProc("CloseTrace")
)

/*
ULONG WMIAPI StartTraceW(
            CONTROLTRACE_ID         *TraceId,
  [in]      LPCWSTR                 InstanceName,
  [in, out] PEVENT_TRACE_PROPERTIES Properties
);
*/

// StartTrace starts a trace session using the StartTraceW function
func StartTrace(
	handle *syscall.Handle,
	name *uint16,
	properties *EventTraceProperties) (errorCode syscall.Errno, err error) {
	r, _, err := startTraceW.Call(
		uintptr(unsafe.Pointer(handle)),
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(properties)),
	)
	return syscall.Errno(r), err
}

const (
	EVENT_TRACE_CONTROL_QUERY  uint32 = 0
	EVENT_TRACE_CONTROL_STOP   uint32 = 1
	EVENT_TRACE_CONTROL_UPDATE uint32 = 2
	EVENT_TRACE_CONTROL_FLUSH  uint32 = 3
)

/*
	  typedef struct _WNODE_HEADER {
	  ULONG BufferSize;
	  ULONG ProviderId;
	  __C89_NAMELESS union {
	    ULONG64 HistoricalContext;
	    __C89_NAMELESS struct {
	      ULONG Version;
	      ULONG Linkage;
	    };
	  };
	  __C89_NAMELESS union {
	    ULONG CountLost;
	    HANDLE KernelHandle;
	    LARGE_INTEGER TimeStamp;
	  };
	  GUID Guid;
	  ULONG ClientContext;
	  ULONG Flags;
	} WNODE_HEADER,*PWNODE_HEADER
*/
type WnodeHeader struct {
	BufferSize    uint32
	ProviderId    windows_.GUID
	Union1        uint64
	Union2        int64
	Guid          windows_.GUID
	ClientContext uint32
	Flags         uint32
}

// EventTraceProperties is a direct port of the Windows EVENT_TRACE_PROPERTIES struct
type EventTraceProperties struct {
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

// https://learn.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-controltracew

/*
	ULONG WMIAPI ControlTraceW(
				CONTROLTRACE_ID         TraceId,
	[in]      LPCWSTR                 InstanceName,
	[in, out] PEVENT_TRACE_PROPERTIES Properties,
	[in]      ULONG                   ControlCode
	);
*/

func ControlTrace(handle *syscall.Handle, control uint32, instanceName *uint16, properties *EventTraceProperties) (errorCode syscall.Errno, err error) {
	r, _, err := controlTraceW.Call(
		uintptr(unsafe.Pointer(handle)),
		uintptr(unsafe.Pointer(instanceName)),
		uintptr(unsafe.Pointer(properties)),
		uintptr(control),
	)
	return syscall.Errno(r), err
}

const (
	EVENT_CONTROL_CODE_DISABLE_PROVIDER uint32 = 0
	EVENT_CONTROL_CODE_ENABLE_PROVIDER  uint32 = 1
	EVENT_CONTROL_CODE_CAPTURE_STATE    uint32 = 2
)

type TraceLevel int

const (
	TRACE_LEVEL_NONE        TraceLevel = 0
	TRACE_LEVEL_CRITICAL    TraceLevel = 1
	TRACE_LEVEL_FATAL       TraceLevel = 1
	TRACE_LEVEL_ERROR       TraceLevel = 2
	TRACE_LEVEL_WARNING     TraceLevel = 3
	TRACE_LEVEL_INFORMATION TraceLevel = 4
	TRACE_LEVEL_VERBOSE     TraceLevel = 5
	TRACE_LEVEL_RESERVED6   TraceLevel = 6
	TRACE_LEVEL_RESERVED7   TraceLevel = 7
	TRACE_LEVEL_RESERVED8   TraceLevel = 8
	TRACE_LEVEL_RESERVED9   TraceLevel = 9
)

type processTraceMode int

const (
	PROCESS_TRACE_MODE_REAL_TIME    processTraceMode = 0x00000100
	PROCESS_TRACE_MODE_EVENT_RECORD processTraceMode = 0x10000000
)

type EnableTraceParameters struct {
	Version uint32
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
func EnableTrace(handle syscall.Handle, providerGUID *windows.GUID, controlCode uint32, level TraceLevel, matchAnyKeyword uint64, matchAllKeyword uint64, timeout uint32, enableParameters *EnableTraceParameters) (errorCode syscall.Errno, err error) {
	r, _, err := enableTraceEx2.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(providerGUID)),
		uintptr(controlCode),
		uintptr(uint8(level)),
		uintptr(matchAnyKeyword),
		uintptr(matchAllKeyword),
		uintptr(timeout),
		uintptr(unsafe.Pointer(enableParameters)),
	)

	if r != 0 && r != uintptr(windows.ERROR_INVALID_PARAMETER) {
		errCode := uint32(r)
		return syscall.Errno(errCode), fmt.Errorf("EnableTraceEx2 failed (code=%d): %w", errCode, err)
	}

	return 0, nil
}

// stopTrace stops a trace session using the ControlTraceW function
func StopTrace(sessionName string) error {
	// Create a properties structure for stopping
	sessionNameSize := (len(sessionName) + 1) * 2 // UTF-16 characters plus null terminator
	logFileNameSize := 2                          // null terminator

	propSize := unsafe.Sizeof(EventTraceProperties{})
	totalSize := propSize + uintptr(sessionNameSize) + uintptr(logFileNameSize)

	buffer := make([]byte, totalSize)
	props := (*EventTraceProperties)(unsafe.Pointer(&buffer[0]))

	props.Wnode.BufferSize = uint32(totalSize)
	props.Wnode.Flags = WNODE_FLAG_ALL_DATA

	props.LoggerNameOffset = uint32(propSize)
	props.LogFileNameOffset = uint32(propSize + uintptr(sessionNameSize))

	// Copy the session name to the buffer
	sessionNamePtr, err := syscall.UTF16FromString(sessionName)
	if err != nil {
		return fmt.Errorf("failed to convert session name to UTF-16: %w", err)
	}

	for i, ch := range sessionNamePtr {
		offset := props.LoggerNameOffset + uint32(i*2)
		if int(offset+1) >= len(buffer) {
			break
		}
		buffer[offset] = byte(ch)
		buffer[offset+1] = byte(ch >> 8)
	}

	_, err = ControlTrace(nil, EVENT_TRACE_CONTROL_STOP, nil, props)
	return err
}

type Event struct {
	Flags struct {
		// Use to flag event as being skippable for performance reason
		Skippable bool
	} `json:"-"`

	EventData map[string]interface{} `json:",omitempty"`
	UserData  map[string]interface{} `json:",omitempty"`
	System    struct {
		Channel     string
		Computer    string
		EventID     uint16
		EventType   string `json:",omitempty"`
		EventGuid   string `json:",omitempty"`
		Correlation struct {
			ActivityID        string
			RelatedActivityID string
		}
		Execution struct {
			ProcessID uint32
			ThreadID  uint32
		}
		Keywords struct {
			Value uint64
			Name  string
		}
		Level struct {
			Value uint8
			Name  string
		}
		Opcode struct {
			Value uint8
			Name  string
		}
		Task struct {
			Value uint8
			Name  string
		}
		Provider struct {
			Guid string
			Name string
		}
		TimeCreated struct {
			SystemTime time.Time
		}
	}
	ExtendedData []string `json:",omitempty"`
}

/*
	typedef struct _EVENT_TRACE_LOGFILEW {
	  LPWSTR                        LogFileName;
	  LPWSTR                        LoggerName;
	  LONGLONG                      CurrentTime;
	  ULONG                         BuffersRead;
	  union {
	    ULONG LogFileMode;
	    ULONG ProcessTraceMode;
	  } DUMMYUNIONNAME;
	  EVENT_TRACE                   CurrentEvent;
	  TRACE_LOGFILE_HEADER          LogfileHeader;
	  PEVENT_TRACE_BUFFER_CALLBACKW BufferCallback;
	  ULONG                         BufferSize;
	  ULONG                         Filled;
	  ULONG                         EventsLost;
	  union {
	    PEVENT_CALLBACK        EventCallback;
	    PEVENT_RECORD_CALLBACK EventRecordCallback;
	  } DUMMYUNIONNAME2;
	  ULONG                         IsKernelTrace;
	  PVOID                         Context;
	} EVENT_TRACE_LOGFILEW, *PEVENT_TRACE_LOGFILEW;
*/
type EventTraceLogfile struct {
	LogFileName   *uint16
	LoggerName    *uint16
	CurrentTime   int64
	BuffersRead   uint32
	Union1        uint32
	CurrentEvent  EventTrace
	LogfileHeader TraceLogfileHeader
	//BufferCallback *EventTraceBufferCallback
	BufferCallback uintptr
	BufferSize     uint32
	Filled         uint32
	EventsLost     uint32
	Callback       uintptr
	IsKernelTrace  uint32
	Context        uintptr
}

func (e *EventTraceLogfile) SetProcessTraceMode(ptm uint32) {
	e.Union1 = ptm
}

/*
	typedef struct _EVENT_TRACE {
	  EVENT_TRACE_HEADER Header;
	  ULONG              InstanceId;
	  ULONG              ParentInstanceId;
	  GUID               ParentGuid;
	  PVOID              MofData;
	  ULONG              MofLength;
	  union {
	    ULONG              ClientContext;
	    ETW_BUFFER_CONTEXT BufferContext;
	  } DUMMYUNIONNAME;
	} EVENT_TRACE, *PEVENT_TRACE;
*/
type EventTrace struct {
	Header           EventTraceHeader
	InstanceId       uint32
	ParentInstanceId uint32
	ParentGuid       windows.GUID
	MofData          uintptr
	MofLength        uint32
	UnionCtx         uint32
}

/*
typedef struct _EVENT_TRACE_HEADER {
  USHORT        Size;
  union {
    USHORT FieldTypeFlags;
    struct {
      UCHAR HeaderType;
      UCHAR MarkerFlags;
    } DUMMYSTRUCTNAME;
  } DUMMYUNIONNAME;
  union {
    ULONG Version;
    struct {
      UCHAR  Type;
      UCHAR  Level;
      USHORT Version;
    } Class;
  } DUMMYUNIONNAME2;
  ULONG         ThreadId;
  ULONG         ProcessId;
  LARGE_INTEGER TimeStamp;
  union {
    GUID      Guid;
    ULONGLONG GuidPtr;
  } DUMMYUNIONNAME3;
  union {
    struct {
      ULONG KernelTime;
      ULONG UserTime;
    } DUMMYSTRUCTNAME; uint64
    ULONG64 ProcessorTime; uint64
    struct {
      ULONG ClientContext;
      ULONG Flags;
    } DUMMYSTRUCTNAME2; uint64
  } DUMMYUNIONNAME4;
} EVENT_TRACE_HEADER, *PEVENT_TRACE_HEADER;
*/

// sizeof: 0x30 (48)
type EventTraceHeader struct {
	Size      uint16
	Union1    uint16
	Union2    uint32
	ThreadId  uint32
	ProcessId uint32
	TimeStamp int64
	Union3    [16]byte
	Union4    uint64
}

/*
	typedef struct _TRACE_LOGFILE_HEADER {
	  ULONG                 BufferSize;
	  union {
	    ULONG  Version;
	    struct {
	      UCHAR MajorVersion;
	      UCHAR MinorVersion;
	      UCHAR SubVersion;
	      UCHAR SubMinorVersion;
	    } VersionDetail;
	  };
	  ULONG                 ProviderVersion;
	  ULONG                 NumberOfProcessors;
	  LARGE_INTEGER         EndTime;
	  ULONG                 TimerResolution;
	  ULONG                 MaximumFileSize;
	  ULONG                 LogFileMode;
	  ULONG                 BuffersWritten;
	  union {
	    GUID   LogInstanceGuid;
	    struct {
	      ULONG StartBuffers;
	      ULONG PointerSize;
	      ULONG EventsLost;
	      ULONG CpuSpeedInMHz;
	    };
	  };
	  LPWSTR                LoggerName;
	  LPWSTR                LogFileName;
	  TIME_ZONE_INFORMATION TimeZone;
	  LARGE_INTEGER         BootTime;
	  LARGE_INTEGER         PerfFreq;
	  LARGE_INTEGER         StartTime;
	  ULONG                 ReservedFlags;
	  ULONG                 BuffersLost;
	} TRACE_LOGFILE_HEADER, *PTRACE_LOGFILE_HEADER;
*/
type TraceLogfileHeader struct {
	BufferSize         uint32
	VersionUnion       uint32
	ProviderVersion    uint32
	NumberOfProcessors uint32
	EndTime            int64
	TimerResolution    uint32
	MaximumFileSize    uint32
	LogFileMode        uint32
	BuffersWritten     uint32
	Union1             [16]byte
	LoggerName         *uint16
	LogFileName        *uint16
	TimeZone           TimeZoneInformation
	BootTime           int64
	PerfFreq           int64
	StartTime          int64
	ReservedFlags      uint32
	BuffersLost        uint32
}

/*
typedef struct _TIME_ZONE_INFORMATION {
  LONG       Bias;
  WCHAR      StandardName[32];
  SYSTEMTIME StandardDate;
  LONG       StandardBias;
  WCHAR      DaylightName[32];
  SYSTEMTIME DaylightDate;
  LONG       DaylightBias;
} TIME_ZONE_INFORMATION, *PTIME_ZONE_INFORMATION, *LPTIME_ZONE_INFORMATION;
*/

type TimeZoneInformation struct {
	Bias         int32
	StandardName [32]uint16
	StandardDate SystemTime
	StandardBias int32
	DaylightName [32]uint16
	DaylightDate SystemTime
	DaylighBias  int32
}

/*
typedef struct _SYSTEMTIME {
  WORD wYear;
  WORD wMonth;
  WORD wDayOfWeek;
  WORD wDay;
  WORD wHour;
  WORD wMinute;
  WORD wSecond;
  WORD wMilliseconds;
} SYSTEMTIME, *PSYSTEMTIME, *LPSYSTEMTIME;
*/
// sizeof: 0x10 (OK)
type SystemTime struct {
	Year         uint16
	Month        uint16
	DayOfWeek    uint16
	Day          uint16
	Hour         uint16
	Minute       uint16
	Second       uint16
	Milliseconds uint16
}

/*
https://learn.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-opentracew

ETW_APP_DECLSPEC_DEPRECATED PROCESSTRACE_HANDLE WMIAPI OpenTraceW(
  [in, out] PEVENT_TRACE_LOGFILEW Logfile
);
*/

func OpenTrace(logfile *EventTraceLogfile) (syscall.Handle, error) {
	r, _, err := openTraceW.Call(
		uintptr(unsafe.Pointer(logfile)))

	if err.(syscall.Errno) == 0 {
		return syscall.Handle(r), nil
	}
	return syscall.Handle(r), err
}

/*
ETW_APP_DECLSPEC_DEPRECATED ULONG WMIAPI ProcessTrace(
  [in] PROCESSTRACE_HANDLE *HandleArray,
  [in] ULONG               HandleCount,
  [in] LPFILETIME          StartTime,
  [in] LPFILETIME          EndTime
);
*/

// processTrace processes a trace session using the ProcessTraceW function
func ProcessTrace(handle *syscall.Handle) error {
	r, _, err := processTraceW.Call(
		uintptr(unsafe.Pointer(handle)),
		uintptr(1),
		uintptr(unsafe.Pointer(nil)),
		uintptr(unsafe.Pointer(nil)),
	)
	if r != 0 {
		return fmt.Errorf("ProcessTraceW failed: %w", err)
	}
	return nil
}

// CloseTrace closes a trace session using the CloseTraceW given a trace handle from OpenTrace
func CloseTrace(traceHandle syscall.Handle) (errorCode syscall.Errno, err error) {
	r, _, err := closeTraceW.Call(
		uintptr(traceHandle),
	)
	// if we're pending a close, we can ignore the error
	if r != 0 && r != uintptr(windows.ERROR_CTX_CLOSE_PENDING) {
		return syscall.Errno(r), fmt.Errorf("CloseTraceW failed(%d): %w", r, err)
	}
	return syscall.Errno(r), nil
}

const (
	EVENT_TRACE_REAL_TIME_MODE = 0x00000100
)

const (
	WNODE_FLAG_ALL_DATA              uint32 = 0x00000001
	WNODE_FLAG_SINGLE_INSTANCE       uint32 = 0x00000002
	WNODE_FLAG_SINGLE_ITEM           uint32 = 0x00000004
	WNODE_FLAG_EVENT_ITEM            uint32 = 0x00000008
	WNODE_FLAG_FIXED_INSTANCE_SIZE   uint32 = 0x00000010
	WNODE_FLAG_TOO_SMALL             uint32 = 0x00000020
	WNODE_FLAG_INSTANCES_SAME        uint32 = 0x00000040
	WNODE_FLAG_STATIC_INSTANCE_NAMES uint32 = 0x00000080
	WNODE_FLAG_INTERNAL              uint32 = 0x00000100
	WNODE_FLAG_USE_TIMESTAMP         uint32 = 0x00000200
	WNODE_FLAG_PERSIST_EVENT         uint32 = 0x00000400
	WNODE_FLAG_EVENT_REFERENCE       uint32 = 0x00002000
	WNODE_FLAG_ANSI_INSTANCENAwin32  uint32 = 0x00040000
	WNODE_FLAG_USE_GUID_PTR          uint32 = 0x00080000
	WNODE_FLAG_USE_MOF_PTR           uint32 = 0x00100000
	WNODE_FLAG_NO_HEADER             uint32 = 0x00200000
	WNODE_FLAG_SEND_DATA_BLOCK       uint32 = 0x00400000
	WNODE_FLAG_TRACED_GUID           uint32 = 0x00020000
)

const (
	EVENT_HEADER_FLAG_EXTENDED_INFO   = 0x0001
	EVENT_HEADER_FLAG_PRIVATE_SESSION = 0x0002
	EVENT_HEADER_FLAG_STRING_ONLY     = 0x0004
	EVENT_HEADER_FLAG_TRACE_MESSAGE   = 0x0008
	EVENT_HEADER_FLAG_NO_CPUTIME      = 0x0010
	EVENT_HEADER_FLAG_32_BIT_HEADER   = 0x0020
	EVENT_HEADER_FLAG_64_BIT_HEADER   = 0x0040
	EVENT_HEADER_FLAG_CLASSIC_HEADER  = 0x0100
	EVENT_HEADER_FLAG_PROCESSOR_INDEX = 0x0200
)

/*
	typedef struct _EVENT_RECORD {
	  EVENT_HEADER                     EventHeader;
	  ETW_BUFFER_CONTEXT               BufferContext;
	  USHORT                           ExtendedDataCount;
	  USHORT                           UserDataLength;
	  PEVENT_HEADER_EXTENDED_DATA_ITEM ExtendedData;
	  PVOID                            UserData;
	  PVOID                            UserContext;
	} EVENT_RECORD, *PEVENT_RECORD;
*/
type EventRecord struct {
	EventHeader       EventHeader
	BufferContext     EtwBufferContext
	ExtendedDataCount uint16
	UserDataLength    uint16
	ExtendedData      *EventHeaderExtendedDataItem
	UserData          uintptr
	UserContext       uintptr
}

func (e *EventRecord) PointerSize() uint32 {
	if e.EventHeader.Flags&EVENT_HEADER_FLAG_32_BIT_HEADER == EVENT_HEADER_FLAG_32_BIT_HEADER {
		return 4
	}
	return 8
}

/*
	typedef struct _EVENT_HEADER {
	  USHORT           Size;
	  USHORT           HeaderType;
	  USHORT           Flags;
	  USHORT           EventProperty;
	  ULONG            ThreadId;
	  ULONG            ProcessId;
	  LARGE_INTEGER    TimeStamp;
	  GUID             ProviderId;
	  EVENT_DESCRIPTOR EventDescriptor;
	  union {
	    struct {
	      ULONG KernelTime;
	      ULONG UserTime;
	    } DUMMYSTRUCTNAME;
	    ULONG64 ProcessorTime;
	  } DUMMYUNIONNAME;
	  GUID             ActivityId;
	} EVENT_HEADER, *PEVENT_HEADER;
*/
type EventHeader struct {
	Size            uint16
	HeaderType      uint16
	Flags           uint16
	EventProperty   uint16
	ThreadId        uint32
	ProcessId       uint32
	TimeStamp       int64
	ProviderId      windows_.GUID
	EventDescriptor EventDescriptor
	Time            int64
	ActivityId      windows_.GUID
}

func (e *EventHeader) UTC() time.Time {
	nano := int64(10000000)
	sec := int64(float64(e.TimeStamp)/float64(nano) - 11644473600.0)
	nsec := ((e.TimeStamp - 11644473600*nano) - sec*nano) * 100
	return time.Unix(sec, nsec).UTC()
}

/*
	typedef struct _EVENT_DESCRIPTOR {
	  USHORT    Id;
	  UCHAR     Version;
	  UCHAR     Channel;
	  UCHAR     Level;
	  UCHAR     Opcode;
	  USHORT    Task;
	  ULONGLONG Keyword;
	} EVENT_DESCRIPTOR, *PEVENT_DESCRIPTOR;
*/
type EventDescriptor struct {
	Id      uint16
	Version uint8
	Channel uint8
	Level   uint8
	Opcode  uint8
	Task    uint16
	Keyword uint64
}

/*
	typedef struct _EVENT_HEADER_EXTENDED_DATA_ITEM {
	  USHORT    Reserved1;
	  USHORT    ExtType;
	  struct {
	    USHORT Linkage : 1;
	    USHORT Reserved2 : 15;
	  };
	  USHORT    DataSize;
	  ULONGLONG DataPtr;
	} EVENT_HEADER_EXTENDED_DATA_ITEM, *PEVENT_HEADER_EXTENDED_DATA_ITEM;
*/
type EventHeaderExtendedDataItem struct {
	Reserved1      uint16
	ExtType        uint16
	InternalStruct uint16
	DataSize       uint16
	DataPtr        uintptr
}

/*
typedef struct _ETW_BUFFER_CONTEXT {
  union {
    struct {
      UCHAR ProcessorNumber;
      UCHAR Alignment;
    } DUMMYSTRUCTNAME; // siize UCHAR
    USHORT ProcessorIndex; // USHORT
  } DUMMYUNIONNAME; // USHORT
  USHORT LoggerId;
} ETW_BUFFER_CONTEXT, *PETW_BUFFER_CONTEXT;
*/
// sizeof: 0x4 (OK)
type EtwBufferContext struct {
	Union    uint16
	LoggerId uint16
}
