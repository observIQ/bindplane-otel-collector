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
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	windowsSys "golang.org/x/sys/windows"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
	tdh "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/tdh"
	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/windows"
	"go.uber.org/zap"
)

const (
	StructurePropertyName = "Structures"
)

var (
	hostname, _ = os.Hostname()

	ErrPropertyParsing = fmt.Errorf("error parsing property")
	ErrUnknownProperty = fmt.Errorf("unknown property")
)

type Property struct {
	evtRecordHelper *EventRecordHelper
	evtPropInfo     *tdh.EventPropertyInfo

	name   string
	value  string
	length uint32

	pValue         uintptr
	userDataLength uint16
}

type EventRecordHelper struct {
	EventRec  *advapi32.EventRecord
	TraceInfo *tdh.TraceEventInfo

	Properties      map[string]*Property
	ArrayProperties map[string][]*Property
	Structures      []map[string]*Property

	Flags struct {
		Skip      bool
		Skippable bool
	}

	userDataIt         uintptr
	selectedProperties map[string]bool

	logger *zap.Logger
}

func GetEventProperties(r *advapi32.EventRecord, logger *zap.Logger) (map[string]any, error) {
	if r.EventHeader.Flags == advapi32.EVENT_HEADER_FLAG_STRING_ONLY {
		userDataPtr := (*uint16)(unsafe.Pointer(r.UserData))
		return map[string]any{
			"UserData": utf16StringAtOffset(uintptr(unsafe.Pointer(userDataPtr)), 0),
		}, nil
	}

	ti, err := getEventInformation(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get event information: %w", err)
	}

	p := &parser{
		eventRecord: r,
		tei:         ti,
		ptrSize:     r.PointerSize(),
	}

	properties := make(map[string]any, ti.TopLevelPropertyCount)
	for i := 0; i < int(ti.TopLevelPropertyCount); i++ {
		namePtr := unsafe.Add(unsafe.Pointer(ti), ti.GetEventPropertyInfoAtIndex(uint32(i)).NameOffset)
		propName := windowsSys.UTF16PtrToString((*uint16)(namePtr))
		value, err := p.getPropertyValue(r, ti, uint32(i))
		if err != nil {
			return nil, fmt.Errorf("failed to get property value: %w", err)
		}

		properties[propName] = value
	}

	return properties, nil
}

type parser struct {
	eventRecord *advapi32.EventRecord
	tei         *tdh.TraceEventInfo
	ptrSize     uint32
	data        []byte
}

func (p *parser) getPropertyValue(r *advapi32.EventRecord, propInfo *tdh.TraceEventInfo, i uint32) (any, error) {
	propertyInfo := propInfo.GetEventPropertyInfoAtIndex(i)

	arraySize, err := p.getArraySize(propertyInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to get array size: %w", err)
	}

	result := make([]any, arraySize)
	for i := 0; i < int(arraySize); i++ {
		var (
			value any
			err   error
		)
		if (propertyInfo.Flags & tdh.PropertyStruct) == tdh.PropertyStruct {
			value, err = p.parseStruct(propertyInfo)
		} else {
			value, err = p.parseSimpleType(r, propertyInfo, uint32(i))
		}

		if err != nil {
			return nil, err
		}
		result[i] = value
	}

	return result, nil
}

// parseStruct extracts and returns the fields from an embedded structure within a property.
func (p *parser) parseStruct(propertyInfo *tdh.EventPropertyInfo) (map[string]any, error) {
	// Determine the start and end indexes of the structure members within the property info.
	startIndex := propertyInfo.StructStartIndex()
	lastIndex := startIndex + propertyInfo.NumOfStructMembers()

	// Initialize a map to hold the structure's fields.
	structure := make(map[string]any, (lastIndex - startIndex))
	// Iterate through each member of the structure.
	for j := startIndex; j < lastIndex; j++ {
		name := p.getPropertyName(int(j))
		value, err := p.getPropertyValue(p.eventRecord, p.tei, uint32(j))
		if err != nil {
			return nil, fmt.Errorf("failed parse field '%s' of complex property type: %w", name, err)
		}
		structure[name] = value // Add the field to the structure map.
	}

	return structure, nil
}

