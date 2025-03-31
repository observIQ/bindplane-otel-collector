//go:build windows

package advapi32

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// DLL and function references
var (
	advapi32       = windows.NewLazySystemDLL("advapi32.dll")
	startTraceW    = advapi32.NewProc("StartTraceW")
	controlTraceW  = advapi32.NewProc("ControlTraceW")
	openTraceW     = advapi32.NewProc("OpenTraceW")
	enableTraceEx2 = advapi32.NewProc("EnableTraceEx2")
	processTraceW  = advapi32.NewProc("ProcessTraceW")
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
	properties *EventTraceProperties) error {
	r, _, err := startTraceW.Call(
		uintptr(unsafe.Pointer(handle)),
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(properties)),
	)
	if r != 0 {
		errCode := uint32(r)

		// Translate error code
		switch errCode {
		case 5:
			return fmt.Errorf("access denied (code=%d) - try running as Administrator", errCode)
		case 183:
			return fmt.Errorf("trace session already exists (code=%d) - check for existing sessions", errCode)
		case 161:
			return fmt.Errorf("invalid path (code=%d) - check buffer layout or run as Administrator", errCode)
		default:
			return fmt.Errorf("failed to start trace (code=%d): %v", errCode, err)
		}
	}

	return nil
}

type eventTraceControl int

const (
	EVENT_TRACE_CONTROL_QUERY  eventTraceControl = 0
	EVENT_TRACE_CONTROL_STOP   eventTraceControl = 1
	EVENT_TRACE_CONTROL_UPDATE eventTraceControl = 2
	EVENT_TRACE_CONTROL_FLUSH  eventTraceControl = 3
)

type WnodeHeader struct {
	BufferSize    uint32
	ProviderId    windows.GUID
	Union1        uint64
	Union2        int64
	Guid          windows.GUID
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

/*
	typedef struct _EVENT_TRACE_PROPERTIES_V2 {
	  WNODE_HEADER             Wnode;
	  ULONG                    BufferSize;
	  ULONG                    MinimumBuffers;
	  ULONG                    MaximumBuffers;
	  ULONG                    MaximumFileSize;
	  ULONG                    LogFileMode;
	  ULONG                    FlushTimer;
	  ULONG                    EnableFlags;
	  union {
	    LONG AgeLimit;
	    LONG FlushThreshold;
	  } DUMMYUNIONNAME;
	  ULONG                    NumberOfBuffers;
	  ULONG                    FreeBuffers;
	  ULONG                    EventsLost;
	  ULONG                    BuffersWritten;
	  ULONG                    LogBuffersLost;
	  ULONG                    RealTimeBuffersLost;
	  HANDLE                   LoggerThreadId;
	  ULONG                    LogFileNameOffset;
	  ULONG                    LoggerNameOffset;
	  union {
	    struct {
	      ULONG VersionNumber : 8;
	    } DUMMYSTRUCTNAME;
	    ULONG V2Control;
	  } DUMMYUNIONNAME2;
	  ULONG                    FilterDescCount;
	  PEVENT_FILTER_DESCRIPTOR FilterDesc;
	  union {
	    struct {
	      ULONG Wow : 1;
	      ULONG QpcDeltaTracking : 1;
	      ULONG LargeMdlPages : 1;
	      ULONG ExcludeKernelStack : 1;
	    } DUMMYSTRUCTNAME;
	    ULONG64 V2Options;
	  } DUMMYUNIONNAME3;
	} EVENT_TRACE_PROPERTIES_V2, *PEVENT_TRACE_PROPERTIES_V2;
*/
type EventTracePropertiesV2 struct {
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
	VersionNumber       uint32
	V2Control           uint32
	FilterDescCount     uint32
	FilterDesc          *EventFilterDescriptor
	Wow                 uint32
	QpcDeltaTracking    uint32
	LargeMdlPages       uint32
	ExcludeKernelStack  uint32
	V2Options           uint64
}

type EventFilterDescriptor struct {
	Size            uint32
	Flags           uint32
	FilterKeyOffset uint32
	FilterKeySize   uint32
}

/*
https://learn.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-controltracew
	ULONG WMIAPI ControlTraceW(
				CONTROLTRACE_ID         TraceId,
	[in]      LPCWSTR                 InstanceName,
	[in, out] PEVENT_TRACE_PROPERTIES Properties,
	[in]      ULONG                   ControlCode
	);
*/

func ControlTrace(handle *syscall.Handle, control eventTraceControl, instanceName *uint16, properties *EventTraceProperties, timeout uintptr) error {
	r, _, err := controlTraceW.Call(
		uintptr(unsafe.Pointer(handle)),
		uintptr(control),
		uintptr(unsafe.Pointer(instanceName)),
		uintptr(unsafe.Pointer(properties)),
		uintptr(timeout),
	)
	if r != 0 {
		return fmt.Errorf("ControlTraceW failed: %w", err)
	}
	return nil
}

type eventControlCode int

const (
	EVENT_CONTROL_CODE_DISABLE_PROVIDER eventControlCode = 0
	EVENT_CONTROL_CODE_ENABLE_PROVIDER  eventControlCode = 1
	EVENT_CONTROL_CODE_CAPTURE_STATE    eventControlCode = 2
)

type traceLevel int

const (
	TRACE_LEVEL_NONE        traceLevel = 0
	TRACE_LEVEL_CRITICAL    traceLevel = 1
	TRACE_LEVEL_FATAL       traceLevel = 1
	TRACE_LEVEL_ERROR       traceLevel = 2
	TRACE_LEVEL_WARNING     traceLevel = 3
	TRACE_LEVEL_INFORMATION traceLevel = 4
	TRACE_LEVEL_VERBOSE     traceLevel = 5
	TRACE_LEVEL_RESERVED6   traceLevel = 6
	TRACE_LEVEL_RESERVED7   traceLevel = 7
	TRACE_LEVEL_RESERVED8   traceLevel = 8
	TRACE_LEVEL_RESERVED9   traceLevel = 9
)

type processTraceMode int

const (
	PROCESS_TRACE_MODE_REAL_TIME    processTraceMode = 0x00000100
	PROCESS_TRACE_MODE_EVENT_RECORD processTraceMode = 0x10000000
)

/*
https://learn.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-enabletraceex2

ULONG WMIAPI EnableTraceEx2(
  [in]           CONTROLTRACE_ID          TraceId,
  [in]           LPCGUID                  ProviderId,
  [in]           ULONG                    ControlCode,
  [in]           UCHAR                    Level,
  [in]           ULONGLONG                MatchAnyKeyword,
  [in]           ULONGLONG                MatchAllKeyword,
  [in]           ULONG                    Timeout,
  [in, optional] PENABLE_TRACE_PARAMETERS EnableParameters
);
*/

type EnableTraceParameters struct {
	Version uint32
}

// enableTrace enables a trace session using the EnableTraceEx2 function
func EnableTrace(handle syscall.Handle, providerGUID windows.GUID, controlCode eventControlCode, level traceLevel, matchAnyKeyword uintptr, matchAllKeyword uintptr, timeout uintptr, enableParameters EnableTraceParameters) error {
	r, _, err := enableTraceEx2.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&providerGUID)),
		uintptr(controlCode),
		uintptr(level),
		uintptr(matchAnyKeyword),
		uintptr(matchAllKeyword),
		uintptr(timeout),
		uintptr(unsafe.Pointer(&enableParameters)),
	)

	if r != 0 {
		errCode := uint32(r)
		return fmt.Errorf("EnableTraceEx2 failed (code=%d): %w", errCode, err)
	}

	return nil
}

