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
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
	tdh "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/tdh"
	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/windows"
)

var (
	rtLostEventGuid = "{6A399AE0-4BC6-4DE9-870B-3657F8947E7E}"
)

// Consumer handles consuming ETW events from sessions
type Consumer struct {
	logger      *zap.Logger
	traceHandle *traceHandle
	lastError   error
	closed      bool
	sessionName string
	providerMap map[string]*Provider
	consumeRaw  bool
	session     *Session

	eventCallback      func(eventRecord *advapi32.EventRecord) uintptr
	bufferCallback     func(buffer *advapi32.EventTraceLogfile) uintptr
	getEventProperties func(r *advapi32.EventRecord, logger *zap.Logger) (map[string]any, *tdh.TraceEventInfo, error)

	// Channel for received events
	Events chan *Event

	LostEvents uint64
	Skipped    uint64

	doneChan chan struct{}
	wg       *sync.WaitGroup
}

// NewRealTimeConsumer creates a new Consumer to consume ETW in RealTime mode
func NewRealTimeConsumer(_ context.Context, logger *zap.Logger, session *Session, consumeRaw bool) *Consumer {
	var providerMap map[string]*Provider
	if len(session.providerMap) == 0 {
		providerMap = make(map[string]*Provider)
	} else {
		providerMap = session.providerMap
	}

	c := &Consumer{
		Events:      make(chan *Event),
		wg:          &sync.WaitGroup{},
		doneChan:    make(chan struct{}),
		logger:      logger,
		sessionName: session.name,
		providerMap: providerMap,
		consumeRaw:  consumeRaw,
	}
	c.eventCallback = c.defaultEventCallback
	c.bufferCallback = c.defaultBufferCallback
	c.getEventProperties = GetEventProperties
	c.session = session
	return c
}

var zeroGuid = windows.GUID{}

// eventCallback is called for each event
func (c *Consumer) defaultEventCallback(eventRecord *advapi32.EventRecord) (rc uintptr) {
	if eventRecord == nil {
		c.logger.Error("Event record is nil cannot safely continue processing")
		return 1
	}

	if eventRecord.EventHeader.ProviderId.String() == rtLostEventGuid {
		c.LostEvents++
		c.logger.Error("Lost event", zap.Uint64("total_lost_events", c.LostEvents))
		return 1
	}

	if c.consumeRaw {
		return c.rawEventCallback(eventRecord)
	}
	return c.parsedEventCallback(eventRecord)
}

