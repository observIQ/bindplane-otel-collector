//go:build windows
// +build windows

package etw

import (
	"syscall"
	"time"
	"unsafe"
)

const (
	WNODE_FLAG_ALL_DATA              = 0x00000001
	WNODE_FLAG_SINGLE_INSTANCE       = 0x00000002
	WNODE_FLAG_SINGLE_ITEM           = 0x00000004
	WNODE_FLAG_EVENT_ITEM            = 0x00000008
	WNODE_FLAG_FIXED_INSTANCE_SIZE   = 0x00000010
	WNODE_FLAG_TOO_SMALL             = 0x00000020
	WNODE_FLAG_INSTANCES_SAME        = 0x00000040
	WNODE_FLAG_STATIC_INSTANCE_NAMES = 0x00000080
	WNODE_FLAG_INTERNAL              = 0x00000100
	WNODE_FLAG_USE_TIMESTAMP         = 0x00000200
	WNODE_FLAG_PERSIST_EVENT         = 0x00000400
	WNODE_FLAG_EVENT_REFERENCE       = 0x00002000
	WNODE_FLAG_ANSI_INSTANCENAwin32  = 0x00040000
	WNODE_FLAG_USE_GUID_PTR          = 0x00080000
	WNODE_FLAG_USE_MOF_PTR           = 0x00100000
	WNODE_FLAG_NO_HEADER             = 0x00200000
	WNODE_FLAG_SEND_DATA_BLOCK       = 0x00400000
	WNODE_FLAG_SEVERITY_MASK         = 0xff000000
)

