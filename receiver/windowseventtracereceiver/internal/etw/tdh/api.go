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

package tdh

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
	windows_ "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/windows"
	"golang.org/x/sys/windows"
)

var (
	tdh = syscall.NewLazyDLL("tdh.dll")

	enumProviders          = tdh.NewProc("TdhEnumerateProviders")
	formatProperty         = tdh.NewProc("TdhFormatProperty")
	getEventInfo           = tdh.NewProc("TdhGetEventInformation")
	getEventMapInformation = tdh.NewProc("TdhGetEventMapInformation")
	getProperty            = tdh.NewProc("TdhGetProperty")
	getPropertySize        = tdh.NewProc("TdhGetPropertySize")
)

/*
GetEventInformation API wrapper generated from prototype
ULONG __stdcall GetEventInformation(
	 PEVENT_RECORD pEvent,
	 ULONG TdhContextCount,
	 PTDH_CONTEXT pTdhContext,
	 PTRACE_EVENT_INFO pBuffer,
	 ULONG *pBufferSize );

*/
// GetEventInformation retrieves event information
func GetEventInformation(
	event *advapi32.EventRecord,
	tdhContextCount uint32,
	pTdhContext *TdhContext,
	pBuffer *TraceEventInfo,
	pBufferSize *uint32,
) error {
	r0, _, _ := getEventInfo.Call(
		uintptr(unsafe.Pointer(event)),
		uintptr(tdhContextCount),
		uintptr(unsafe.Pointer(pTdhContext)),
		uintptr(unsafe.Pointer(pBuffer)),
		uintptr(unsafe.Pointer(pBufferSize)))
	if r0 == 0 {
		return nil
	}
	return syscall.Errno(r0)
}