// TODO; this is kind of a hack, we should use the wevtapi to get properly render the event properties
func (c *Consumer) rawEventCallback(eventRecord *advapi32.EventRecord) uintptr {
	providerGUID := eventRecord.EventHeader.ProviderId.String()
	var providerName string
	if provider, ok := c.providerMap[providerGUID]; ok {
		providerName = provider.Name
	}

	eventData, ti, err := c.getEventProperties(eventRecord, c.logger.Named("event_record_helper"))
	if err != nil {
		c.logger.Error("Failed to get event properties", zap.Error(err))
		return 1
	}
	// Create an XML-like representation
	var xmlBuilder strings.Builder
	xmlBuilder.WriteString("<Event>\n")

	// System section
	xmlBuilder.WriteString("  <System>\n")
	xmlBuilder.WriteString(fmt.Sprintf("    <Provider Name=\"%s\" Guid=\"{%s}\"/>\n",
		xmlEscape(providerName), providerGUID))
	if ti != nil {
		if !ti.EventGUID.Equals(&windows.GUID{}) {
			xmlBuilder.WriteString(fmt.Sprintf("    <EventGuid>%s</EventGuid>\n", ti.EventGUID.String()))
		}
		xmlBuilder.WriteString(fmt.Sprintf("    <DecodingSource>%s</DecodingSource>\n", xmlEscape(decodingSourceName(ti.DecodingSource))))
	}
	xmlBuilder.WriteString(fmt.Sprintf("    <EventID>%d</EventID>\n",
		ti.EventID()))
	xmlBuilder.WriteString(fmt.Sprintf("    <Version>%d</Version>\n",
		eventRecord.EventHeader.EventDescriptor.Version))
	xmlBuilder.WriteString(fmt.Sprintf("    <Level>%d</Level>\n",
		eventRecord.EventHeader.EventDescriptor.Level))
	if levelName := ti.LevelName(); levelName != "" {
		xmlBuilder.WriteString(fmt.Sprintf("    <LevelName>%s</LevelName>\n", xmlEscape(levelName)))
	}
	xmlBuilder.WriteString(fmt.Sprintf("    <Task>%s</Task>\n",
		xmlEscape(ti.TaskName())))
	xmlBuilder.WriteString(fmt.Sprintf("    <Opcode>%s</Opcode>\n",
		xmlEscape(ti.OpcodeName())))
	xmlBuilder.WriteString(fmt.Sprintf("    <Keywords>0x%x</Keywords>\n",
		eventRecord.EventHeader.EventDescriptor.Keyword))
	xmlBuilder.WriteString(fmt.Sprintf("    <Flags>0x%x</Flags>\n",
		eventRecord.EventHeader.Flags))
	if keywordName := ti.KeywordName(); keywordName != "" {
		xmlBuilder.WriteString(fmt.Sprintf("    <KeywordName>%s</KeywordName>\n", xmlEscape(keywordName)))
	}

	timeStr := eventRecord.EventHeader.UTC().Format(time.RFC3339Nano)
	xmlBuilder.WriteString(fmt.Sprintf("    <TimeCreated SystemTime=\"%s\"/>\n", timeStr))

	if !eventRecord.EventHeader.ActivityId.Equals(&windows.GUID{}) {
		xmlBuilder.WriteString(fmt.Sprintf("    <Correlation ActivityID=\"%s\" RelatedActivityID=\"%s\"/>\n",
			eventRecord.EventHeader.ActivityId.String(), eventRecord.RelatedActivityID()))
	} else {
		xmlBuilder.WriteString("    <Correlation />\n")
	}

	xmlBuilder.WriteString(fmt.Sprintf("    <Execution ProcessID=\"%d\" ThreadID=\"%d\"/>\n",
		eventRecord.EventHeader.ProcessId, eventRecord.EventHeader.ThreadId))

	xmlBuilder.WriteString(fmt.Sprintf("    <Channel>%s</Channel>\n", xmlEscape(ti.ChannelName())))

	xmlBuilder.WriteString(fmt.Sprintf("    <Computer>%s</Computer>\n", xmlEscape(hostname)))
	xmlBuilder.WriteString(fmt.Sprintf("    <Session>%s</Session>\n", xmlEscape(c.sessionName)))

	processorNum := uint16(eventRecord.BufferContext.Union & 0xFF)
	if eventRecord.EventHeader.Flags&advapi32.EVENT_HEADER_FLAG_PROCESSOR_INDEX != 0 {
		processorNum = eventRecord.BufferContext.Union
	}
	xmlBuilder.WriteString(fmt.Sprintf("    <ProcessorNumber>%d</ProcessorNumber>\n", processorNum))
	xmlBuilder.WriteString(fmt.Sprintf("    <LoggerId>%d</LoggerId>\n", eventRecord.BufferContext.LoggerId))

	if sid := eventRecord.SID(); sid != "" {
		xmlBuilder.WriteString(fmt.Sprintf("    <Security UserID=\"%s\"/>\n", sid))
	}
	xmlBuilder.WriteString("  </System>\n")

	// EventData/UserData section — tag follows the TEMPLATE_FLAGS layout indicator.
	dataTag := "EventData"
	if ti != nil && ti.Flags&tdh.TEMPLATE_USER_DATA != 0 {
		dataTag = "UserData"
	}
	xmlBuilder.WriteString(fmt.Sprintf("  <%s>\n", dataTag))
	for key, value := range eventData {
		xmlBuilder.WriteString(fmt.Sprintf("    <Data Name=\"%s\">%s</Data>\n", xmlEscape(key), xmlEscape(fmt.Sprintf("%v", value))))
	}
	xmlBuilder.WriteString(fmt.Sprintf("  </%s>\n", dataTag))

	if extData := collectExtendedData(eventRecord); len(extData) > 0 {
		xmlBuilder.WriteString("  <ExtendedData>\n")
		for key, value := range extData {
			xmlBuilder.WriteString(fmt.Sprintf("    <Data Name=\"%s\">%s</Data>\n", xmlEscape(key), xmlEscape(fmt.Sprintf("%v", value))))
		}
		xmlBuilder.WriteString("  </ExtendedData>\n")
	}

	xmlBuilder.WriteString("</Event>")

	event := &Event{
		Timestamp: parseTimestamp(uint64(eventRecord.EventHeader.TimeStamp)),
		Raw:       xmlBuilder.String(),
	}

	select {
	case c.Events <- event:
		return 0
	case <-c.doneChan:
		return 1
	}
}

