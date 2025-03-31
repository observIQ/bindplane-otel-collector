//go:build windows
// +build windows

package etw

import (
	"fmt"
	"strings"
	"unsafe"
)

/*
typedef struct _TDH_CONTEXT {
  ULONGLONG        ParameterValue;
  TDH_CONTEXT_TYPE ParameterType;
  ULONG            ParameterSize;
} TDH_CONTEXT;
*/

type TdhContext struct {
	ParameterValue uint32
	ParameterType  TdhContextType
	ParameterSize  uint32
}

/*
typedef enum _TDH_CONTEXT_TYPE {
  TDH_CONTEXT_WPP_TMFFILE,
  TDH_CONTEXT_WPP_TMFSEARCHPATH,
  TDH_CONTEXT_WPP_GMT,
  TDH_CONTEXT_POINTERSIZE,
  TDH_CONTEXT_PDB_PATH,
  TDH_CONTEXT_MAXIMUM
} TDH_CONTEXT_TYPE;
*/

type TdhContextType int32

const (
	TDH_CONTEXT_WPP_TMFFILE       = TdhContextType(0)
	TDH_CONTEXT_WPP_TMFSEARCHPATH = TdhContextType(1)
	TDH_CONTEXT_WPP_GMT           = TdhContextType(2)
	TDH_CONTEXT_POINTERSIZE       = TdhContextType(3)
	TDH_CONTEXT_MAXIMUM           = TdhContextType(4)
)

/*
typedef struct _PROPERTY_DATA_DESCRIPTOR {
  ULONGLONG PropertyName;
  ULONG     ArrayIndex;
  ULONG     Reserved;
} PROPERTY_DATA_DESCRIPTOR;
*/

type PropertyDataDescriptor struct {
	PropertyName uint64
	ArrayIndex   uint32
	Reserved     uint32
}

/*
typedef struct _PROVIDER_FIELD_INFOARRAY {
  ULONG               NumberOfElements;
  EVENT_FIELD_TYPE    FieldType;
  PROVIDER_FIELD_INFO FieldInfoArray[ANYSIZE_ARRAY];
} PROVIDER_FIELD_INFOARRAY;
*/

type ProviderFieldInfoArray struct {
	NumberOfElements uint32
	FieldType        EventFieldType // This field is initially an enum so I guess it has the size of an int
	FieldInfoArray   [1]ProviderFieldInfo
}

/*
typedef struct _PROVIDER_FIELD_INFO {
  ULONG     NameOffset;
  ULONG     DescriptionOffset;
  ULONGLONG Value;
} PROVIDER_FIELD_INFO;
*/
type ProviderFieldInfo struct {
	NameOffset        uint32
	DescriptionOffset uint32
	Value             uint64
}

/*
typedef enum _EVENT_FIELD_TYPE {
  EventKeywordInformation   = 0,
  EventLevelInformation     = 1,
  EventChannelInformation   = 2,
  EventTaskInformation      = 3,
  EventOpcodeInformation    = 4,
  EventInformationMax       = 5
} EVENT_FIELD_TYPE;
*/

type EventFieldType int32

const (
	EventKeywordInformation = EventFieldType(0)
	EventLevelInformation   = EventFieldType(1)
	EventChannelInformation = EventFieldType(2)
	EventTaskInformation    = EventFieldType(3)
	EventOpcodeInformation  = EventFieldType(4)
	EventInformationMax     = EventFieldType(5)
)

/*
typedef struct _PROVIDER_ENUMERATION_INFO {
  ULONG               NumberOfProviders;
  ULONG               Reserved;
  TRACE_PROVIDER_INFO TraceProviderInfoArray[ANYSIZE_ARRAY];
} PROVIDER_ENUMERATION_INFO;
*/

type ProviderEnumerationInfo struct {
	NumberOfProviders      uint32
	Reserved               uint32
	TraceProviderInfoArray [1]TraceProviderInfo
}

/*
typedef struct _TRACE_PROVIDER_INFO {
  GUID  ProviderGuid;
  ULONG SchemaSource;
  ULONG ProviderNameOffset;
} TRACE_PROVIDER_INFO;
*/

type TraceProviderInfo struct {
	ProviderGuid       GUID
	SchemaSource       uint32
	ProviderNameOffset uint32
}

