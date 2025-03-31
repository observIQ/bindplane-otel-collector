//go:build windows
// +build windows

package etw

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"syscall"
)

var (
	rtLostEventGuid = MustParseGUIDFromString("{6A399AE0-4BC6-4DE9-870B-3657F8947E7E}")
)

// SessionSlice converts a slice of structures implementing Session
// to a slice of Session.
func SessionSlice(i interface{}) (out []Session) {
	v := reflect.ValueOf(i)

	switch v.Kind() {
	case reflect.Slice:
		out = make([]Session, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			if s, ok := v.Index(i).Interface().(Session); ok {
				out = append(out, s)
				continue
			}
			panic("slice item must implement Session interface")
		}
	default:
		panic("interface parameter must be []Session")
	}

	return
}

type Consumer struct {
	sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	traceHandles []syscall.Handle
	lastError    error
	closed       bool

	// First callback executed, it allows to filter out events
	// based on fields of raw ETW EventRecord structure. When this callback
	// returns true event processing will continue, otherwise it is aborted.
	// Filtering out events here has the lowest overhead.
	EventRecordCallback func(*EventRecord) bool

	// Callback which executes after TraceEventInfo is parsed.
	// To filter out some events call Skip method of EventRecordHelper
	// As Properties are not parsed yet, trying to get/set Properties is
	// not possible and might cause unexpected behaviours.
	EventRecordHelperCallback func(*EventRecordHelper) error

	// Callback executed after event properties got prepared (step before parsing).
	// Properties are not parsed yet and this is the right place to filter
	// events based only on some properties.
	// NB: events skipped in EventRecordCallback never reach this function
	PreparedCallback func(*EventRecordHelper) error

	// Callback executed after the event got parsed and defines what to do
	// with the event (printed, sent to a channel ...)
	EventCallback func(*Event) error

	Traces map[string]bool
	// Filter EventFilter
	Events chan *Event

	LostEvents uint64

	Skipped uint64
}

