//go:build windows
// +build windows

package etw

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"syscall"
	"unsafe"
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
	evtPropInfo     *EventPropertyInfo

	name   string
	value  string
	length uint32

	pValue         uintptr
	userDataLength uint16
}

func maxu32(a, b uint32) uint32 {
	if a < b {
		return b
	}
	return a
}

func (p *Property) Parseable() bool {
	return p.evtRecordHelper != nil && p.evtPropInfo != nil && p.pValue > 0
}

func (p *Property) Value() (string, error) {
	var err error

	if p.value == "" && p.Parseable() {
		// we parse only if not already done
		p.value, err = p.parse()
	}

	return p.value, err
}

func (p *Property) parse() (value string, err error) {
	var mapInfo *EventMapInfo
	var udc uint16
	var buff []uint16

	formattedDataSize := maxu32(16, p.length)

	// Get the name/value mapping if the property specifies a value map.
	if p.evtPropInfo.MapNameOffset() > 0 {
		pMapName := (*uint16)(unsafe.Pointer(p.evtRecordHelper.TraceInfo.pointerOffset(uintptr(p.evtPropInfo.MapNameOffset()))))
		decSrc := p.evtRecordHelper.TraceInfo.DecodingSource
		if mapInfo, err = p.evtRecordHelper.EventRec.GetMapInfo(pMapName, uint32(decSrc)); err != nil {
			err = fmt.Errorf("failed to get map info: %s", err)
			return
		}
	}

	for {
		buff = make([]uint16, formattedDataSize)

		err = TdhFormatProperty(
			p.evtRecordHelper.TraceInfo,
			mapInfo,
			p.evtRecordHelper.EventRec.PointerSize(),
			p.evtPropInfo.InType(),
			p.evtPropInfo.OutType(),
			uint16(p.length),
			p.userDataLength,
			(*byte)(unsafe.Pointer(p.pValue)),
			&formattedDataSize,
			&buff[0],
			&udc)

		if err == syscall.ERROR_INSUFFICIENT_BUFFER {
			continue
		}

		if err == ERROR_EVT_INVALID_EVENT_DATA {
			if mapInfo == nil {
				break
			}
			mapInfo = nil
			continue
		}

		if err == nil {
			break
		}

		err = fmt.Errorf("failed to format property : %s", err)
		return
	}

	value = syscall.UTF16ToString(buff)

	return
}

type EventRecordHelper struct {
	EventRec  *EventRecord
	TraceInfo *TraceEventInfo

	Properties      map[string]*Property
	ArrayProperties map[string][]*Property
	Structures      []map[string]*Property

	Flags struct {
		Skip      bool
		Skippable bool
	}

	userDataIt         uintptr
	selectedProperties map[string]bool
}

func newEventRecordHelper(er *EventRecord) (erh *EventRecordHelper, err error) {
	erh = &EventRecordHelper{}
	erh.EventRec = er

	if erh.TraceInfo, err = er.GetEventInformation(); err != nil {
		return
	}

	return
}

func (e *EventRecordHelper) initialize() {
	e.Properties = make(map[string]*Property)
	e.ArrayProperties = make(map[string][]*Property)
	e.Structures = make([]map[string]*Property, 0)
	e.selectedProperties = make(map[string]bool)

	e.userDataIt = e.EventRec.UserData
}

func (e *EventRecordHelper) setEventMetadata(event *Event) {
	event.System.Computer = hostname
	event.System.Execution.ProcessID = e.EventRec.EventHeader.ProcessId
	event.System.Execution.ThreadID = e.EventRec.EventHeader.ThreadId
	event.System.Correlation.ActivityID = e.EventRec.EventHeader.ActivityId.String()
	event.System.Correlation.RelatedActivityID = e.EventRec.RelatedActivityID()
	event.System.EventID = e.TraceInfo.EventID()
	event.System.Channel = e.TraceInfo.ChannelName()
	event.System.Provider.Guid = e.TraceInfo.ProviderGUID.String()
	event.System.Provider.Name = e.TraceInfo.ProviderName()
	event.System.Level.Value = e.TraceInfo.EventDescriptor.Level
	event.System.Level.Name = e.TraceInfo.LevelName()
	event.System.Opcode.Value = e.TraceInfo.EventDescriptor.Opcode
	event.System.Opcode.Name = e.TraceInfo.OpcodeName()
	event.System.Keywords.Value = e.TraceInfo.EventDescriptor.Keyword
	event.System.Keywords.Name = e.TraceInfo.KeywordName()
	event.System.Task.Value = uint8(e.TraceInfo.EventDescriptor.Task)
	event.System.Task.Name = e.TraceInfo.TaskName()
	event.System.TimeCreated.SystemTime = e.EventRec.EventHeader.UTCTimeStamp()

	if e.TraceInfo.IsMof() {
		var eventType string
		if t, ok := MofClassMapping[e.TraceInfo.EventGUID.Data1]; ok {
			eventType = fmt.Sprintf("%s/%s", t.Name, event.System.Opcode.Name)
		} else {
			eventType = fmt.Sprintf("UnknownClass/%s", event.System.Opcode.Name)
		}
		event.System.EventType = eventType
		event.System.EventGuid = e.TraceInfo.EventGUID.String()
	}
}