/*
GetEventMapInformation API wrapper generated from prototype
ULONG __stdcall GetEventMapInformation(

	PEVENT_RECORD pEvent,
	LPWSTR pMapName,
	PEVENT_MAP_INFO pBuffer,
	ULONG *pBufferSize );
*/
func GetEventMapInformation(pEvent *advapi32.EventRecord,
	pMapName *uint16,
	pBuffer *EventMapInfo,
	pBufferSize *uint32) error {
	r1, _, _ := getEventMapInformation.Call(
		uintptr(unsafe.Pointer(pEvent)),
		uintptr(unsafe.Pointer(pMapName)),
		uintptr(unsafe.Pointer(pBuffer)),
		uintptr(unsafe.Pointer(pBufferSize)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
TdhGetPropertySize API wrapper generated from prototype
ULONG __stdcall TdhGetPropertySize(

	PEVENT_RECORD pEvent,
	ULONG TdhContextCount,
	PTDH_CONTEXT pTdhContext,
	ULONG PropertyDataCount,
	PPROPERTY_DATA_DESCRIPTOR pPropertyData,
	ULONG *pPropertySize );
*/
func TdhGetPropertySize(pEvent *advapi32.EventRecord,
	tdhContextCount uint32,
	pTdhContext *TdhContext,
	propertyDataCount uint32,
	pPropertyData *PropertyDataDescriptor,
	pPropertySize *uint32) error {
	r1, _, _ := getPropertySize.Call(
		uintptr(unsafe.Pointer(pEvent)),
		uintptr(tdhContextCount),
		uintptr(unsafe.Pointer(pTdhContext)),
		uintptr(propertyDataCount),
		uintptr(unsafe.Pointer(pPropertyData)),
		uintptr(unsafe.Pointer(pPropertySize)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
FormatProperty API wrapper generated from prototype
TDHSTATUS FormatProperty(

	PTRACE_EVENT_INFO EventInfo,
	PEVENT_MAP_INFO MapInfo,
	ULONG PointerSize,
	USHORT PropertyInType,
	USHORT PropertyOutType,
	USHORT PropertyLength,
	USHORT UserDataLength,
	PBYTE UserData,
	PULONG BufferSize,
	PWCHAR Buffer,
	PUSHORT UserDataConsumed );
*/
func FormatProperty(
	eventInfo *TraceEventInfo,
	mapInfo *EventMapInfo,
	pointerSize uint32,
	propertyInType uint16,
	propertyOutType uint16,
	propertyLength uint16,
	userDataLength uint16,
	userData *byte,
	bufferSize *uint32,
	buffer *uint16,
	userDataConsumed *uint16) error {
	r1, _, _ := formatProperty.Call(
		uintptr(unsafe.Pointer(eventInfo)),
		uintptr(unsafe.Pointer(mapInfo)),
		uintptr(pointerSize),
		uintptr(propertyInType),
		uintptr(propertyOutType),
		uintptr(propertyLength),
		uintptr(userDataLength),
		uintptr(unsafe.Pointer(userData)),
		uintptr(unsafe.Pointer(bufferSize)),
		uintptr(unsafe.Pointer(buffer)),
		uintptr(unsafe.Pointer(userDataConsumed)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
TdhGetProperty API wrapper generated from prototype
ULONG __stdcall TdhGetProperty(

	PEVENT_RECORD pEvent,
	ULONG TdhContextCount,
	PTDH_CONTEXT pTdhContext,
	ULONG PropertyDataCount,
	PPROPERTY_DATA_DESCRIPTOR pPropertyData,
	ULONG BufferSize,
	PBYTE pBuffer );
*/
func TdhGetProperty(pEvent *advapi32.EventRecord,
	tdhContextCount uint32,
	pTdhContext *TdhContext,
	propertyDataCount uint32,
	pPropertyData *PropertyDataDescriptor,
	bufferSize uint32,
	pBuffer *byte) error {
	r1, _, _ := getProperty.Call(
		uintptr(unsafe.Pointer(pEvent)),
		uintptr(tdhContextCount),
		uintptr(unsafe.Pointer(pTdhContext)),
		uintptr(propertyDataCount),
		uintptr(unsafe.Pointer(pPropertyData)),
		uintptr(bufferSize),
		uintptr(unsafe.Pointer(pBuffer)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

// Helper methods for TraceEventInfo
func (tei *TraceEventInfo) ProviderName() string {
	return stringAt(unsafe.Pointer(tei), tei.ProviderNameOffset)
}

func (tei *TraceEventInfo) TaskName() string {
	return stringAt(unsafe.Pointer(tei), tei.TaskNameOffset)
}

func (tei *TraceEventInfo) LevelName() string {
	return stringAt(unsafe.Pointer(tei), tei.LevelNameOffset)
}

func (tei *TraceEventInfo) OpcodeName() string {
	return stringAt(unsafe.Pointer(tei), tei.OpcodeNameOffset)
}

func (tei *TraceEventInfo) KeywordName() string {
	return stringAt(unsafe.Pointer(tei), tei.KeywordsNameOffset)
}

func (tei *TraceEventInfo) ChannelName() string {
	return stringAt(unsafe.Pointer(tei), tei.ChannelNameOffset)
}

func (tei *TraceEventInfo) GetEventPropertyInfoAt(index uint32) *EventPropertyInfo {
	if index >= tei.PropertyCount {
		return nil
	}
	return &tei.EventPropertyInfoArray[index]
}

// Seems to be always empty
func (t *TraceEventInfo) RelatedActivityIDName() string {
	return t.stringAt(t.Pointer(), uintptr(t.RelatedActivityIDNameOffset))
}

func (t *TraceEventInfo) Pointer() uintptr {
	return uintptr(unsafe.Pointer(t))
}

func (t *TraceEventInfo) PointerOffset(offset uintptr) uintptr {
	return t.Pointer() + offset
}

func (t *TraceEventInfo) stringAt(pstruct uintptr, offset uintptr) string {
	out := make([]uint16, 0, 64)
	wc := (*uint16)(unsafe.Pointer(pstruct + offset))
	for i := uintptr(2); *wc != 0; i += 2 {
		out = append(out, *wc)
		wc = (*uint16)(unsafe.Pointer(pstruct + offset + i))
	}
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(pstruct + offset)))
}

func (tei *TraceEventInfo) PropertyNameOffset(index uint32) uint32 {
	if index >= tei.PropertyCount {
		return 0
	}
	return tei.EventPropertyInfoArray[index].NameOffset
}

func (t *TraceEventInfo) IsMof() bool {
	return t.DecodingSource == DecodingSourceWbem
}

func (t *TraceEventInfo) EventID() uint16 {
	if t.IsXML() {
		return t.EventDescriptor.Id
	} else if t.IsMof() {
		if c, ok := windows_.MofClassMapping[t.EventGUID.Data1]; ok {
			return c.BaseId + uint16(t.EventDescriptor.Opcode)
		}
	}
	// not meaningful, cannot be used to identify event
	return 0
}

func (t *TraceEventInfo) IsXML() bool {
	return t.DecodingSource == DecodingSourceXMLFile
}

// Helper function to get string at offset
func stringAt(base unsafe.Pointer, offset uint32) string {
	if offset == 0 {
		return ""
	}
	ptr := unsafe.Add(base, uintptr(offset))
	return windows.UTF16PtrToString((*uint16)(ptr))
}

// EnumerateProviders enumerates the providers in the buffer
// returns windows.ERROR_INSUFFICIENT_BUFFER if the buffer is too small
func EnumerateProviders(pei *ProviderEnumerationInfo, bufferSize *uint32) error {
	r1, _, _ := enumProviders.Call(
		uintptr(unsafe.Pointer(pei)),
		uintptr(unsafe.Pointer(bufferSize)),
	)
	if r1 != 0 {
		winErr := windows.Errno(r1)
		// if the error is ERROR_INSUFFICIENT_BUFFER, we need to expand our buffer
		if winErr == windows.ERROR_INSUFFICIENT_BUFFER {
			return windows.ERROR_INSUFFICIENT_BUFFER
		}
		errMsg := winErr.Error()
		return fmt.Errorf("TdhEnumerateProviders failed with error code %d: %s", r1, errMsg)
	}

	return nil
}

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
	PropertyName unsafe.Pointer
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
	ProviderGuid       windows_.GUID
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
	ProviderGUID                windows_.GUID
	EventGUID                   windows_.GUID
	EventDescriptor             advapi32.EventDescriptor
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

// getEventPropertyInfoAtIndex looks for the EventPropertyInfo object at a specified index.
func (info *TraceEventInfo) GetEventPropertyInfoAtIndex(i uint32) *EventPropertyInfo {
	if i < info.PropertyCount {
		// Calculate the address of the first element in EventPropertyInfoArray.
		eventPropertyInfoPtr := uintptr(unsafe.Pointer(&info.EventPropertyInfoArray[0]))
		// Adjust the pointer to point to the i-th EventPropertyInfo element.
		eventPropertyInfoPtr += uintptr(i) * unsafe.Sizeof(EventPropertyInfo{})

		return ((*EventPropertyInfo)(unsafe.Pointer(eventPropertyInfoPtr)))
	}
	return nil
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
		byteLen := (wcslen(((*uint16)(unsafe.Pointer(pStr)))) - 1) * 2
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

func wcslen(uintf16 *uint16) (len uint64) {
	for it := uintptr((unsafe.Pointer(uintf16))); ; it += 2 {
		wc := (*uint16)(unsafe.Pointer(it))
		if *wc == 0 {
			return
		}
		len++
	}
}
