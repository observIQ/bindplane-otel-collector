//go:build windows

package etw

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	tdhpkg "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/tdh"
	"golang.org/x/sys/windows"

	wpkg "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/windows"
)

// parseEventRecord parses an EventRecord into an Event
func parseEventRecord(eventRecord *wpkg.EventRecord) (*Event, error) {
	// Basic event structure
	event := &Event{
		EventData:    make(map[string]interface{}),
		UserData:     make(map[string]interface{}),
		ExtendedData: make([]string, 0),
	}

	// Set basic properties from the event record
	event.System.Provider.Guid = eventRecord.EventHeader.ProviderId.String()
	event.System.EventID = eventRecord.EventHeader.EventDescriptor.Id
	event.System.Execution.ProcessID = eventRecord.EventHeader.ProcessId
	event.System.Execution.ThreadID = eventRecord.EventHeader.ThreadId

	// Convert timestamp to time.Time
	timeStamp := eventRecord.EventHeader.TimeStamp
	event.System.TimeCreated.SystemTime = time.Unix(0, timeStamp)

	// Set level and other properties
	event.System.Level.Value = eventRecord.EventHeader.EventDescriptor.Level
	event.System.Opcode.Value = eventRecord.EventHeader.EventDescriptor.Opcode
	event.System.Task.Value = uint8(eventRecord.EventHeader.EventDescriptor.Task)
	event.System.Keywords.Value = eventRecord.EventHeader.EventDescriptor.Keyword

	// Get event information using TDH
	bufferSize := uint32(0)
	err := tdhpkg.TdhGetEventInformation(eventRecord, 0, nil, &bufferSize)
	if err != nil && err != syscall.ERROR_INSUFFICIENT_BUFFER {
		return nil, fmt.Errorf("failed to get event information size: %w", err)
	}

	buff := make([]byte, bufferSize)
	tei := (*tdhpkg.TraceEventInfo)(unsafe.Pointer(&buff[0]))
	err = tdhpkg.TdhGetEventInformation(eventRecord, 0, tei, &bufferSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get event information: %w", err)
	}

	// Set provider name and other fields
	event.System.Provider.Name = tei.ProviderName()
	event.System.Task.Name = tei.TaskName()
	event.System.Level.Name = tei.LevelName()
	event.System.Opcode.Name = tei.OpcodeName()
	event.System.Keywords.Name = tei.KeywordName()
	event.System.Channel = tei.ChannelName()

	// Parse event properties
	event.EventData = make(map[string]interface{})
	for i := uint32(0); i < tei.PropertyCount; i++ {
		prop := tei.GetEventPropertyInfoAt(i)
		if prop == nil {
			continue
		}
		name := tei.PropertyNameOffset(i)
		value, err := tdhpkg.TdhGetProperty(eventRecord, prop, fmt.Sprintf("%d", name))
		if err != nil {
			continue // Skip properties we can't parse
		}
		event.EventData[fmt.Sprintf("%v", value)] = value
	}

	// Get the computer name from the system
	computerName, err := windows.ComputerName()
	if err == nil {
		event.System.Computer = computerName
	}

	return event, nil
}

// eventLevelToName maps event level values to their corresponding names
func eventLevelToName(level uint8) string {
	switch level {
	case 1:
		return "Critical"
	case 2:
		return "Error"
	case 3:
		return "Warning"
	case 4:
		return "Information"
	case 5:
		return "Verbose"
	default:
		return fmt.Sprintf("Level%d", level)
	}
}