// NewRealTimeConsumer creates a new Consumer to consume ETW
// in RealTime mode
func NewRealTimeConsumer(ctx context.Context) (c *Consumer) {
	c = &Consumer{
		traceHandles: make([]syscall.Handle, 0, 64),
		Traces:       make(map[string]bool),
		// Filter:       NewProviderFilter(),
		Events: make(chan *Event, 4096),
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.EventRecordHelperCallback = c.DefaultEventRecordCallback
	c.EventCallback = c.DefaultEventCallback

	return c
}

func (c *Consumer) bufferCallback(e *EventTraceLogfile) uintptr {
	if c.ctx.Err() != nil {
		// if the consumer has been stopped we
		// don't process event records anymore
		return 0
	}
	// we keep processing event records
	return 1
}

func (c *Consumer) callback(er *EventRecord) (rc uintptr) {
	var event *Event

	if er.EventHeader.ProviderId.Equals(rtLostEventGuid) {
		c.LostEvents++
	}

	// calling EventHeaderCallback if possible
	if c.EventRecordCallback != nil {
		if !c.EventRecordCallback(er) {
			return
		}
	}

	// we get the consumer from user context
	if h, err := newEventRecordHelper(er); err == nil {

		if c.EventRecordHelperCallback != nil {
			if err = c.EventRecordHelperCallback(h); err != nil {
				c.lastError = err
			}
		}

		// if event must be skipped we do not further process it
		if h.Flags.Skip {
			return
		}

		// initialize record helper
		h.initialize()

		if err := h.prepareProperties(); err != nil {
			c.lastError = err
			return
		}

		// running a hook before parsing event properties
		if c.PreparedCallback != nil {
			if err := c.PreparedCallback(h); err != nil {
				c.lastError = err
			}
		}

		// check if we must skip event after next hook
		if h.Flags.Skip || c.EventCallback == nil {
			return
		}

		if event, err = h.buildEvent(); err != nil {
			c.lastError = err
		}

		if err := c.EventCallback(event); err != nil {
			c.lastError = err
		}
	}

	return
}

func (c *Consumer) newRealTimeLogfile() (loggerInfo EventTraceLogfile) {
	// PROCESS_TRACE_MODE_EVENT_RECORD to receive EventRecords (new format)
	// PROCESS_TRACE_MODE_RAW_TIMESTAMP don't convert TimeStamp member of EVENT_HEADER and EVENT_TRACE_HEADER converted to system time
	// PROCESS_TRACE_MODE_REAL_TIME to receive events in real time
	//loggerInfo.SetProcessTraceMode(PROCESS_TRACE_MODE_EVENT_RECORD | PROCESS_TRACE_MODE_RAW_TIMESTAMP | PROCESS_TRACE_MODE_REAL_TIME)
	loggerInfo.SetProcessTraceMode(PROCESS_TRACE_MODE_EVENT_RECORD | PROCESS_TRACE_MODE_REAL_TIME)
	loggerInfo.BufferCallback = syscall.NewCallbackCDecl(c.bufferCallback)
	loggerInfo.Callback = syscall.NewCallbackCDecl(c.callback)
	return
}

// close closes the Consumer and eventually waits for ProcessTraces calls
// to end
func (c *Consumer) close(wait bool) (lastErr error) {
	if c.closed {
		return
	}

	// closing trace handles
	for _, h := range c.traceHandles {
		// if we don't wait for traces ERROR_CTX_CLOSE_PENDING is a valid error
		if err := CloseTrace(h); err != nil && err != ERROR_CTX_CLOSE_PENDING {
			lastErr = err
		}
	}

	if wait {
		c.Wait()
	}

	close(c.Events)
	c.closed = true

	return
}

// OpenTrace opens a
func (c *Consumer) OpenTrace(name string) (err error) {
	var traceHandle syscall.Handle

	loggerInfo := c.newRealTimeLogfile()

	// We use the session name to open the trace
	if loggerInfo.LoggerName, err = syscall.UTF16PtrFromString(name); err != nil {
		return err
	}

	if traceHandle, err = OpenTrace(&loggerInfo); err != nil {
		return err
	}

	c.traceHandles = append(c.traceHandles, syscall.Handle(traceHandle))
	return nil
}

// FromSessions initializes the consumer from sessions
func (c *Consumer) FromSessions(sessions ...Session) *Consumer {

	for _, s := range sessions {
		c.InitFilters(s.Providers())
		c.Traces[s.TraceName()] = true
	}

	return c
}

// FromTraceNames initializes consumer from existing traces
func (c *Consumer) FromTraceNames(names ...string) *Consumer {
	for _, n := range names {
		c.Traces[n] = true
	}
	return c
}

// InitFilters initializes event filtering from a Provider slice
func (c *Consumer) InitFilters(providers []Provider) {
	// for _, p := range providers {
	// 	c.Filter.Update(&p)
	// }
}

// DefaultEventRecordCallback is the default EventRecordCallback method applied
// to Consumer created with NewRealTimeConsumer
func (c *Consumer) DefaultEventRecordCallback(h *EventRecordHelper) error {
	// h.Flags.Skip = !c.Filter.Match(h)
	return nil
}

// DefaultEventCallback is the default EventCallback method applied
// to Consumer created with NewRealTimeConsumer
func (c *Consumer) DefaultEventCallback(event *Event) (err error) {
	// we have to check again here as the lock introduced delay
	if c.ctx.Err() == nil {

		// if the event can be skipped we send it in a non-blocking way
		if event.Flags.Skippable {
			select {
			case c.Events <- event:
			default:
				c.Skipped++
			}

			return
		}

		// if we cannot skip event we send it in a blocking way
		c.Events <- event
	}

	return
}

// Start starts the consumer
func (c *Consumer) Start() (err error) {

	// opening all traces first
	for n := range c.Traces {
		if err = c.OpenTrace(n); err != nil {
			return fmt.Errorf("failed to open trace %s: %w", n, err)
		}
	}

	for i := range c.traceHandles {
		i := i
		c.Add(1)
		go func() {
			defer c.Done()
			// ProcessTrace can contain only ONE handle to a real-time processing session
			// src: https://docs.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-processtrace
			if err := ProcessTrace(&c.traceHandles[i], 1, nil, nil); err != nil {
				c.lastError = err
			}
		}()
	}

	return
}

// Err returns the last error encountered by the consumer
func (c *Consumer) Err() error {
	return c.lastError
}

// Stop stops the Consumer and waits for the ProcessTrace calls
// to be terminated
func (c *Consumer) Stop() (err error) {
	// calling context cancel function
	c.cancel()
	return c.close(true)
}