/*
typedef struct _TRACE_EVENT_INFO {
  GUID                ProviderGuid;
  GUID                EventGuid;
  EVENT_DESCRIPTOR    EventDescriptor;
  DECODING_SOURCE     DecodingSource;
  ULONG               ProviderNameOffset;
  ULONG               LevelNameOffset;
  ULONG               ChannelNameOffset;
  ULONG               KeywordsNameOffset;
  ULONG               TaskNameOffset;
  ULONG               OpcodeNameOffset;
  ULONG               EventMessageOffset;
  ULONG               ProviderMessageOffset;
  ULONG               BinaryXMLOffset;
  ULONG               BinaryXMLSize;
  union {
    ULONG EventNameOffset;
    ULONG ActivityIDNameOffset;
  };
  union {
    ULONG EventAttributesOffset;
    ULONG RelatedActivityIDNameOffset;
  };
  ULONG               PropertyCount;
  ULONG               TopLevelPropertyCount;
  union {
    TEMPLATE_FLAGS Flags;
    struct {
      ULONG Reserved : 4;
      ULONG Tags : 28;
    };
  };
  EVENT_PROPERTY_INFO EventPropertyInfoArray[ANYSIZE_ARRAY];
} TRACE_EVENT_INFO;

typedef struct _TRACE_EVENT_INFO {
  GUID                ProviderGuid;
  GUID                EventGuid;
  EVENT_DESCRIPTOR    EventDescriptor;
  DECODING_SOURCE     DecodingSource;
  ULONG               ProviderNameOffset;
  ULONG               LevelNameOffset;
  ULONG               ChannelNameOffset;
  ULONG               KeywordsNameOffset;
  ULONG               TaskNameOffset;
  ULONG               OpcodeNameOffset;
  ULONG               EventMessageOffset;
  ULONG               ProviderMessageOffset;
  ULONG               BinaryXMLOffset;
  ULONG               BinaryXMLSize;
  ULONG               ActivityIDNameOffset;
  ULONG               RelatedActivityIDNameOffset;
  ULONG               PropertyCount;
  ULONG               TopLevelPropertyCount;
  TEMPLATE_FLAGS      Flags;
  EVENT_PROPERTY_INFO EventPropertyInfoArray[ANYSIZE_ARRAY];
} TRACE_EVENT_INFO, *PTRACE_EVENT_INFO;

*/

type TraceEventInfo struct {
	ProviderGUID                GUID
	EventGUID                   GUID
	EventDescriptor             EventDescriptor
	DecodingSource              DecodingSource
	ProviderNameOffset          uint32
	LevelNameOffset             uint32
	ChannelNameOffset           uint32
	KeywordsNameOffset          uint32
	TaskNameOffset              uint32
	OpcodeNameOffset            uint32
	EventMessageOffset          uint32
	ProviderMessageOffset       uint32
	BinaryXMLOffset             uint32
	BinaryXMLSize               uint32
	ActivityIDNameOffset        uint32
	RelatedActivityIDNameOffset uint32
	PropertyCount               uint32
	TopLevelPropertyCount       uint32
	Flags                       TemplateFlags
	EventPropertyInfoArray      [1]EventPropertyInfo
}

func (t *TraceEventInfo) pointer() uintptr {
	return uintptr(unsafe.Pointer(t))
}

func (t *TraceEventInfo) pointerOffset(offset uintptr) uintptr {
	return t.pointer() + offset
}

func (t *TraceEventInfo) stringAt(offset uintptr) string {
	if offset > 0 {
		return UTF16AtOffsetToString(t.pointer(), offset)
	}
	return ""
}

func (t *TraceEventInfo) cleanStringAt(offset uintptr) string {
	if offset > 0 {
		return strings.Trim(t.stringAt(offset), " ")
	}
	return ""
}

// Seems to be always empty
func (t *TraceEventInfo) EventMessage() string {
	return t.cleanStringAt(uintptr(t.EventMessageOffset))
}

func (t *TraceEventInfo) ProviderName() string {
	return t.cleanStringAt(uintptr(t.ProviderNameOffset))
}

func (t *TraceEventInfo) TaskName() string {
	return t.cleanStringAt(uintptr(t.TaskNameOffset))
}

func (t *TraceEventInfo) LevelName() string {
	return t.cleanStringAt(uintptr(t.LevelNameOffset))
}

func (t *TraceEventInfo) OpcodeName() string {
	return t.cleanStringAt(uintptr(t.OpcodeNameOffset))
}