const (
	EVENT_TRACE_TYPE_INFO           = 0x00
	EVENT_TRACE_TYPE_START          = 0x01
	EVENT_TRACE_TYPE_END            = 0x02
	EVENT_TRACE_TYPE_STOP           = 0x02
	EVENT_TRACE_TYPE_DC_START       = 0x03
	EVENT_TRACE_TYPE_DC_END         = 0x04
	EVENT_TRACE_TYPE_EXTENSION      = 0x05
	EVENT_TRACE_TYPE_REPLY          = 0x06
	EVENT_TRACE_TYPE_DEQUEUE        = 0x07
	EVENT_TRACE_TYPE_RESUME         = 0x07
	EVENT_TRACE_TYPE_CHECKPOINT     = 0x08
	EVENT_TRACE_TYPE_SUSPEND        = 0x08
	EVENT_TRACE_TYPE_WINEVT_SEND    = 0x09
	EVENT_TRACE_TYPE_WINEVT_RECEIVE = 0xf0

	EVENT_TRACE_TYPE_LOAD = 0x0a

	EVENT_TRACE_TYPE_IO_READ       = 0x0a
	EVENT_TRACE_TYPE_IO_WRITE      = 0x0b
	EVENT_TRACE_TYPE_IO_READ_INIT  = 0x0c
	EVENT_TRACE_TYPE_IO_WRITE_INIT = 0x0d
	EVENT_TRACE_TYPE_IO_FLUSH      = 0x0e
	EVENT_TRACE_TYPE_IO_FLUSH_INIT = 0x0f

	EVENT_TRACE_TYPE_MM_TF  = 0x0a
	EVENT_TRACE_TYPE_MM_DZF = 0x0b
	EVENT_TRACE_TYPE_MM_COW = 0x0c
	EVENT_TRACE_TYPE_MM_GPF = 0x0d
	EVENT_TRACE_TYPE_MM_HPF = 0x0e
	EVENT_TRACE_TYPE_MM_AV  = 0x0f

	EVENT_TRACE_TYPE_SEND       = 0x0a
	EVENT_TRACE_TYPE_RECEIVE    = 0x0b
	EVENT_TRACE_TYPE_CONNECT    = 0x0c
	EVENT_TRACE_TYPE_DISCONNECT = 0x0d
	EVENT_TRACE_TYPE_RETRANSMIT = 0x0e
	EVENT_TRACE_TYPE_ACCEPT     = 0x0f
	EVENT_TRACE_TYPE_RECONNECT  = 0x10
	EVENT_TRACE_TYPE_CONNFAIL   = 0x11
	EVENT_TRACE_TYPE_COPY_TCP   = 0x12
	EVENT_TRACE_TYPE_COPY_ARP   = 0x13
	EVENT_TRACE_TYPE_ACKFULL    = 0x14
	EVENT_TRACE_TYPE_ACKPART    = 0x15
	EVENT_TRACE_TYPE_ACKDUP     = 0x16

	EVENT_TRACE_TYPE_GUIDMAP    = 0x0a
	EVENT_TRACE_TYPE_CONFIG     = 0x0b
	EVENT_TRACE_TYPE_SIDINFO    = 0x0c
	EVENT_TRACE_TYPE_SECURITY   = 0x0d
	EVENT_TRACE_TYPE_DBGID_RSDS = 0x40

	EVENT_TRACE_TYPE_REGCREATE             = 0x0a
	EVENT_TRACE_TYPE_REGOPEN               = 0x0b
	EVENT_TRACE_TYPE_REGDELETE             = 0x0c
	EVENT_TRACE_TYPE_REGQUERY              = 0x0d
	EVENT_TRACE_TYPE_REGSETVALUE           = 0x0e
	EVENT_TRACE_TYPE_REGDELETEVALUE        = 0x0f
	EVENT_TRACE_TYPE_REGQUERYVALUE         = 0x10
	EVENT_TRACE_TYPE_REGENUMERATEKEY       = 0x11
	EVENT_TRACE_TYPE_REGENUMERATEVALUEKEY  = 0x12
	EVENT_TRACE_TYPE_REGQUERYMULTIPLEVALUE = 0x13
	EVENT_TRACE_TYPE_REGSETINFORMATION     = 0x14
	EVENT_TRACE_TYPE_REGFLUSH              = 0x15
	EVENT_TRACE_TYPE_REGKCBCREATE          = 0x16
	EVENT_TRACE_TYPE_REGKCBDELETE          = 0x17
	EVENT_TRACE_TYPE_REGKCBRUNDOWNBEGIN    = 0x18
	EVENT_TRACE_TYPE_REGKCBRUNDOWNEND      = 0x19
	EVENT_TRACE_TYPE_REGVIRTUALIZE         = 0x1a
	EVENT_TRACE_TYPE_REGCLOSE              = 0x1b
	EVENT_TRACE_TYPE_REGSETSECURITY        = 0x1c
	EVENT_TRACE_TYPE_REGQUERYSECURITY      = 0x1d
	EVENT_TRACE_TYPE_REGCOMMIT             = 0x1e
	EVENT_TRACE_TYPE_REGPREPARE            = 0x1f
	EVENT_TRACE_TYPE_REGROLLBACK           = 0x20
	EVENT_TRACE_TYPE_REGMOUNTHIVE          = 0x21

	EVENT_TRACE_TYPE_CONFIG_CPU          = 0x0a
	EVENT_TRACE_TYPE_CONFIG_PHYSICALDISK = 0x0b
	EVENT_TRACE_TYPE_CONFIG_LOGICALDISK  = 0x0c
	EVENT_TRACE_TYPE_CONFIG_NIC          = 0x0d
	EVENT_TRACE_TYPE_CONFIG_VIDEO        = 0x0e
	EVENT_TRACE_TYPE_CONFIG_SERVICES     = 0x0f
	EVENT_TRACE_TYPE_CONFIG_POWER        = 0x10
	EVENT_TRACE_TYPE_CONFIG_NETINFO      = 0x11
	EVENT_TRACE_TYPE_CONFIG_OPTICALMEDIA = 0x12

	EVENT_TRACE_TYPE_CONFIG_IRQ             = 0x15
	EVENT_TRACE_TYPE_CONFIG_PNP             = 0x16
	EVENT_TRACE_TYPE_CONFIG_IDECHANNEL      = 0x17
	EVENT_TRACE_TYPE_CONFIG_NUMANODE        = 0x18
	EVENT_TRACE_TYPE_CONFIG_PLATFORM        = 0x19
	EVENT_TRACE_TYPE_CONFIG_PROCESSORGROUP  = 0x1a
	EVENT_TRACE_TYPE_CONFIG_PROCESSORNUMBER = 0x1b
	EVENT_TRACE_TYPE_CONFIG_DPI             = 0x1c

	EVENT_TRACE_TYPE_OPTICAL_IO_READ       = 0x37
	EVENT_TRACE_TYPE_OPTICAL_IO_WRITE      = 0x38
	EVENT_TRACE_TYPE_OPTICAL_IO_FLUSH      = 0x39
	EVENT_TRACE_TYPE_OPTICAL_IO_READ_INIT  = 0x3a
	EVENT_TRACE_TYPE_OPTICAL_IO_WRITE_INIT = 0x3b
	EVENT_TRACE_TYPE_OPTICAL_IO_FLUSH_INIT = 0x3c

	EVENT_TRACE_TYPE_FLT_PREOP_INIT        = 0x60
	EVENT_TRACE_TYPE_FLT_POSTOP_INIT       = 0x61
	EVENT_TRACE_TYPE_FLT_PREOP_COMPLETION  = 0x62
	EVENT_TRACE_TYPE_FLT_POSTOP_COMPLETION = 0x63
	EVENT_TRACE_TYPE_FLT_PREOP_FAILURE     = 0x64
	EVENT_TRACE_TYPE_FLT_POSTOP_FAILURE    = 0x65

	// See flag documentation here
	// https://docs.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-event_trace_properties
	EVENT_TRACE_FLAG_PROCESS    = 0x00000001
	EVENT_TRACE_FLAG_THREAD     = 0x00000002
	EVENT_TRACE_FLAG_IMAGE_LOAD = 0x00000004

	EVENT_TRACE_FLAG_DISK_IO      = 0x00000100
	EVENT_TRACE_FLAG_DISK_FILE_IO = 0x00000200

	EVENT_TRACE_FLAG_MEMORY_PAGE_FAULTS = 0x00001000
	EVENT_TRACE_FLAG_MEMORY_HARD_FAULTS = 0x00002000

	EVENT_TRACE_FLAG_NETWORK_TCPIP = 0x00010000

	EVENT_TRACE_FLAG_REGISTRY = 0x00020000
	EVENT_TRACE_FLAG_DBGPRINT = 0x00040000

	EVENT_TRACE_FLAG_PROCESS_COUNTERS = 0x00000008
	EVENT_TRACE_FLAG_CSWITCH          = 0x00000010
	EVENT_TRACE_FLAG_DPC              = 0x00000020
	EVENT_TRACE_FLAG_INTERRUPT        = 0x00000040
	EVENT_TRACE_FLAG_SYSTEMCALL       = 0x00000080

	EVENT_TRACE_FLAG_DISK_IO_INIT = 0x00000400
	EVENT_TRACE_FLAG_ALPC         = 0x00100000
	EVENT_TRACE_FLAG_SPLIT_IO     = 0x00200000

	EVENT_TRACE_FLAG_DRIVER       = 0x00800000
	EVENT_TRACE_FLAG_PROFILE      = 0x01000000
	EVENT_TRACE_FLAG_FILE_IO      = 0x02000000
	EVENT_TRACE_FLAG_FILE_IO_INIT = 0x04000000

	EVENT_TRACE_FLAG_DISPATCHER    = 0x00000800
	EVENT_TRACE_FLAG_VIRTUAL_ALLOC = 0x00004000

	EVENT_TRACE_FLAG_VAMAP        = 0x00008000
	EVENT_TRACE_FLAG_NO_SYSCONFIG = 0x10000000

	EVENT_TRACE_FLAG_EXTENSION      = 0x80000000
	EVENT_TRACE_FLAG_FORWARD_WMI    = 0x40000000
	EVENT_TRACE_FLAG_ENABLE_RESERVE = 0x20000000

	// File Modes
	EVENT_TRACE_FILE_MODE_NONE       = 0x00000000
	EVENT_TRACE_FILE_MODE_SEQUENTIAL = 0x00000001
	EVENT_TRACE_FILE_MODE_CIRCULAR   = 0x00000002
	EVENT_TRACE_FILE_MODE_APPEND     = 0x00000004

	EVENT_TRACE_REAL_TIME_MODE       = 0x00000100
	EVENT_TRACE_DELAY_OPEN_FILE_MODE = 0x00000200
	EVENT_TRACE_BUFFERING_MODE       = 0x00000400
	EVENT_TRACE_PRIVATE_LOGGER_MODE  = 0x00000800
	EVENT_TRACE_ADD_HEADER_MODE      = 0x00001000

	EVENT_TRACE_USE_GLOBAL_SEQUENCE = 0x00004000
	EVENT_TRACE_USE_LOCAL_SEQUENCE  = 0x00008000

	EVENT_TRACE_RELOG_MODE = 0x00010000

	EVENT_TRACE_USE_PAGED_MEMORY = 0x01000000

	EVENT_TRACE_FILE_MODE_NEWFILE     = 0x00000008
	EVENT_TRACE_FILE_MODE_PREALLOCATE = 0x00000020

	EVENT_TRACE_NONSTOPPABLE_MODE   = 0x00000040
	EVENT_TRACE_SECURE_MODE         = 0x00000080
	EVENT_TRACE_USE_KBYTES_FOR_SIZE = 0x00002000
	EVENT_TRACE_PRIVATE_IN_PROC     = 0x00020000
	EVENT_TRACE_MODE_RESERVED       = 0x00100000

	EVENT_TRACE_NO_PER_PROCESSOR_BUFFERING = 0x10000000

	EVENT_TRACE_SYSTEM_LOGGER_MODE         = 0x02000000
	EVENT_TRACE_ADDTO_TRIAGE_DUMP          = 0x80000000
	EVENT_TRACE_STOP_ON_HYBRID_SHUTDOWN    = 0x00400000
	EVENT_TRACE_PERSIST_ON_HYBRID_SHUTDOWN = 0x00800000

	EVENT_TRACE_CONTROL_QUERY  = 0
	EVENT_TRACE_CONTROL_STOP   = 1
	EVENT_TRACE_CONTROL_UPDATE = 2
	EVENT_TRACE_CONTROL_FLUSH  = 3

	EVENT_TRACE_USE_PROCTIME  = 0x0001
	EVENT_TRACE_USE_NOCPUTIME = 0x0002
)