// stopTrace stops a trace session using the ControlTraceW function
func StopTrace(sessionName string) error {
	// Create a properties structure for stopping
	sessionNameSize := (len(sessionName) + 1) * 2 // UTF-16 characters plus null terminator
	logFileNameSize := 2                          // Just null terminator

	propSize := unsafe.Sizeof(EventTraceProperties{})
	totalSize := propSize + uintptr(sessionNameSize) + uintptr(logFileNameSize)

	buffer := make([]byte, totalSize)
	props := (*EventTraceProperties)(unsafe.Pointer(&buffer[0]))

	props.Wnode.BufferSize = uint32(totalSize)
	props.Wnode.Flags = WNODE_FLAG_TRACED_GUID

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

	return ControlTrace(nil, EVENT_TRACE_CONTROL_STOP, nil, props, 0)
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
	r1, _, err := openTraceW.Call(
		uintptr(unsafe.Pointer(logfile)))
	// This call stores error in lastError so we can keep it like this
	if err.(syscall.Errno) == 0 {
		return syscall.Handle(r1), nil
	}
	return syscall.Handle(r1), err
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
func ProcessTrace(handles []syscall.Handle) error {
	r, _, err := processTraceW.Call(
		uintptr(unsafe.Pointer(&handles)),
		uintptr(len(handles)),
		uintptr(unsafe.Pointer(nil)),
		uintptr(unsafe.Pointer(nil)),
	)
	if r != 0 {
		return fmt.Errorf("ProcessTraceW failed: %w", err)
	}
	return nil
}

const (
	EVENT_TRACE_REAL_TIME_MODE       = 0x00000100
	EVENT_TRACE_DELAY_OPEN_FILE_MODE = 0x00000200
	EVENT_TRACE_BUFFERING_MODE       = 0x00000400
	EVENT_TRACE_PRIVATE_LOGGER_MODE  = 0x00000800
	EVENT_TRACE_ADD_HEADER_MODE      = 0x00001000
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