func (t *TraceEventInfo) KeywordName() string {
	return t.cleanStringAt(uintptr(t.KeywordsNameOffset))
}

func (t *TraceEventInfo) ChannelName() string {
	return t.cleanStringAt(uintptr(t.ChannelNameOffset))
}

// Seems to be always empty
func (t *TraceEventInfo) ActivityIDName() string {
	return t.stringAt(uintptr(t.ActivityIDNameOffset))
}

// Seems to be always empty
func (t *TraceEventInfo) RelatedActivityIDName() string {
	return t.stringAt(uintptr(t.RelatedActivityIDNameOffset))
}

func (t *TraceEventInfo) IsMof() bool {
	return t.DecodingSource == DecodingSourceWbem
}

func (t *TraceEventInfo) IsXML() bool {
	return t.DecodingSource == DecodingSourceXMLFile
}

func (t *TraceEventInfo) EventID() uint16 {
	if t.IsXML() {
		return t.EventDescriptor.Id
	} else if t.IsMof() {
		if c, ok := MofClassMapping[t.EventGUID.Data1]; ok {
			return c.BaseId + uint16(t.EventDescriptor.Opcode)
		}
	}
	// not meaningful, cannot be used to identify event
	return 0
}

func (t *TraceEventInfo) GetEventPropertyInfoAt(i uint32) *EventPropertyInfo {
	if i < t.PropertyCount {
		pEpi := uintptr(unsafe.Pointer(&t.EventPropertyInfoArray[0]))
		pEpi += uintptr(i) * unsafe.Sizeof(EventPropertyInfo{})
		// this line triggers checkptr
		// I guess that is because TraceInfo is variable size C
		// struct we had to hack with to make it compatible with Go
		return ((*EventPropertyInfo)(unsafe.Pointer(pEpi)))
	}
	panic(fmt.Errorf("index out of range"))
}

func (t *TraceEventInfo) PropertyNameOffset(i uint32) uintptr {
	return t.pointer() + uintptr(t.GetEventPropertyInfoAt(i).NameOffset)
}

/*
typedef enum _DECODING_SOURCE {
  DecodingSourceXMLFile   = 0,
  DecodingSourceWbem      = 1,
  DecodingSourceWPP       = 2
} DECODING_SOURCE;
*/

type DecodingSource int32

const (
	DecodingSourceXMLFile = DecodingSource(0)
	DecodingSourceWbem    = DecodingSource(1)
	DecodingSourceWPP     = DecodingSource(2)
)

/*
typedef enum _TEMPLATE_FLAGS {
  TEMPLATE_EVENT_DATA   = 1,
  TEMPLATE_USER_DATA    = 2
} TEMPLATE_FLAGS;
*/

type TemplateFlags int32

const (
	TEMPLATE_EVENT_DATA = TemplateFlags(1)
	TEMPLATE_USER_DATA  = TemplateFlags(2)
)

/*
typedef struct _EVENT_MAP_INFO {
  ULONG           NameOffset;
  MAP_FLAGS       Flag;
  ULONG           EntryCount;
  union {
    MAP_VALUETYPE MapEntryValueType;
    ULONG         FormatStringOffset;
  };
  EVENT_MAP_ENTRY MapEntryArray[ANYSIZE_ARRAY];
} EVENT_MAP_INFO;
*/

type EventMapInfo struct {
	NameOffset    uint32
	Flag          MapFlags
	EntryCount    uint32
	Union         uint32 // Not sure about size of union depends on size of enum MAP_VALUETYPE
	MapEntryArray [1]EventMapEntry
}

func (e *EventMapInfo) GetEventMapEntryAt(i int) *EventMapEntry {
	if uint32(i) < e.EntryCount {
		pEmi := uintptr(unsafe.Pointer(&e.MapEntryArray[0]))
		pEmi += uintptr(i) * unsafe.Sizeof(EventMapEntry{})
		return ((*EventMapEntry)(unsafe.Pointer(pEmi)))
	}
	panic(fmt.Errorf("Index out of range"))
}