const (
	EVENT_CONTROL_CODE_DISABLE_PROVIDER = 0
	EVENT_CONTROL_CODE_ENABLE_PROVIDER  = 1
	EVENT_CONTROL_CODE_CAPTURE_STATE    = 2
)

const (
	// Information levels
	TRACE_LEVEL_NONE        = 0
	TRACE_LEVEL_CRITICAL    = 1
	TRACE_LEVEL_FATAL       = 1
	TRACE_LEVEL_ERROR       = 2
	TRACE_LEVEL_WARNING     = 3
	TRACE_LEVEL_INFORMATION = 4
	TRACE_LEVEL_VERBOSE     = 5
	TRACE_LEVEL_RESERVED6   = 6
	TRACE_LEVEL_RESERVED7   = 7
	TRACE_LEVEL_RESERVED8   = 8
	TRACE_LEVEL_RESERVED9   = 9
)

const (
	PROCESS_TRACE_MODE_REAL_TIME     = 0x00000100
	PROCESS_TRACE_MODE_RAW_TIMESTAMP = 0x00001000
	PROCESS_TRACE_MODE_EVENT_RECORD  = 0x10000000
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

const (
	EVENT_HEADER_PROPERTY_XML             = 0x0001
	EVENT_HEADER_PROPERTY_FORWARDED_XML   = 0x0002
	EVENT_HEADER_PROPERTY_LEGACY_EVENTLOG = 0x0004
)

const (
	EVENT_HEADER_EXT_TYPE_RELATED_ACTIVITYID = 0x0001
	EVENT_HEADER_EXT_TYPE_SID                = 0x0002
	EVENT_HEADER_EXT_TYPE_TS_ID              = 0x0003
	EVENT_HEADER_EXT_TYPE_INSTANCE_INFO      = 0x0004
	EVENT_HEADER_EXT_TYPE_STACK_TRACE32      = 0x0005
	EVENT_HEADER_EXT_TYPE_STACK_TRACE64      = 0x0006
	EVENT_HEADER_EXT_TYPE_PEBS_INDEX         = 0x0007
	EVENT_HEADER_EXT_TYPE_PMC_COUNTERS       = 0x0008
	EVENT_HEADER_EXT_TYPE_PSM_KEY            = 0x0009
	EVENT_HEADER_EXT_TYPE_EVENT_KEY          = 0x000A
	EVENT_HEADER_EXT_TYPE_EVENT_SCHEMA_TL    = 0x000B
	EVENT_HEADER_EXT_TYPE_PROV_TRAITS        = 0x000C
	EVENT_HEADER_EXT_TYPE_PROCESS_START_KEY  = 0x000D
	EVENT_HEADER_EXT_TYPE_CONTROL_GUID       = 0x000E
	EVENT_HEADER_EXT_TYPE_QPC_DELTA          = 0x000F
	EVENT_HEADER_EXT_TYPE_CONTAINER_ID       = 0x0010
	EVENT_HEADER_EXT_TYPE_STACK_KEY32        = 0x0011
	EVENT_HEADER_EXT_TYPE_STACK_KEY64        = 0x0012
	EVENT_HEADER_EXT_TYPE_MAX                = 0x0013
)

//////////////////////////////////////////////////////////////////

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
	ProviderId    uint32
	Union1        uint64
	Union2        int64
	Guid          GUID
	ClientContext uint32
	Flags         uint32
}

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