func (p *parser) parseSimpleType(r *advapi32.EventRecord, propertyInfo *tdh.EventPropertyInfo, i uint32) (any, error) {
	var mapInfo *tdh.EventMapInfo
	if propertyInfo.MapNameOffset() > 0 {
		// If failed retrieving the map information, returns on error
		var err error
		mapInfo, err = p.getMapInfo(propertyInfo)
		if err != nil {
			return "", fmt.Errorf("failed to get map information due to: %w", err)
		}
	}

	// Get the length of the property.
	propertyLength, err := p.getPropertyLength(propertyInfo)
	if err != nil {
		return "", fmt.Errorf("failed to get property length due to: %w", err)
	}

	var userDataConsumed uint16

	// Set a default buffer size for formatted data.
	const DEFAULT_PROPERTY_BUFFER_SIZE = 1024
	formattedDataSize := uint32(DEFAULT_PROPERTY_BUFFER_SIZE)
	formattedData := make([]byte, int(formattedDataSize))

	// Retry loop to handle buffer size adjustments.
retryLoop:
	for {
		var dataPtr *uint8
		if len(p.data) > 0 {
			dataPtr = &p.data[0]
		}
		err := tdh.TdhFormatProperty(
			p.tei,
			mapInfo,
			p.ptrSize,
			propertyInfo.InType(),
			propertyInfo.OutType(),
			uint16(propertyLength),
			uint16(len(p.data)),
			dataPtr,
			&formattedDataSize,
			(*uint16)(unsafe.Pointer(&formattedData[0])),
			&userDataConsumed,
		)

		switch {
		case err == nil:
			// If formatting is successful, break out of the loop.
			break retryLoop
		case errors.Is(err, windows.ErrorInsufficientBuffer):
			// Increase the buffer size if it's insufficient.
			formattedData = make([]byte, formattedDataSize)
			continue
		case errors.Is(err, windows.ErrorEVTInvalidEventData):
			// Handle invalid event data error.
			// Discarding MapInfo allows us to access
			// at least the non-interpreted data.
			if mapInfo != nil {
				mapInfo = nil
				continue
			}
			return "", fmt.Errorf("TdhFormatProperty failed: %w", err) // Handle unknown error
		default:
			return "", fmt.Errorf("TdhFormatProperty failed: %w", err)
		}
	}
	// Update the data slice to account for consumed data.
	p.data = p.data[userDataConsumed:]

	// Convert the formatted data to string and return.
	return windowsSys.UTF16PtrToString((*uint16)(unsafe.Pointer(&formattedData[0]))), nil
}

// getArraySize calculates the size of an array property within an event.
func (p *parser) getArraySize(propertyInfo *tdh.EventPropertyInfo) (uint32, error) {
	const PropertyParamCount = 0x0001
	if (propertyInfo.Flags & PropertyParamCount) == PropertyParamCount {
		var dataDescriptor tdh.PropertyDataDescriptor
		dataDescriptor.PropertyName = readPropertyName(p, int(propertyInfo.Count()))
		dataDescriptor.ArrayIndex = 0xFFFFFFFF
		return getLengthFromProperty(p.eventRecord, &dataDescriptor)
	} else {
		return uint32(propertyInfo.Count()), nil
	}
}

// getPropertyName retrieves the name of the i-th event property in the event record.
func (p *parser) getPropertyName(i int) string {
	// Convert the UTF16 property name to a Go string.
	namePtr := readPropertyName(p, i)
	return windowsSys.UTF16PtrToString((*uint16)(namePtr))
}

// readPropertyName gets the pointer to the property name in the event information structure.
func readPropertyName(p *parser, i int) unsafe.Pointer {
	// Calculate the pointer to the property name using its offset in the event property array.
	return unsafe.Add(unsafe.Pointer(p.tei), p.tei.GetEventPropertyInfoAtIndex(uint32(i)).NameOffset)
}