// xmlEscape returns s with XML special characters escaped so it is safe to
// embed in element content or attribute values.
func xmlEscape(s string) string {
	var buf strings.Builder
	xml.EscapeText(&buf, []byte(s)) //nolint:errcheck // strings.Builder never returns an error
	return buf.String()
}

// collectExtendedData extracts extended data items from an event record into a
// map of named fields. Each known ExtType maps to a descriptive key; unknown
// types are keyed as "ext_0x<hex>" so no data is silently dropped.
//
// RelatedActivityID and SID are also present in dedicated Correlation and
// Security fields on the parsed event, but are included here too because this
// is where the data originates.
//
// Note: item.DataPtr points to Windows-allocated memory outside the Go heap.
// The unsafe pointer arithmetic below is safe because checkptr only validates
// pointers that fall within Go heap allocations, and Windows memory is not
// tracked by the Go runtime.
func collectExtendedData(r *advapi32.EventRecord) map[string]any {
	if r.ExtendedDataCount == 0 {
		return nil
	}

	result := make(map[string]any, int(r.ExtendedDataCount))
	for i := uint16(0); i < r.ExtendedDataCount; i++ {
		item := r.ExtendedDataItem(i)
		if item == nil || item.DataSize == 0 {
			continue
		}

		switch item.ExtType {
		case advapi32.EventHeaderExtTypeRelatedActivityID:
			// Already captured in dedicated Correlation field.
			// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_extended_item_related_activityid
			// We are also capturing this here to indicate the data's source.
			if v := r.RelatedActivityID(); v != "" {
				result["related_activity_id"] = v
			}

		case advapi32.EventHeaderExtTypeSID:
			// Already captured in dedicated Security field.
			// https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-sid
			// We are also capturing this here to indicate the data's source.
			if v := r.SID(); v != "" {
				result["sid"] = v
			}

		case advapi32.EventHeaderExtTypeTerminalSessionID:
			// struct { uint32 SessionId; }
			// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_extended_item_ts_id
			if item.DataSize >= 4 {
				result["terminal_session_id"] = fmt.Sprintf("%d", *(*uint32)(unsafe.Pointer(item.DataPtr)))
			}

		case advapi32.EventHeaderExtTypeInstanceInfo:
			// struct { uint32 InstanceId; uint32 ParentInstanceId; GUID ParentGuid; }
			// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_extended_item_instance
			if item.DataSize >= 24 {
				result["instance_info"] = map[string]any{
					"instance_id":        fmt.Sprintf("%d", *(*uint32)(unsafe.Pointer(item.DataPtr))),
					"parent_instance_id": fmt.Sprintf("%d", *(*uint32)(unsafe.Pointer(item.DataPtr + 4))),
					"parent_guid":        (*windows.GUID)(unsafe.Pointer(item.DataPtr + 8)).String(),
				}
			}

		case advapi32.EventHeaderExtTypeStackTrace32:
			// struct { uint64 MatchId; uint32 Address[]; }
			// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_extended_item_stack_trace32
			if item.DataSize > 8 {
				addrCount := (uint32(item.DataSize) - 8) / 4
				addrs := make([]any, addrCount)
				for j := uint32(0); j < addrCount; j++ {
					addrs[j] = fmt.Sprintf("0x%08x", *(*uint32)(unsafe.Pointer(item.DataPtr + 8 + uintptr(j)*4)))
				}
				result["stack_trace_32"] = addrs
			}

		case advapi32.EventHeaderExtTypeStackTrace64:
			// struct { uint64 MatchId; uint64 Address[]; }
			// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_extended_item_stack_trace64
			if item.DataSize > 8 {
				addrCount := (uint32(item.DataSize) - 8) / 8
				addrs := make([]any, addrCount)
				for j := uint32(0); j < addrCount; j++ {
					addrs[j] = fmt.Sprintf("0x%016x", *(*uint64)(unsafe.Pointer(item.DataPtr + 8 + uintptr(j)*8)))
				}
				result["stack_trace_64"] = addrs
			}

		case advapi32.EventHeaderExtTypePMCCounters:
			// struct { uint64 Counters[]; }
			// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_extended_item_pmc_counters
			if counterCount := uint32(item.DataSize) / 8; counterCount > 0 {
				counters := make([]any, counterCount)
				for j := uint32(0); j < counterCount; j++ {
					counters[j] = fmt.Sprintf("%d", *(*uint64)(unsafe.Pointer(item.DataPtr + uintptr(j)*8)))
				}
				result["pmc_counters"] = counters
			}

		case advapi32.EventHeaderExtTypeEventKey:
			// struct { uint64 EventKey; }
			// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_extended_item_event_key
			if item.DataSize >= 8 {
				result["event_key"] = fmt.Sprintf("0x%016x", *(*uint64)(unsafe.Pointer(item.DataPtr)))
			}

		case advapi32.EventHeaderExtTypeProcessStartKey:
			// struct { uint64 ProcessStartKey; }
			// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_extended_item_process_start_key
			if item.DataSize >= 8 {
				result["process_start_key"] = fmt.Sprintf("0x%016x", *(*uint64)(unsafe.Pointer(item.DataPtr)))
			}

		case advapi32.EventHeaderExtTypeControlGUID:
			// A GUID identifying the control GUID of the provider session.
			// https://microsoft.github.io/windows-docs-rs/doc/windows/Win32/System/Diagnostics/Etw/constant.EVENT_HEADER_EXT_TYPE_CONTROL_GUID.html
			if item.DataSize >= 16 {
				result["control_guid"] = (*windows.GUID)(unsafe.Pointer(item.DataPtr)).String()
			}

		case advapi32.EventHeaderExtTypeContainerID:
			// Container GUID (16 bytes).
			// https://microsoft.github.io/windows-docs-rs/doc/windows/Win32/System/Diagnostics/Etw/constant.EVENT_HEADER_EXT_TYPE_CONTAINER_ID.html
			if item.DataSize >= 16 {
				result["container_id"] = (*windows.GUID)(unsafe.Pointer(item.DataPtr)).String()
			}

		case advapi32.EventHeaderExtTypeStackKey32:
			// struct { uint64 MatchId; uint32 StackKey; uint32 Padding; }
			// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_extended_item_stack_key32
			// MatchId correlates separate kernel-mode and user-mode stack capture events.
			// StackKey is an opaque reference into the kernel's compacted stack trace table.
			if item.DataSize >= 12 {
				result["stack_key_32"] = map[string]any{
					"match_id": fmt.Sprintf("0x%016x", *(*uint64)(unsafe.Pointer(item.DataPtr))),
					"key":      fmt.Sprintf("0x%08x", *(*uint32)(unsafe.Pointer(item.DataPtr + 8))),
				}
			}

		case advapi32.EventHeaderExtTypeStackKey64:
			// struct { uint64 MatchId; uint64 StackKey; }
			// https://learn.microsoft.com/en-us/windows/win32/api/evntcons/ns-evntcons-event_extended_item_stack_key64
			// MatchId correlates separate kernel-mode and user-mode stack capture events.
			// StackKey is an opaque reference into the kernel's compacted stack trace table.
			if item.DataSize >= 16 {
				result["stack_key_64"] = map[string]any{
					"match_id": fmt.Sprintf("0x%016x", *(*uint64)(unsafe.Pointer(item.DataPtr))),
					"key":      fmt.Sprintf("0x%016x", *(*uint64)(unsafe.Pointer(item.DataPtr + 8))),
				}
			}

		default:
			// Known types that currently fall through to hex because no struct definition
			// is publicly documented. If Microsoft publishes struct layouts for these,
			// they can be promoted to their own cases above:
			//
			//   EventHeaderExtTypePEBSIndex     (0x0007) — Intel PEBS hardware counter index; no struct found in SDK docs
			//   EventHeaderExtTypePSMKey        (0x0009) — PSM (Process State Monitor?) key; no struct found in SDK docs
			//   EventHeaderExtTypeEventSchemaTL (0x000B) — TraceLogging schema metadata; variable-length binary blob
			//   EventHeaderExtTypeProvTraits    (0x000C) — provider traits data; variable-length binary blob
			//   EventHeaderExtTypeQPCDelta      (0x000F) — QPC delta from preceding event; likely uint64 but unconfirmed
			//   EventHeaderExtTypeMax           (0x0013) — sentinel upper bound; should never appear in real data
			//
			// Preserve as hex so data is not silently dropped.
			const maxHexBytes = 64
			size := int(item.DataSize)
			truncated := size > maxHexBytes
			if truncated {
				size = maxHexBytes
			}
			data := unsafe.Slice((*byte)(unsafe.Pointer(item.DataPtr)), size)
			v := fmt.Sprintf("%x", data)
			if truncated {
				v += "..."
			}
			result[fmt.Sprintf("ext_0x%04x", item.ExtType)] = v
		}
	}
	return result
}