func NewEventTraceSessionProperties(sessionName string) (*EventTraceProperties, uint32) {
	size := ((len(sessionName) + 1) * 2) + int(unsafe.Sizeof(EventTraceProperties{}))
	s := make([]byte, size)
	return (*EventTraceProperties)(unsafe.Pointer(&s[0])), uint32(size)
}

func NewRealTimeEventTraceSessionProperties(logSessionName string) *EventTraceProperties {
	sessionProperties, size := NewEventTraceSessionProperties(logSessionName)

	// Necessary fields for SessionProperties struct
	sessionProperties.Wnode.BufferSize = size // this is optimized by ETWframework
	sessionProperties.Wnode.Guid = GUID{}     //To set
	sessionProperties.Wnode.ClientContext = 1 // QPC
	sessionProperties.Wnode.Flags = WNODE_FLAG_ALL_DATA
	sessionProperties.LogFileMode = EVENT_TRACE_REAL_TIME_MODE
	sessionProperties.LogFileNameOffset = 0
	// ETW event can be up to 64KB size so if the buffer size is not at least
	// big enough to contain such an event, the event will be lost
	// source: https://docs.microsoft.com/en-us/message-analyzer/specifying-advanced-etw-session-configuration-settings
	sessionProperties.BufferSize = 64
	sessionProperties.LoggerNameOffset = uint32(unsafe.Sizeof(EventTraceProperties{}))

	return sessionProperties
}

