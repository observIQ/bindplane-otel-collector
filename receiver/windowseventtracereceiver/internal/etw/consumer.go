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
	"fmt"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
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

	eventCallback  func(eventRecord *advapi32.EventRecord) uintptr
	bufferCallback func(buffer *advapi32.EventTraceLogfile) uintptr

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

	eventData, err := GetEventProperties(eventRecord, c.logger.Named("event_record_helper"))
	if err != nil {
		c.logger.Error("Failed to get event properties", zap.Error(err))
		return 1
	}

	channelName, opcodeName, taskName, eventID := c.getEventInfoFromRecord(eventRecord)

	// Create an XML-like representation
	var xmlBuilder strings.Builder
	xmlBuilder.WriteString("<Event>\n")

	// System section
	xmlBuilder.WriteString("  <System>\n")
	xmlBuilder.WriteString(fmt.Sprintf("    <Provider Name=\"%s\" Guid=\"{%s}\"/>\n",
		providerName, providerGUID))
	xmlBuilder.WriteString(fmt.Sprintf("    <EventID>%d</EventID>\n",
		eventID))
	xmlBuilder.WriteString(fmt.Sprintf("    <Version>%d</Version>\n",
		eventRecord.EventHeader.EventDescriptor.Version))
	xmlBuilder.WriteString(fmt.Sprintf("    <Level>%d</Level>\n",
		eventRecord.EventHeader.EventDescriptor.Level))
	xmlBuilder.WriteString(fmt.Sprintf("    <Task>%s</Task>\n",
		taskName))
	xmlBuilder.WriteString(fmt.Sprintf("    <Opcode>%s</Opcode>\n",
		opcodeName))
	xmlBuilder.WriteString(fmt.Sprintf("    <Keywords>0x%x</Keywords>\n",
		eventRecord.EventHeader.EventDescriptor.Keyword))

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

	xmlBuilder.WriteString(fmt.Sprintf("    <Channel>%s</Channel>\n", channelName))

	xmlBuilder.WriteString(fmt.Sprintf("    <Computer>%s</Computer>\n", hostname))

	if sid := eventRecord.SID(); sid != "" {
		xmlBuilder.WriteString(fmt.Sprintf("    <Security UserID=\"%s\"/>\n", sid))
	}
	xmlBuilder.WriteString("  </System>\n")

	// EventData section
	xmlBuilder.WriteString("  <EventData>\n")
	for key, value := range eventData {
		xmlBuilder.WriteString(fmt.Sprintf("    <Data Name=\"%s\">%v</Data>\n", key, value))
	}
	xmlBuilder.WriteString("  </EventData>\n")

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

func (c *Consumer) parsedEventCallback(eventRecord *advapi32.EventRecord) uintptr {
	data, err := GetEventProperties(eventRecord, c.logger.Named("event_record_helper"))
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

	// Get event information from TraceEventInfo
	channelName, opcodeName, taskName, eventID := c.getEventInfoFromRecord(eventRecord)

	level := eventRecord.EventHeader.EventDescriptor.Level
	event := &Event{
		Flags:     strconv.FormatUint(uint64(eventRecord.EventHeader.Flags), 10),
		Session:   c.sessionName,
		Timestamp: parseTimestamp(uint64(eventRecord.EventHeader.TimeStamp)),
		System: EventSystem{
			ActivityID: eventRecord.EventHeader.ActivityId.String(),
			Channel:    channelName,
			Keywords:   strconv.FormatUint(uint64(eventRecord.EventHeader.EventDescriptor.Keyword), 10),
			EventID:    strconv.FormatUint(uint64(eventID), 10),
			Opcode:     opcodeName,
			Task:       taskName,
			Provider: EventProvider{
				GUID: providerGUID,
				Name: providerName,
			},
			Level:       level,
			Computer:    hostname,
			Correlation: EventCorrelation{},
			Execution: EventExecution{
				ThreadID:  eventRecord.EventHeader.ThreadId,
				ProcessID: eventRecord.EventHeader.ProcessId,
			},
			Version: eventRecord.EventHeader.EventDescriptor.Version,
		},
		Security: EventSecurity{
			SID: eventRecord.SID(),
		},
		EventData:    data,
		ExtendedData: []string{},
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

func (c *Consumer) getEventInfoFromRecord(eventRecord *advapi32.EventRecord) (channelName string, opcodeName string, taskName string, eventID uint16) {
	ti, err := getEventInformation(eventRecord)
	if err != nil {
		c.logger.Error("Failed to get event information", zap.Error(err))
		return "", "", "", 0
	}
	return ti.ChannelName(), ti.OpcodeName(), ti.TaskName(), ti.EventID()
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
