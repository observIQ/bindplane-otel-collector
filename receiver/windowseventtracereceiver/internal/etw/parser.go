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
	"math"
	"os"
	"syscall"
	"time"
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

type parser struct {
	eventRecord *advapi32.EventRecord
	tei         *tdh.TraceEventInfo
	ptrSize     uint32
	data        []byte
	logger      *zap.Logger
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
		data:        unsafe.Slice((*uint8)(unsafe.Pointer(r.UserData)), r.UserDataLength),
		logger:      logger,
	}

	properties := make(map[string]any, ti.TopLevelPropertyCount)
	for i := range ti.TopLevelPropertyCount {
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

func (p *parser) getPropertyValue(r *advapi32.EventRecord, propInfo *tdh.TraceEventInfo, i uint32) (any, error) {
	propertyInfo := propInfo.GetEventPropertyInfoAtIndex(i)

	arraySize, err := p.getArraySize(propertyInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to get array size: %w", err)
	}

	result := make([]any, arraySize)
	for i := range arraySize {
		var (
			value any
			err   error
		)
		if (propertyInfo.Flags & tdh.PropertyStruct) == tdh.PropertyStruct {
			value, err = p.parseObject(propertyInfo)
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

// parseObject extracts and returns the fields from an embedded structure within a property.
func (p *parser) parseObject(propertyInfo *tdh.EventPropertyInfo) (map[string]any, error) {
	startIndex := propertyInfo.StructStartIndex()
	lastIndex := startIndex + propertyInfo.NumOfStructMembers()
	diff := lastIndex - startIndex

	object := make(map[string]any, diff)
	for j := startIndex; j < lastIndex; j++ {
		name := p.getPropertyName(int(j))
		value, err := p.getPropertyValue(p.eventRecord, p.tei, uint32(j))
		if err != nil {
			return nil, fmt.Errorf("failed parse field '%s' of complex property type: %w", name, err)
		}
		object[name] = value
	}

	return object, nil
}

func (p *parser) parseSimpleType(r *advapi32.EventRecord, propertyInfo *tdh.EventPropertyInfo, i uint32) (any, error) {
	var mapInfo *tdh.EventMapInfo
	if propertyInfo.MapNameOffset() > 0 {
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
		err := tdh.FormatProperty(
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
			break retryLoop
		case errors.Is(err, windows.ErrorInsufficientBuffer):
			formattedData = make([]byte, formattedDataSize)
			continue
		case errors.Is(err, windows.ErrorEVTInvalidEventData):
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
	mapName := (*uint16)(unsafe.Add(unsafe.Pointer(p.tei), propertyInfo.MapNameOffset()))

	// get the size of the map info buffer
	err := tdh.GetEventMapInformation(p.eventRecord, mapName, nil, &mapSize)
	switch {
	case errors.Is(err, windows.ErrorNotFound):
		// return nil if no map info is available
		return nil, nil
	case errors.Is(err, windows.ErrorInsufficientBuffer):
		// expected error, allocate a larger buffer and try once more
	default:
		return nil, fmt.Errorf("TdhGetEventMapInformation failed to get size: %w", err)
	}

	// Allocate buffer and retrieve the actual map information.
	buff := make([]byte, int(mapSize))
	mapInfo := ((*tdh.EventMapInfo)(unsafe.Pointer(&buff[0])))
	err = tdh.GetEventMapInformation(p.eventRecord, mapName, mapInfo, &mapSize)
	if err != nil {
		return nil, fmt.Errorf("TdhGetEventMapInformation failed: %w", err)
	}

	// return nil if no entries are available
	if mapInfo.EntryCount == 0 {
		return nil, nil
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
		// fixed size of an IPv6 address
		return 16, nil
	}

	// Default case: return the length as defined in the property info.
	// Note: A length of 0 can indicate a variable-length field (e.g., structure, string).
	return uint32(propertyInfo.Length()), nil
}

func parseTimestamp(fileTime uint64) time.Time {
	// Define the offset between Windows epoch (1601) and Unix epoch (1970)
	const epochDifference = 116444736000000000
	if fileTime < epochDifference {
		// Time is before the Unix epoch, adjust accordingly
		return time.Time{}
	}

	windowsTime := windowsSys.Filetime{
		HighDateTime: uint32(fileTime >> 32),
		LowDateTime:  uint32(fileTime & math.MaxUint32),
	}

	return time.Unix(0, windowsTime.Nanoseconds()).UTC()
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
