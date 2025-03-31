//go:build windows

package tdh

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	wpkg "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/windows"
)

var (
	tdh = syscall.NewLazyDLL("tdh.dll")

	enumProviders = tdh.NewProc("TdhEnumerateProviders")
	getEventInfo  = tdh.NewProc("TdhGetEventInformation")
	getProperty   = tdh.NewProc("TdhGetProperty")
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
	TraceProviderInfoArray []TraceProviderInfo
}

/*
typedef struct _TRACE_PROVIDER_INFO {
  GUID  ProviderGuid;
  ULONG SchemaSource;
  ULONG ProviderNameOffset;
} TRACE_PROVIDER_INFO;
*/

type TraceProviderInfo struct {
	ProviderGuid       windows.GUID
	SchemaSource       uint32
	ProviderNameOffset uint32
}

// Constants for event parsing
const (
	EVENT_HEADER_EXT_TYPE_RELATED_ACTIVITYID = 0x0001
)

// TraceEventInfo represents the event information structure
type TraceEventInfo struct {
	ProviderGUID          windows.GUID
	EventGUID             windows.GUID
	EventDescriptor       EventDescriptor
	DecodingSource        uint32
	ProviderNameOffset    uint32
	LevelNameOffset       uint32
	ChannelNameOffset     uint32
	KeywordsNameOffset    uint32
	TaskNameOffset        uint32
	OpcodeNameOffset      uint32
	EventMessageOffset    uint32
	ProviderMessageOffset uint32
	PropertyCount         uint32
	TopLevelPropertyCount uint32
	Flags                 uint32
	PropertyArray         [1]EventPropertyInfo
}

// EventPropertyInfo represents the property information structure
type EventPropertyInfo struct {
	Flags      uint32
	NameOffset uint32
	Type       uint16
	Length     uint16
	Reserved   uint32
}

// EventDescriptor represents the event descriptor structure
type EventDescriptor struct {
	Id      uint16
	Version uint8
	Channel uint8
	Level   uint8
	Opcode  uint8
	Task    uint16
	Keyword uint64
}

// TdhGetEventInformation retrieves event information
func TdhGetEventInformation(event *wpkg.EventRecord, flags uint32, buffer *TraceEventInfo, bufferSize *uint32) error {
	r1, _, _ := getEventInfo.Call(
		uintptr(unsafe.Pointer(event)),
		uintptr(flags),
		uintptr(0),
		uintptr(unsafe.Pointer(buffer)),
		uintptr(unsafe.Pointer(bufferSize)),
	)
	if r1 != 0 {
		winErr := windows.Errno(r1)
		if winErr == windows.ERROR_INSUFFICIENT_BUFFER {
			return windows.ERROR_INSUFFICIENT_BUFFER
		}
		return fmt.Errorf("TdhGetEventInformation failed with error code %d: %s", r1, winErr.Error())
	}
	return nil
}

// TdhGetProperty retrieves property value
func TdhGetProperty(event *wpkg.EventRecord, propertyInfo *EventPropertyInfo, propertyName string) (interface{}, error) {
	// For now, return a placeholder value
	// In a real implementation, we would parse the property based on its type
	return fmt.Sprintf("Property: %s", propertyName), nil
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
	return &tei.PropertyArray[index]
}

func (tei *TraceEventInfo) PropertyNameOffset(index uint32) uint32 {
	if index >= tei.PropertyCount {
		return 0
	}
	return tei.PropertyArray[index].NameOffset
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

	fmt.Printf("TdhEnumerateProviders succeeded, new buffer size: %d\n", *bufferSize)
	return nil
}