func (e *EventRecordHelper) endUserData() uintptr {
	return e.EventRec.UserData + uintptr(e.EventRec.UserDataLength)
}

func (e *EventRecordHelper) userDataLength() uint16 {
	return uint16(e.endUserData() - e.userDataIt)
}

func (e *EventRecordHelper) getPropertyLength(i uint32) (uint32, error) {
	if epi := e.TraceInfo.GetEventPropertyInfoAt(i); epi.Flags&PropertyParamLength == PropertyParamLength {
		propSize := uint32(0)
		length := uint32(0)
		j := uint32(epi.LengthPropertyIndex())
		pdd := PropertyDataDescriptor{}
		pdd.PropertyName = uint64(e.TraceInfo.pointer()) + uint64(e.TraceInfo.GetEventPropertyInfoAt(j).NameOffset)
		pdd.ArrayIndex = math.MaxUint32
		if err := TdhGetPropertySize(e.EventRec, 0, nil, 1, &pdd, &propSize); err != nil {
			return 0, fmt.Errorf("failed to get property size: %s", err)
		} else {
			if err := TdhGetProperty(e.EventRec, 0, nil, 1, &pdd, propSize, (*byte)(unsafe.Pointer(&length))); err != nil {
				return 0, fmt.Errorf("failed to get property: %s", err)
			}
			return length, nil
		}
	} else {
		if epi.Length() > 0 {
			return uint32(epi.Length()), nil
		} else {
			switch {
			// if there is an error returned here just try to add a switch case
			// with the proper in type
			case epi.InType() == uint16(TdhInTypeBinary) && epi.OutType() == uint16(TdhOutTypeIpv6):
				// sizeof(IN6_ADDR) == 16
				return uint32(16), nil
			case epi.InType() == uint16(TdhInTypeUnicodestring):
				return uint32(epi.Length()), nil
			case epi.InType() == uint16(TdhInTypeAnsistring):
				return uint32(epi.Length()), nil
			case epi.InType() == uint16(TdhInTypeSid):
				return uint32(epi.Length()), nil
			case epi.InType() == uint16(TdhInTypeWbemsid):
				return uint32(epi.Length()), nil
			case epi.Flags&PropertyStruct == PropertyStruct:
				return uint32(epi.Length()), nil
			default:
				return 0, fmt.Errorf("unexpected length of 0 for intype %d and outtype %d", epi.InType(), epi.OutType())
			}
		}
	}
}

func (e *EventRecordHelper) getPropertySize(i uint32) (size uint32, err error) {
	dataDesc := PropertyDataDescriptor{}
	dataDesc.PropertyName = uint64(e.TraceInfo.PropertyNameOffset(i))
	dataDesc.ArrayIndex = math.MaxUint32
	err = TdhGetPropertySize(e.EventRec, 0, nil, 1, &dataDesc, &size)
	return
}

func (e *EventRecordHelper) getArraySize(i uint32) (arraySize uint16, err error) {
	dataDesc := PropertyDataDescriptor{}
	propSz := uint32(0)

	epi := e.TraceInfo.GetEventPropertyInfoAt(i)
	if (epi.Flags & PropertyParamCount) == PropertyParamCount {
		count := uint32(0)
		j := epi.CountUnion
		dataDesc.PropertyName = uint64(e.TraceInfo.pointer() + uintptr(e.TraceInfo.GetEventPropertyInfoAt(uint32(j)).NameOffset))
		dataDesc.ArrayIndex = math.MaxUint32
		if err = TdhGetPropertySize(e.EventRec, 0, nil, 1, &dataDesc, &propSz); err != nil {
			return
		}
		if err = TdhGetProperty(e.EventRec, 0, nil, 1, &dataDesc, propSz, ((*byte)(unsafe.Pointer(&count)))); err != nil {
			return
		}
		arraySize = uint16(count)
	} else {
		arraySize = epi.CountUnion
	}
	return
}