/*
// The mapped string values defined in a manifest will contain a trailing space
// in the EVENT_MAP_ENTRY structure. Replace the trailing space with a null-
// terminating character, so that the bit mapped strings are correctly formatted.

void RemoveTrailingSpace(PEVENT_MAP_INFO pMapInfo)
{
    SIZE_T ByteLength = 0;

    for (DWORD i = 0; i < pMapInfo->EntryCount; i++)
    {
        ByteLength = (wcslen((LPWSTR)((PBYTE)pMapInfo + pMapInfo->MapEntryArray[i].OutputOffset)) - 1) * 2;
        *((LPWSTR)((PBYTE)pMapInfo + (pMapInfo->MapEntryArray[i].OutputOffset + ByteLength))) = L'\0';
    }
}
*/

func (e *EventMapInfo) RemoveTrailingSpace() {
	for i := uint32(0); i < e.EntryCount; i++ {
		me := e.GetEventMapEntryAt(int(i))
		pStr := uintptr(unsafe.Pointer(e)) + uintptr(me.OutputOffset)
		byteLen := (Wcslen(((*uint16)(unsafe.Pointer(pStr)))) - 1) * 2
		*((*uint16)(unsafe.Pointer(pStr + uintptr(byteLen)))) = 0
	}
}

/*
typedef enum _MAP_FLAGS {
  EVENTMAP_INFO_FLAG_MANIFEST_VALUEMAP     = 1,
  EVENTMAP_INFO_FLAG_MANIFEST_BITMAP       = 2,
  EVENTMAP_INFO_FLAG_MANIFEST_PATTERNMAP   = 4,
  EVENTMAP_INFO_FLAG_WBEM_VALUEMAP         = 8,
  EVENTMAP_INFO_FLAG_WBEM_BITMAP           = 16,
  EVENTMAP_INFO_FLAG_WBEM_FLAG             = 32,
  EVENTMAP_INFO_FLAG_WBEM_NO_MAP           = 64
} MAP_FLAGS;
*/

type MapFlags int32

const (
	EVENTMAP_INFO_FLAG_MANIFEST_VALUEMAP   = MapFlags(1)
	EVENTMAP_INFO_FLAG_MANIFEST_BITMAP     = MapFlags(2)
	EVENTMAP_INFO_FLAG_MANIFEST_PATTERNMAP = MapFlags(4)
	EVENTMAP_INFO_FLAG_WBEM_VALUEMAP       = MapFlags(8)
	EVENTMAP_INFO_FLAG_WBEM_BITMAP         = MapFlags(16)
	EVENTMAP_INFO_FLAG_WBEM_FLAG           = MapFlags(32)
	EVENTMAP_INFO_FLAG_WBEM_NO_MAP         = MapFlags(64)
)

/*
typedef enum _MAP_VALUETYPE
{
  EVENTMAP_ENTRY_VALUETYPE_ULONG  = 0,
  EVENTMAP_ENTRY_VALUETYPE_STRING = 1
} MAP_VALUETYPE;
*/

type MapValueType int32

const (
	EVENTMAP_ENTRY_VALUETYPE_ULONG  = MapValueType(0)
	EVENTMAP_ENTRY_VALUETYPE_STRING = MapValueType(1)
)

/*
typedef struct _EVENT_MAP_ENTRY {
  ULONG OutputOffset;
  __C89_NAMELESS union {
    ULONG Value;
    ULONG InputOffset;
  };
} EVENT_MAP_ENTRY, *PEVENT_MAP_ENTRY;
*/

type EventMapEntry struct {
	OutputOffset uint32
	Union        uint32
}

type PropertyFlags int32

const (
	PropertyStruct           = PropertyFlags(0x1)
	PropertyParamLength      = PropertyFlags(0x2)
	PropertyParamCount       = PropertyFlags(0x4)
	PropertyWBEMXmlFragment  = PropertyFlags(0x8)
	PropertyParamFixedLength = PropertyFlags(0x10)
)

/*
typedef struct _EVENT_PROPERTY_INFO {
  PROPERTY_FLAGS Flags;
  ULONG          NameOffset;
  union {
    struct {
      USHORT InType;
      USHORT OutType;
      ULONG  MapNameOffset;
    } nonStructType;
    struct {
      USHORT StructStartIndex;
      USHORT NumOfStructMembers;
      ULONG  padding;
    } structType;
    struct {
      USHORT InType;
      USHORT OutType;
      ULONG  CustomSchemaOffset;
    } customSchemaType;
  };
  union {
    USHORT count;
    USHORT countPropertyIndex;
  };
  union {
    USHORT length;
    USHORT lengthPropertyIndex;
  };
  union {
    ULONG Reserved;
    struct {
      ULONG Tags : 28;
    };
  };
} EVENT_PROPERTY_INFO;
*/

