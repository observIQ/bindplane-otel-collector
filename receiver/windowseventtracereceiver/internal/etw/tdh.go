//go:build windows
// +build windows

package etw

import (
	"syscall"
	"unsafe"
)

/*
TdhEnumerateProviderFieldInformation API wrapper generated from prototype
ULONG __stdcall TdhEnumerateProviderFieldInformation(
	 LPGUID pGuid,
	 EVENT_FIELD_TYPE EventFieldType,
	 PPROVIDER_FIELD_INFOARRAY pBuffer,
	 ULONG *pBufferSize );

Tested: NOK
*/
func TdhEnumerateProviderFieldInformation(
	pGuid *GUID,
	eventFieldType int,
	pBuffer *ProviderFieldInfoArray,
	pBufferSize *uint32) error {
	r1, _, _ := tdhEnumerateProviderFieldInformation.Call(
		uintptr(unsafe.Pointer(pGuid)),
		uintptr(eventFieldType),
		uintptr(unsafe.Pointer(pBuffer)),
		uintptr(unsafe.Pointer(pBufferSize)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
TdhEnumerateProviders API wrapper generated from prototype
ULONG __stdcall TdhEnumerateProviders(
	 PPROVIDER_ENUMERATION_INFO pBuffer,
	 ULONG *pBufferSize );

Tested: NOK
*/
func TdhEnumerateProviders(
	pBuffer *ProviderEnumerationInfo,
	pBufferSize *uint32) error {
	r1, _, _ := tdhEnumerateProviders.Call(
		uintptr(unsafe.Pointer(pBuffer)),
		uintptr(unsafe.Pointer(pBufferSize)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
TdhGetEventInformation API wrapper generated from prototype
ULONG __stdcall TdhGetEventInformation(
	 PEVENT_RECORD pEvent,
	 ULONG TdhContextCount,
	 PTDH_CONTEXT pTdhContext,
	 PTRACE_EVENT_INFO pBuffer,
	 ULONG *pBufferSize );

Tested: OK
*/
func TdhGetEventInformation(pEvent *EventRecord,
	tdhContextCount uint32,
	pTdhContext *TdhContext,
	pBuffer *TraceEventInfo,
	pBufferSize *uint32) error {
	r1, _, _ := tdhGetEventInformation.Call(
		uintptr(unsafe.Pointer(pEvent)),
		uintptr(tdhContextCount),
		uintptr(unsafe.Pointer(pTdhContext)),
		uintptr(unsafe.Pointer(pBuffer)),
		uintptr(unsafe.Pointer(pBufferSize)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
TdhGetEventMapInformation API wrapper generated from prototype
ULONG __stdcall TdhGetEventMapInformation(
	 PEVENT_RECORD pEvent,
	 LPWSTR pMapName,
	 PEVENT_MAP_INFO pBuffer,
	 ULONG *pBufferSize );

Tested: OK
*/
func TdhGetEventMapInformation(pEvent *EventRecord,
	pMapName *uint16,
	pBuffer *EventMapInfo,
	pBufferSize *uint32) error {
	r1, _, _ := tdhGetEventMapInformation.Call(
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
TdhGetProperty API wrapper generated from prototype
ULONG __stdcall TdhGetProperty(
	 PEVENT_RECORD pEvent,
	 ULONG TdhContextCount,
	 PTDH_CONTEXT pTdhContext,
	 ULONG PropertyDataCount,
	 PPROPERTY_DATA_DESCRIPTOR pPropertyData,
	 ULONG BufferSize,
	 PBYTE pBuffer );

Tested: OK
*/
func TdhGetProperty(pEvent *EventRecord,
	tdhContextCount uint32,
	pTdhContext *TdhContext,
	propertyDataCount uint32,
	pPropertyData *PropertyDataDescriptor,
	bufferSize uint32,
	pBuffer *byte) error {
	r1, _, _ := tdhGetProperty.Call(
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

/*
TdhGetPropertySize API wrapper generated from prototype
ULONG __stdcall TdhGetPropertySize(
	 PEVENT_RECORD pEvent,
	 ULONG TdhContextCount,
	 PTDH_CONTEXT pTdhContext,
	 ULONG PropertyDataCount,
	 PPROPERTY_DATA_DESCRIPTOR pPropertyData,
	 ULONG *pPropertySize );

Tested: OK
*/
func TdhGetPropertySize(pEvent *EventRecord,
	tdhContextCount uint32,
	pTdhContext *TdhContext,
	propertyDataCount uint32,
	pPropertyData *PropertyDataDescriptor,
	pPropertySize *uint32) error {
	r1, _, _ := tdhGetPropertySize.Call(
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
TdhQueryProviderFieldInformation API wrapper generated from prototype
ULONG __stdcall TdhQueryProviderFieldInformation(
	 LPGUID pGuid,
	 ULONGLONG EventFieldValue,
	 EVENT_FIELD_TYPE EventFieldType,
	 PPROVIDER_FIELD_INFOARRAY pBuffer,
	 ULONG *pBufferSize );

Tested: NOK
*/
func TdhQueryProviderFieldInformation(
	pGuid *GUID,
	eventFieldValue uint64,
	eventFieldType int,
	pBuffer *ProviderFieldInfoArray,
	pBufferSize *uint32) error {
	r1, _, _ := tdhQueryProviderFieldInformation.Call(
		uintptr(unsafe.Pointer(pGuid)),
		uintptr(eventFieldValue),
		uintptr(eventFieldType),
		uintptr(unsafe.Pointer(pBuffer)),
		uintptr(unsafe.Pointer(pBufferSize)))
	if r1 == 0 {
		return nil
	}
	return syscall.Errno(r1)
}

/*
TdhFormatProperty API wrapper generated from prototype
TDHSTATUS TdhFormatProperty(
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

Tested: OK
*/
func TdhFormatProperty(
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
	r1, _, _ := tdhFormatProperty.Call(
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