func decodingSourceName(ds tdh.DecodingSource) string {
	switch ds {
	case tdh.DecodingSourceXMLFile:
		return "xml"
	case tdh.DecodingSourceWbem:
		return "wbem"
	case tdh.DecodingSourceWPP:
		return "wpp"
	default:
		return strconv.Itoa(int(ds))
	}
}

func (c *Consumer) parsedEventCallback(eventRecord *advapi32.EventRecord) uintptr {
	data, ti, err := c.getEventProperties(eventRecord, c.logger.Named("event_record_helper"))
	if err != nil {
		c.logger.Error("Failed to get event properties", zap.Error(err))
		c.LostEvents++
		return 1
	}

	var providerGUID string
	if eventRecord.EventHeader.ProviderId == zeroGuid {
		providerGUID = ""
	} else {
		providerGUID = eventRecord.EventHeader.ProviderId.String()
	}

	var providerName string
	if provider, ok := c.providerMap[providerGUID]; ok {
		providerName = provider.Name
	}

	var eventData map[string]any
	var userData map[string]any
	if ti != nil && ti.Flags&tdh.TEMPLATE_USER_DATA != 0 {
		userData = data
	} else {
		eventData = data
	}

	level := eventRecord.EventHeader.EventDescriptor.Level

	processorNumber := uint16(eventRecord.BufferContext.Union & 0xFF)
	if eventRecord.EventHeader.Flags&advapi32.EVENT_HEADER_FLAG_PROCESSOR_INDEX != 0 {
		processorNumber = eventRecord.BufferContext.Union
	}

	event := &Event{
		Flags:     strconv.FormatUint(uint64(eventRecord.EventHeader.Flags), 10),
		Session:   c.sessionName,
		Timestamp: parseTimestamp(uint64(eventRecord.EventHeader.TimeStamp)),
		System: EventSystem{
			ActivityID:  eventRecord.EventHeader.ActivityId.String(),
			Channel:     ti.ChannelName(),
			Keywords:    strconv.FormatUint(uint64(eventRecord.EventHeader.EventDescriptor.Keyword), 10),
			KeywordName: ti.KeywordName(),
			EventID:     strconv.FormatUint(uint64(ti.EventID()), 10),
			Opcode:      ti.OpcodeName(),
			Task:        ti.TaskName(),
			Provider: EventProvider{
				GUID: providerGUID,
				Name: providerName,
			},
			Level:       level,
			LevelName:   ti.LevelName(),
			Computer:    hostname,
			Correlation: EventCorrelation{},
			Execution: EventExecution{
				ThreadID:  eventRecord.EventHeader.ThreadId,
				ProcessID: eventRecord.EventHeader.ProcessId,
			},
			Version:         eventRecord.EventHeader.EventDescriptor.Version,
			ProcessorNumber: processorNumber,
			LoggerID:        eventRecord.BufferContext.LoggerId,
		},
		Security: EventSecurity{
			SID: eventRecord.SID(),
		},
		EventData:    eventData,
		UserData:     userData,
		ExtendedData: collectExtendedData(eventRecord),
	}

	if ti != nil {
		if !ti.EventGUID.Equals(&windows.GUID{}) {
			event.System.EventGUID = ti.EventGUID.String()
		}
		event.System.DecodingSource = decodingSourceName(ti.DecodingSource)
	}

	if activityID := eventRecord.EventHeader.ActivityId.String(); activityID != zeroGUID {
		event.System.Correlation.ActivityID = activityID
	}

	if relatedActivityID := eventRecord.RelatedActivityID(); relatedActivityID != zeroGUID {
		event.System.Correlation.RelatedActivityID = relatedActivityID
	}

	select {
	case c.Events <- event:
		return 0
	case <-c.doneChan:
		return 1
	}
}