func (e *EventRecordHelper) prepareProperty(i uint32) (p *Property, err error) {
	var size uint32

	p = &Property{}

	p.evtPropInfo = e.TraceInfo.GetEventPropertyInfoAt(i)
	p.evtRecordHelper = e
	p.name = UTF16AtOffsetToString(e.TraceInfo.pointer(), uintptr(p.evtPropInfo.NameOffset))
	p.pValue = e.userDataIt
	p.userDataLength = e.userDataLength()

	if p.length, err = e.getPropertyLength(i); err != nil {
		err = fmt.Errorf("failed to get property length: %s", err)
		return
	}

	// size is different from length
	if size, err = e.getPropertySize(i); err != nil {
		return
	}

	e.userDataIt += uintptr(size)

	return
}

func (e *EventRecordHelper) prepareProperties() (last error) {
	var arraySize uint16
	var p *Property

	for i := uint32(0); i < e.TraceInfo.TopLevelPropertyCount; i++ {
		epi := e.TraceInfo.GetEventPropertyInfoAt(i)
		isArray := epi.Flags&PropertyParamCount == PropertyParamCount

		switch {
		case isArray:

		case epi.Flags&PropertyParamLength == PropertyParamLength:

		case epi.Flags&PropertyParamCount == PropertyStruct:

		default:
			// property is a map
		}

		if arraySize, last = e.getArraySize(i); last != nil {
			return
		} else {
			var arrayName string
			var array []*Property

			// this is not because we have arraySize > 0 that we are an array
			// so if we deal with an array property
			if isArray {
				array = make([]*Property, 0)
			}

			for k := uint16(0); k < arraySize; k++ {

				// If the property is a structure
				if epi.Flags&PropertyStruct == PropertyStruct {
					propStruct := make(map[string]*Property)
					lastMember := epi.StructStartIndex() + epi.NumOfStructMembers()

					for j := epi.StructStartIndex(); j < lastMember; j++ {
						if p, last = e.prepareProperty(uint32(j)); last != nil {
							return
						} else {
							propStruct[p.name] = p
						}
					}

					e.Structures = append(e.Structures, propStruct)

					continue
				}

				if p, last = e.prepareProperty(i); last != nil {
					return
				}

				if isArray {
					arrayName = p.name
					array = append(array, p)
					continue
				}

				e.Properties[p.name] = p
			}

			if len(array) > 0 {
				e.ArrayProperties[arrayName] = array
			}
		}
	}

	return
}

func (e *EventRecordHelper) buildEvent() (event *Event, err error) {
	event = NewEvent()

	event.Flags.Skippable = e.Flags.Skippable

	if err = e.parseAndSetAllProperties(event); err != nil {
		return
	}

	e.setEventMetadata(event)

	return
}

func (e *EventRecordHelper) parseAndSetProperty(name string, out *Event) (err error) {

	eventData := out.EventData

	// it is a user data property
	if (e.TraceInfo.Flags & TEMPLATE_USER_DATA) == TEMPLATE_USER_DATA {
		eventData = out.UserData
	}

	if p, ok := e.Properties[name]; ok {
		if eventData[p.name], err = p.Value(); err != nil {
			return fmt.Errorf("%w %s: %s", ErrPropertyParsing, name, err)
		}
	}

	// parsing array
	if props, ok := e.ArrayProperties[name]; ok {
		values := make([]string, len(props))

		// iterate over the properties
		for _, p := range props {
			var v string
			if v, err = p.Value(); err != nil {
				return fmt.Errorf("%w array %s: %s", ErrPropertyParsing, name, err)
			}

			values = append(values, v)
		}

		eventData[name] = values
	}

	// parsing structures
	if name == StructurePropertyName {
		if len(e.Structures) > 0 {
			structs := make([]map[string]string, len(e.Structures))
			for _, m := range e.Structures {
				s := make(map[string]string)
				for field, prop := range m {
					if s[field], err = prop.Value(); err != nil {
						return fmt.Errorf("%w %s.%s: %s", ErrPropertyParsing, StructurePropertyName, field, err)
					}
				}
			}

			eventData[StructurePropertyName] = structs
		}
	}

	return
}

func (e *EventRecordHelper) shouldParse(name string) bool {
	if len(e.selectedProperties) == 0 {
		return true
	}
	_, ok := e.selectedProperties[name]
	return ok
}