/*
typedef struct _ENABLE_TRACE_PARAMETERS {
  ULONG                    Version;
  ULONG                    EnableProperty;
  ULONG                    ControlFlags;
  GUID                     SourceId;
  PEVENT_FILTER_DESCRIPTOR EnableFilterDesc;
  ULONG                    FilterDescCount;
} ENABLE_TRACE_PARAMETERS, *PENABLE_TRACE_PARAMETERS;
*/

type EnableTraceParameters struct {
	Version          uint32
	EnableProperty   uint32
	ControlFlags     uint32
	SourceId         GUID
	EnableFilterDesc *EventFilterDescriptor
	FilterDescCount  uint32
}

const (
	EVENT_ENABLE_PROPERTY_SID                       = 0x00000001
	EVENT_ENABLE_PROPERTY_TS_ID                     = 0x00000002
	EVENT_ENABLE_PROPERTY_STACK_TRACE               = 0x00000004
	EVENT_ENABLE_PROPERTY_PSM_KEY                   = 0x00000008
	EVENT_ENABLE_PROPERTY_IGNORE_KEYWORD_0          = 0x00000010
	EVENT_ENABLE_PROPERTY_PROVIDER_GROUP            = 0x00000020
	EVENT_ENABLE_PROPERTY_ENABLE_KEYWORD_0          = 0x00000040
	EVENT_ENABLE_PROPERTY_PROCESS_START_KEY         = 0x00000080
	EVENT_ENABLE_PROPERTY_EVENT_KEY                 = 0x00000100
	EVENT_ENABLE_PROPERTY_EXCLUDE_INPRIVATE         = 0x00000200
	EVENT_ENABLE_PROPERTY_ENABLE_SILOS              = 0x00000400
	EVENT_ENABLE_PROPERTY_SOURCE_CONTAINER_TRACKING = 0x00000800
)

const (
	EVENT_FILTER_TYPE_NONE               = 0x00000000
	EVENT_FILTER_TYPE_SCHEMATIZED        = 0x80000000 // Provider-side.
	EVENT_FILTER_TYPE_SYSTEM_FLAGS       = 0x80000001 // Internal use only.
	EVENT_FILTER_TYPE_TRACEHANDLE        = 0x80000002 // Initiate rundown.
	EVENT_FILTER_TYPE_PID                = 0x80000004 // Process ID.
	EVENT_FILTER_TYPE_EXECUTABLE_NAME    = 0x80000008 // EXE file name.
	EVENT_FILTER_TYPE_PACKAGE_ID         = 0x80000010 // Package ID.
	EVENT_FILTER_TYPE_PACKAGE_APP_ID     = 0x80000020 // Package Relative App Id (PRAID).
	EVENT_FILTER_TYPE_PAYLOAD            = 0x80000100 // TDH payload filter.
	EVENT_FILTER_TYPE_EVENT_ID           = 0x80000200 // Event IDs.
	EVENT_FILTER_TYPE_EVENT_NAME         = 0x80000400 // Event name (TraceLogging only).
	EVENT_FILTER_TYPE_STACKWALK          = 0x80001000 // Event IDs for stack.
	EVENT_FILTER_TYPE_STACKWALK_NAME     = 0x80002000 // Event name for stack (TraceLogging only).
	EVENT_FILTER_TYPE_STACKWALK_LEVEL_KW = 0x80004000 // Filter stack collection by level and keyword.
	EVENT_FILTER_TYPE_CONTAINER          = 0x80008000 // Filter by Container ID.
)