func (c *Consumer) defaultBufferCallback(buffer *advapi32.EventTraceLogfile) uintptr {
	select {
	case <-c.doneChan:
		return 0
	default:
		return 1
	}
}

type traceHandle struct {
	handle  syscall.Handle
	session *Session
}

// Start starts consuming events from all registered traces
func (c *Consumer) Start(_ context.Context) error {
	// persisting the logfile to avoid memory reallocation
	logfile := advapi32.EventTraceLogfile{}
	c.logger.Info("starting trace for session", zap.String("session", c.sessionName))

	logfile.SetProcessTraceMode(advapi32.PROCESS_TRACE_MODE_EVENT_RECORD | advapi32.PROCESS_TRACE_MODE_REAL_TIME)
	logfile.BufferCallback = syscall.NewCallbackCDecl(c.bufferCallback)
	logfile.Callback = syscall.NewCallback(c.eventCallback)
	logfile.Context = 0
	loggerName, err := syscall.UTF16PtrFromString(c.sessionName)
	if err != nil {
		c.logger.Error("Failed to convert logger name to UTF-16", zap.Error(err))
		return err
	}
	logfile.LoggerName = loggerName

	handle, err := advapi32.OpenTrace(&logfile)
	if err != nil {
		c.logger.Error("Failed to open trace", zap.Error(err))
		return err
	}

	c.traceHandle = &traceHandle{
		handle:  handle,
		session: c.session,
	}

	if !isValidHandle(c.traceHandle.handle) {
		c.logger.Error("Invalid handle", zap.Uintptr("handle", uintptr(c.traceHandle.handle)))
		return fmt.Errorf("invalid handle")
	}

	c.logger.Debug("Adding trace handle to consumer", zap.Uintptr("handle", uintptr(c.traceHandle.handle)))
	c.wg.Add(1)

	go func(handle syscall.Handle) {
		defer c.wg.Done()

		var err error

		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("ProcessTrace panic: %v", r)
				}
			}()
			for {
				select {
				case <-c.doneChan:
					return
				default:
					// Process trace is a blocking call that will continue to process events until the trace is closed
					if err := advapi32.ProcessTrace(&handle); err != nil {
						c.logger.Error("ProcessTrace failed", zap.Error(err))
					} else {
						c.logger.Info("ProcessTrace completed successfully")
					}
					return
				}
			}
		}()

		if err != nil {
			c.logger.Error("ProcessTrace failed", zap.Error(err))
		}
	}(c.traceHandle.handle)

	return nil
}