type EventPropertyInfo struct {
	Flags      PropertyFlags
	NameOffset uint32
	TypeUnion  struct {
		u1 uint16
		u2 uint16
		u3 uint32
	}
	CountUnion  uint16
	LengthUnion uint16
	ResTagUnion uint32
}

func (i *EventPropertyInfo) InType() uint16 {
	return i.TypeUnion.u1
}
func (i *EventPropertyInfo) StructStartIndex() uint16 {
	return i.InType()
}

func (i *EventPropertyInfo) OutType() uint16 {
	return i.TypeUnion.u2
}

func (i *EventPropertyInfo) NumOfStructMembers() uint16 {
	return i.OutType()
}

func (i *EventPropertyInfo) MapNameOffset() uint32 {
	return i.CustomSchemaOffset()
}

func (i *EventPropertyInfo) CustomSchemaOffset() uint32 {
	return i.TypeUnion.u3
}

func (i *EventPropertyInfo) Count() uint16 {
	return i.CountUnion
}

func (i *EventPropertyInfo) CountPropertyIndex() uint16 {
	return i.CountUnion
}

func (i *EventPropertyInfo) LengthPropertyIndex() uint16 {
	return i.LengthUnion
}

func (i *EventPropertyInfo) Length() uint16 {
	return i.LengthUnion
}

type TdhInType uint32

// found info there: https://github.com/microsoft/ETW2JSON/blob/6721e0438733b316d316d36c488166853a05f836/Deserializer/Tdh.cs
const (
	TdhInTypeNull = TdhInType(iota)
	TdhInTypeUnicodestring
	TdhInTypeAnsistring
	TdhInTypeInt8
	TdhInTypeUint8
	TdhInTypeInt16
	TdhInTypeUint16
	TdhInTypeInt32
	TdhInTypeUint32
	TdhInTypeInt64
	TdhInTypeUint64
	TdhInTypeFloat
	TdhInTypeDouble
	TdhInTypeBoolean
	TdhInTypeBinary
	TdhInTypeGUID
	TdhInTypePointer
	TdhInTypeFiletime
	TdhInTypeSystemtime
	TdhInTypeSid
	TdhInTypeHexint32
	TdhInTypeHexint64 // End of winmeta types
)

const (
	TdhInTypeCountedstring = TdhInType(iota + 300) // Start of TDH intypes for WBEM.
	TdhInTypeCountedansistring
	TdhInTypeReversedcountedstring
	TdhInTypeReversedcountedansistring
	TdhInTypeNonnullterminatedstring
	TdhInTypeNonnullterminatedansistring
	TdhInTypeUnicodechar
	TdhInTypeAnsichar
	TdhInTypeSizet
	TdhInTypeHexdump
	TdhInTypeWbemsid
)

type TdhOutType uint32

const (
	TdhOutTypeNull = TdhOutType(iota)
	TdhOutTypeString
	TdhOutTypeDatetime
	TdhOutTypeByte
	TdhOutTypeUnsignedbyte
	TdhOutTypeShort
	TdhOutTypeUnsignedshort
	TdhOutTypeInt
	TdhOutTypeUnsignedint
	TdhOutTypeLong
	TdhOutTypeUnsignedlong
	TdhOutTypeFloat
	TdhOutTypeDouble
	TdhOutTypeBoolean
	TdhOutTypeGUID
	TdhOutTypeHexbinary
	TdhOutTypeHexint8
	TdhOutTypeHexint16
	TdhOutTypeHexint32
	TdhOutTypeHexint64
	TdhOutTypePid
	TdhOutTypeTid
	TdhOutTypePort
	TdhOutTypeIpv4
	TdhOutTypeIpv6
	TdhOutTypeSocketaddress
	TdhOutTypeCimdatetime
	TdhOutTypeEtwtime
	TdhOutTypeXML
	TdhOutTypeErrorcode
	TdhOutTypeWin32error
	TdhOutTypeNtstatus
	TdhOutTypeHresult                    // End of winmeta outtypes.
	TdhOutTypeCultureInsensitiveDatetime // Culture neutral datetime string.
	TdhOutTypeJSON
)

const (
	// Start of TDH outtypes for WBEM.
	TdhOutTypeREDUCEDSTRING = TdhOutType(iota + 300)
	TdhOutTypeNOPRINT
)