const (
	MAX_EVENT_FILTER_DATA_SIZE = 1024

	MAX_EVENT_FILTER_EVENT_ID_COUNT = 64
)

/*
EVENT_FILTER_EVENT_ID is used to pass EventId filter for
stack walk filters.

	typedef struct _EVENT_FILTER_EVENT_ID {
	    BOOLEAN FilterIn;
	    UCHAR Reserved;
	    USHORT Count;
	    USHORT Events[ANYSIZE_ARRAY];
	} EVENT_FILTER_EVENT_ID, *PEVENT_FILTER_EVENT_ID;
*/
type EventFilterEventID struct {
	FilterIn uint8
	Reserved uint8
	Count    uint16
	// it is easier to implement in Go with a fixed array size
	Events [1]uint16
}

func AllocEventFilterEventID(filter []uint16) (f *EventFilterEventID) {
	count := uint16(len(filter))
	size := max(4+len(filter)*2, int(unsafe.Sizeof(EventFilterEventID{})))
	buf := make([]byte, size)

	// buf[0] should always be valid
	f = (*EventFilterEventID)(unsafe.Pointer(&buf[0]))
	eid := unsafe.Pointer((&f.Events[0]))
	for i := 0; i < len(filter); i++ {
		*((*uint16)(eid)) = filter[i]
		eid = unsafe.Add(eid, 2)
	}
	f.Count = count
	return
}

func (e *EventFilterEventID) Size() int {
	return 4 + int(e.Count)*2
}

/*
typedef struct _EVENT_FILTER_DESCRIPTOR {
  ULONGLONG Ptr;
  ULONG     Size;
  ULONG     Type;
} EVENT_FILTER_DESCRIPTOR, *PEVENT_FILTER_DESCRIPTOR;
*/
// sizeof: 0x10 (OK)
type EventFilterDescriptor struct {
	Ptr  uint64
	Size uint32
	Type uint32
}