// Stop stops the consumer and closes all opened traces
func (c *Consumer) Stop(ctx context.Context) error {
	if c.closed {
		return nil
	}

	close(c.doneChan)

	var sessionToClose *Session

	var lastErr error
	th := c.traceHandle
	if !isValidHandle(th.handle) {
		c.logger.Error("Invalid handle", zap.Uintptr("handle", uintptr(th.handle)))
		return fmt.Errorf("invalid handle")
	}
	c.logger.Info("Closing trace", zap.Uintptr("handle", uintptr(th.handle)))
	// add a goroutine to close and wait until this trace is closed
	c.wg.Add(1)
	go c.waitForTraceToClose(ctx, th.handle, th.session)
	sessionToClose = th.session

	c.logger.Debug("Waiting for processing to complete", zap.Time("start", time.Now()))
	// Wait for processing to complete
	c.wg.Wait()

	if sessionToClose != nil {
		err := sessionToClose.controller.Stop(ctx)
		if err != nil {
			c.logger.Error("session controller stop failed", zap.Error(err))
		}
	}

	c.logger.Debug("Processing complete", zap.Time("end", time.Now()))
	close(c.Events)
	c.closed = true
	return lastErr
}

func (c *Consumer) waitForTraceToClose(ctx context.Context, handle syscall.Handle, session *Session) {
	defer c.wg.Done()
	jitter := 1
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(jitter) * time.Second):
			r, err := advapi32.CloseTrace(handle)
			switch r {
			case 0:
				jitter++
			// we've deleted it so return
			case windows.ErrorInvalidHandle:
				return
			default:
				c.logger.Debug("StopTrace failed", zap.Error(err))
			}
		}
	}
}

const INVALID_PROCESSTRACE_HANDLE = 0xFFFFFFFFFFFFFFFF

func isValidHandle(handle syscall.Handle) bool {
	return handle != 0 && handle != INVALID_PROCESSTRACE_HANDLE
}