func (e *EventRecordHelper) parseAndSetAllProperties(out *Event) (last error) {
	var err error

	eventData := out.EventData

	// it is a user data property
	if (e.TraceInfo.Flags & TEMPLATE_USER_DATA) == TEMPLATE_USER_DATA {
		eventData = out.UserData
	}

	// Properties
	for pname, p := range e.Properties {
		if !e.shouldParse(pname) {
			continue
		}
		/*if err := e.parseAndSetProperty(pname, out); err != nil {
			last = err
		}*/
		if eventData[p.name], err = p.Value(); err != nil {
			last = fmt.Errorf("%w %s: %s", ErrPropertyParsing, p.name, err)
		}
	}

	// Arrays
	for pname, props := range e.ArrayProperties {
		if !e.shouldParse(pname) {
			continue
		}

		values := make([]string, len(props))

		// iterate over the properties
		for _, p := range props {
			var v string
			if v, err = p.Value(); err != nil {
				last = fmt.Errorf("%w array %s: %s", ErrPropertyParsing, pname, err)
			}

			values = append(values, v)
		}

		eventData[pname] = values
	}

	// Structure
	if !e.shouldParse(StructurePropertyName) {
		return
	}

	if len(e.Structures) > 0 {
		structs := make([]map[string]string, len(e.Structures))
		for _, m := range e.Structures {
			s := make(map[string]string)
			for field, prop := range m {
				if s[field], err = prop.Value(); err != nil {
					last = fmt.Errorf("%w %s.%s: %s", ErrPropertyParsing, StructurePropertyName, field, err)
				}
			}
		}

		eventData[StructurePropertyName] = structs
	}

	return
}

/** Public methods **/

// SelectFields selects the properties that will be parsed and populated
// in the parsed ETW event. If this method is not called, all properties will
// be parsed and put in the event.
func (e *EventRecordHelper) SelectFields(names ...string) {
	for _, n := range names {
		e.selectedProperties[n] = true
	}
}

func (e *EventRecordHelper) ProviderGUID() string {
	return e.TraceInfo.ProviderGUID.String()
}

func (e *EventRecordHelper) Provider() string {
	return e.TraceInfo.ProviderName()
}

func (e *EventRecordHelper) Channel() string {
	return e.TraceInfo.ChannelName()
}

func (e *EventRecordHelper) EventID() uint16 {
	return e.TraceInfo.EventID()
}

func (e *EventRecordHelper) GetPropertyString(name string) (s string, err error) {

	if p, ok := e.Properties[name]; ok {
		return p.Value()
	}

	return "", fmt.Errorf("%w %s", ErrUnknownProperty, name)
}

func (e *EventRecordHelper) GetPropertyInt(name string) (i int64, err error) {
	var s string

	if s, err = e.GetPropertyString(name); err != nil {
		return
	}

	return strconv.ParseInt(s, 0, 64)
}

func (e *EventRecordHelper) GetPropertyUint(name string) (u uint64, err error) {
	var s string

	if s, err = e.GetPropertyString(name); err != nil {
		return
	}

	return strconv.ParseUint(s, 0, 64)
}

func (e *EventRecordHelper) SetProperty(name, value string) {

	if p, ok := e.Properties[name]; ok {
		p.value = value
		return
	}

	e.Properties[name] = &Property{name: name, value: value}
}

func (e *EventRecordHelper) ParseProperties(names ...string) (err error) {
	for _, name := range names {
		if err = e.ParseProperty(name); err != nil {
			return
		}
	}

	return
}

func (e *EventRecordHelper) ParseProperty(name string) (err error) {
	if p, ok := e.Properties[name]; ok {
		if _, err = p.Value(); err != nil {
			return fmt.Errorf("%w %s: %s", ErrPropertyParsing, name, err)
		}
	}

	// parsing array
	if props, ok := e.ArrayProperties[name]; ok {
		// iterate over the properties
		for _, p := range props {
			if _, err = p.Value(); err != nil {
				return fmt.Errorf("%w array %s: %s", ErrPropertyParsing, name, err)
			}
		}
	}

	// parsing structures
	if name == StructurePropertyName {
		if len(e.Structures) > 0 {
			for _, m := range e.Structures {
				s := make(map[string]string)
				for field, prop := range m {
					if s[field], err = prop.Value(); err != nil {
						return fmt.Errorf("%w %s.%s: %s", ErrPropertyParsing, StructurePropertyName, field, err)
					}
				}
			}
		}
	}

	return
}

func (e *EventRecordHelper) Skippable() {
	e.Flags.Skippable = true
}

func (e *EventRecordHelper) Skip() {
	e.Flags.Skip = true
}