/*
	typedef struct _FILETIME {
	  DWORD dwLowDateTime;
	  DWORD dwHighDateTime;
	} FILETIME, *PFILETIME, *LPFILETIME;
*/
type FileTime struct {
	dwLowDateTime  uint32
	dwHighDateTime uint32
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

type EventCallback func(*EventTrace)
type EventRecordCallback func(*EventRecord) uintptr
type EventTraceBufferCallback func(*EventTraceLogfile) uint32

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

/*
Un-used uncomment if necessary
func (e *EventRecord) pointer() uintptr {
	return (uintptr)(unsafe.Pointer(e))
}

func (e *EventRecord) pointerOffset(offset uintptr) uintptr {
	return e.pointer() + offset
}
*/

func (e *EventRecord) ExtendedDataItem(i uint16) *EventHeaderExtendedDataItem {
	if i < e.ExtendedDataCount {
		return (*EventHeaderExtendedDataItem)(unsafe.Pointer((uintptr(unsafe.Pointer(e.ExtendedData)) + (uintptr(i) * unsafe.Sizeof(EventHeaderExtendedDataItem{})))))
	}
	panic("out of bound extended data item")
}

func (e *EventRecord) RelatedActivityID() string {
	for i := uint16(0); i < e.ExtendedDataCount; i++ {
		item := e.ExtendedDataItem(i)
		if item.ExtType == EVENT_HEADER_EXT_TYPE_RELATED_ACTIVITYID {
			g := (*GUID)(unsafe.Pointer(item.DataPtr))
			return g.String()
		}
	}
	return nullGUIDStr
}

func (e *EventRecord) GetEventInformation() (tei *TraceEventInfo, err error) {
	bufferSize := uint32(0)
	if err = TdhGetEventInformation(e, 0, nil, nil, &bufferSize); err == syscall.ERROR_INSUFFICIENT_BUFFER {
		// don't know how this would behave
		buff := make([]byte, bufferSize)
		tei = ((*TraceEventInfo)(unsafe.Pointer(&buff[0])))
		err = TdhGetEventInformation(e, 0, nil, tei, &bufferSize)
	}
	return
}

/*
// Both MOF-based events and manifest-based events can specify name/value maps. The
// map values can be integer values or bit values. If the property specifies a value
// map, get the map.

DWORD GetMapInfo(PEVENT_RECORD pEvent, LPWSTR pMapName, DWORD DecodingSource, PEVENT_MAP_INFO & pMapInfo)
{
    DWORD status = ERROR_SUCCESS;
    DWORD MapSize = 0;

    // Retrieve the required buffer size for the map info.

    status = TdhGetEventMapInformation(pEvent, pMapName, pMapInfo, &MapSize);

    if (ERROR_INSUFFICIENT_BUFFER == status)
    {
        pMapInfo = (PEVENT_MAP_INFO) malloc(MapSize);
        if (pMapInfo == NULL)
        {
            wprintf(L"Failed to allocate memory for map info (size=%lu).\n", MapSize);
            status = ERROR_OUTOFMEMORY;
            goto cleanup;
        }

        // Retrieve the map info.

        status = TdhGetEventMapInformation(pEvent, pMapName, pMapInfo, &MapSize);
    }

    if (ERROR_SUCCESS == status)
    {
        if (DecodingSourceXMLFile == DecodingSource)
        {
            RemoveTrailingSpace(pMapInfo);
        }
    }
    else
    {
        if  (ERROR_NOT_FOUND == status)
        {
            status = ERROR_SUCCESS; // This case is okay.
        }
        else
        {
            wprintf(L"TdhGetEventMapInformation failed with 0x%x.\n", status);
        }
    }

cleanup:

    return status;
}
*/

func (e *EventRecord) GetMapInfo(pMapName *uint16, decodingSource uint32) (pMapInfo *EventMapInfo, err error) {
	mapSize := uint32(64)
	buff := make([]byte, mapSize)
	pMapInfo = ((*EventMapInfo)(unsafe.Pointer(&buff[0])))
	err = TdhGetEventMapInformation(e, pMapName, pMapInfo, &mapSize)

	if err == syscall.ERROR_INSUFFICIENT_BUFFER {
		buff := make([]byte, mapSize)
		pMapInfo = ((*EventMapInfo)(unsafe.Pointer(&buff[0])))
		err = TdhGetEventMapInformation(e, pMapName, pMapInfo, &mapSize)
	}

	if err == nil {
		if DecodingSource(decodingSource) == DecodingSourceXMLFile {
			pMapInfo.RemoveTrailingSpace()
		}
	}

	if err == syscall.ERROR_NOT_FOUND {
		err = nil
	}
	return
}

func (e *EventRecord) PointerSize() uint32 {
	if e.EventHeader.Flags&EVENT_HEADER_FLAG_32_BIT_HEADER == EVENT_HEADER_FLAG_32_BIT_HEADER {
		return 4
	}
	return 8
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
	ProviderId      GUID
	EventDescriptor EventDescriptor
	Time            int64
	ActivityId      GUID
}

func (e *EventHeader) UTCTimeStamp() time.Time {
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
	ParentGuid       GUID
	MofData          uintptr
	MofLength        uint32
	UnionCtx         uint32
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

const (
	SDDL_REVISION_1 = 1
)

type SecurityDescriptorControl uint32

const (
	SE_OWNER_DEFAULTED       = SecurityDescriptorControl(0x0001)
	SE_GROUP_DEFAULTED       = SecurityDescriptorControl(0x0002)
	SE_DACL_PRESENT          = SecurityDescriptorControl(0x0004)
	SE_DACL_DEFAULTED        = SecurityDescriptorControl(0x0008)
	SE_SACL_PRESENT          = SecurityDescriptorControl(0x0010)
	SE_SACL_DEFAULTED        = SecurityDescriptorControl(0x0020)
	SE_DACL_AUTO_INHERIT_REQ = SecurityDescriptorControl(0x0100)
	SE_SACL_AUTO_INHERIT_REQ = SecurityDescriptorControl(0x0200)
	SE_DACL_AUTO_INHERITED   = SecurityDescriptorControl(0x0400)
	SE_SACL_AUTO_INHERITED   = SecurityDescriptorControl(0x0800)
	SE_DACL_PROTECTED        = SecurityDescriptorControl(0x1000)
	SE_SACL_PROTECTED        = SecurityDescriptorControl(0x2000)
	SE_RM_CONTROL_VALID      = SecurityDescriptorControl(0x4000)
	SE_SELF_RELATIVE         = SecurityDescriptorControl(0x8000)
)

type SecurityInformation uint32

const (
	OWNER_SECURITY_INFORMATION            = SecurityInformation(0x00000001)
	GROUP_SECURITY_INFORMATION            = SecurityInformation(0x00000002)
	DACL_SECURITY_INFORMATION             = SecurityInformation(0x00000004)
	SACL_SECURITY_INFORMATION             = SecurityInformation(0x00000008)
	LABEL_SECURITY_INFORMATION            = SecurityInformation(0x00000010)
	ATTRIBUTE_SECURITY_INFORMATION        = SecurityInformation(0x00000020)
	SCOPE_SECURITY_INFORMATION            = SecurityInformation(0x00000040)
	BACKUP_SECURITY_INFORMATION           = SecurityInformation(0x00010000)
	PROTECTED_DACL_SECURITY_INFORMATION   = SecurityInformation(0x80000000)
	PROTECTED_SACL_SECURITY_INFORMATION   = SecurityInformation(0x40000000)
	UNPROTECTED_DACL_SECURITY_INFORMATION = SecurityInformation(0x20000000)
	UNPROTECTED_SACL_SECURITY_INFORMATION = SecurityInformation(0x10000000)
)

/*
//0x6 bytes (sizeof)
struct _SID_IDENTIFIER_AUTHORITY
{
    UCHAR Value[6];                                                         //0x0
};
*/

type SidIdentifierAuthority struct {
	Value [6]uint8
}

/*
//0xc bytes (sizeof)
struct _SID
{
    UCHAR Revision;                                                         //0x0
    UCHAR SubAuthorityCount;                                                //0x1
    struct _SID_IDENTIFIER_AUTHORITY IdentifierAuthority;                   //0x2
    ULONG SubAuthority[1];                                                  //0x8
};
*/

type SID struct {
	Revision            uint8
	SubAuthorityCount   uint8
	IdentifierAuthority SidIdentifierAuthority
	SubAuthority        [1]uint32
}

/*
//0x8 bytes (sizeof)
struct _ACL
{
    UCHAR AclRevision;                                                      //0x0
    UCHAR Sbz1;                                                             //0x1
    USHORT AclSize;                                                         //0x2
    USHORT AceCount;                                                        //0x4
    USHORT Sbz2;                                                            //0x6
};
*/

type ACL struct {
	AclRevision uint8
	Sbz1        uint8
	AclSize     uint16
	AceCount    uint16
	Sbz2        uint16
}

/*
typedef struct _SECURITY_DESCRIPTOR {
  BYTE                        Revision;
  BYTE                        Sbz1;
  SECURITY_DESCRIPTOR_CONTROL Control;
  PSID                        Owner;
  PSID                        Group;
  PACL                        Sacl;
  PACL                        Dacl;
} SECURITY_DESCRIPTOR, *PISECURITY_DESCRIPTOR;
*/

type SecurityDescriptor struct {
	Revision byte
	Sbz1     byte
	Control  SecurityDescriptorControl
	Owner    *SID
	Group    *SID
	Sacl     *ACL
	Dacl     *ACL
}

/*
typedef enum {
    EventSecuritySetDACL,
    EventSecuritySetSACL,
    EventSecurityAddDACL,
    EventSecurityAddSACL,
    EventSecurityMax
  } EVENTSECURITYOPERATION;
*/

type EventSecurityOperation uint32

const (
	EVENT_SECURITY_SET_DACL = EventSecurityOperation(0)
	EVENT_SECURITY_SET_SACL = EventSecurityOperation(1)
	EVENT_SECURITY_ADD_DACL = EventSecurityOperation(2)
	EVENT_SECURITY_ADD_SACL = EventSecurityOperation(3)
	EVENT_SECURITY_MAX      = EventSecurityOperation(4)
)

// Permissions for EventAccessControl API
const (
	TRACELOG_CREATE_REALTIME      = 0x0020
	TRACELOG_CREATE_ONDISK        = 0x0040
	TRACELOG_GUID_ENABLE          = 0x0080
	TRACELOG_ACCESS_KERNEL_LOGGER = 0x0100
	TRACELOG_CREATE_INPROC        = 0x0200
	TRACELOG_LOG_EVENT            = 0x0200
	TRACELOG_ACCESS_REALTIME      = 0x0400
	TRACELOG_REGISTER_GUIDS       = 0x0800
	TRACELOG_ALL                  = TRACELOG_CREATE_REALTIME |
		TRACELOG_CREATE_ONDISK |
		TRACELOG_GUID_ENABLE |
		TRACELOG_LOG_EVENT |
		TRACELOG_ACCESS_REALTIME |
		TRACELOG_REGISTER_GUIDS
)