func getLengthFromProperty(r *advapi32.EventRecord, dataDescriptor *tdh.PropertyDataDescriptor) (uint32, error) {
	var length uint32
	// Call TdhGetProperty to get the length of the property specified by the dataDescriptor.
	err := tdh.TdhGetProperty(
		r,
		0,
		nil,
		1,
		dataDescriptor,
		uint32(unsafe.Sizeof(length)),
		(*byte)(unsafe.Pointer(&length)),
	)
	if err != nil {
		return 0, err
	}
	return length, nil
}

func getEventInformation(r *advapi32.EventRecord) (*tdh.TraceEventInfo, error) {
	var bufferSize uint32
	tei := &tdh.TraceEventInfo{}
	if err := tdh.GetEventInformation(r, 0, nil, nil, &bufferSize); errors.Is(err, windows.ErrorInsufficientBuffer) {
		buffer := make([]byte, bufferSize)
		tei = (*tdh.TraceEventInfo)(unsafe.Pointer(&buffer[0]))
		if err := tdh.GetEventInformation(r, 0, nil, tei, &bufferSize); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get event information: %w", err)
	}
	return tei, nil
}

func (p *parser) getMapInfo(propertyInfo *tdh.EventPropertyInfo) (*tdh.EventMapInfo, error) {
	var mapSize uint32
	// Get the name of the map from the property info.
	mapName := (*uint16)(unsafe.Add(unsafe.Pointer(p.tei), propertyInfo.MapNameOffset()))

	// First call to get the required size of the map info.
	err := tdh.TdhGetEventMapInformation(p.eventRecord, mapName, nil, &mapSize)
	switch {
	case errors.Is(err, windows.ErrorNotFound):
		// No mapping information available. This is not an error.
		return nil, nil
	case errors.Is(err, windows.ErrorInsufficientBuffer):
		// Resize the buffer and try again.
	default:
		return nil, fmt.Errorf("TdhGetEventMapInformation failed to get size: %w", err)
	}

	// Allocate buffer and retrieve the actual map information.
	buff := make([]byte, int(mapSize))
	mapInfo := ((*tdh.EventMapInfo)(unsafe.Pointer(&buff[0])))
	err = tdh.TdhGetEventMapInformation(p.eventRecord, mapName, mapInfo, &mapSize)
	if err != nil {
		return nil, fmt.Errorf("TdhGetEventMapInformation failed: %w", err)
	}

	if mapInfo.EntryCount == 0 {
		return nil, nil // No entries in the map.
	}

	return mapInfo, nil
}

func (p *parser) getPropertyLength(propertyInfo *tdh.EventPropertyInfo) (uint32, error) {
	// Check if the length of the property is defined by another property.
	if (propertyInfo.Flags & tdh.PropertyParamLength) == tdh.PropertyParamLength {
		var dataDescriptor tdh.PropertyDataDescriptor
		// Read the property name that contains the length information.
		dataDescriptor.PropertyName = readPropertyName(p, int(propertyInfo.LengthPropertyIndex()))
		dataDescriptor.ArrayIndex = 0xFFFFFFFF
		// Retrieve the length from the specified property.
		return getLengthFromProperty(p.eventRecord, &dataDescriptor)
	}

	inType := propertyInfo.InType()
	outType := propertyInfo.OutType()
	// Special handling for properties representing IPv6 addresses.
	// https://docs.microsoft.com/en-us/windows/win32/api/tdh/nf-tdh-tdhformatproperty#remarks
	if uint16(tdh.TdhInTypeBinary) == inType && uint16(tdh.TdhOutTypeIpv6) == outType {
		// Return the fixed size of an IPv6 address.
		return 16, nil
	}

	// Default case: return the length as defined in the property info.
	// Note: A length of 0 can indicate a variable-length field (e.g., structure, string).
	return uint32(propertyInfo.Length()), nil
}

func UTF16AtOffsetToString(pstruct uintptr, offset uintptr) string {
	out := make([]uint16, 0, 64)
	wc := (*uint16)(unsafe.Pointer(pstruct + offset))
	for i := uintptr(2); *wc != 0; i += 2 {
		out = append(out, *wc)
		wc = (*uint16)(unsafe.Pointer(pstruct + offset + i))
	}
	return syscall.UTF16ToString(out)
}
