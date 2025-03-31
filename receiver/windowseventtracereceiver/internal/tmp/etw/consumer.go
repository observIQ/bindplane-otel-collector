//go:build windows

package etw

import (
	"context"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	advapi32pkg "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/windows"
)

var (
	rtLostEventGuid = "{6A399AE0-4BC6-4DE9-870B-3657F8947E7E}"
)

// Consumer handles consuming ETW events from sessions
type Consumer struct {
	sync.WaitGroup
	cancel       context.CancelFunc
	traceHandles []syscall.Handle
	lastError    error
	closed       bool

	// Record information for callbacks
	callbackContext unsafe.Pointer

	// Maps trace names to their status
	Traces map[string]bool
	// Channel for received events
	Events chan *Event

	LostEvents uint64
	Skipped    uint64
}

// NewRealTimeConsumer creates a new Consumer to consume ETW in RealTime mode
func NewRealTimeConsumer(_ context.Context) *Consumer {
	c := &Consumer{
		traceHandles: make([]syscall.Handle, 0, 64),
		Traces:       make(map[string]bool),
		Events:       make(chan *Event, 4096),
		WaitGroup:    sync.WaitGroup{},
	}
	return c
}

// eventCallback is called for each event
func (c *Consumer) eventCallback(eventRecord *windows.EventRecord) uintptr {
	fmt.Printf("Debug: eventCallback called\n")
	// Parse the event record using our helper function
	event, err := parseEventRecord(eventRecord)
	if err != nil {

		// Skip events that can't be parsed
		c.Skipped++
		return 1
	}

	// Send the event to the channel
	select {
	case c.Events <- event:
		// Event sent successfully
	default:
		// Channel is full, increment Skipped count
		c.Skipped++
	}

	return 1
}

// FromSessions initializes the consumer from sessions
func (c *Consumer) FromSessions(session *Session) *Consumer {
	c.Traces[session.name] = true
	return c
}

func (c *Consumer) FromMinimalSession(session *MinimalSession) *Consumer {
	c.Traces[session.name] = true
	return c
}

// Start starts consuming events from all registered traces
func (c *Consumer) Start(ctx context.Context) error {
	// Reset state
	c.traceHandles = make([]syscall.Handle, 0, len(c.Traces))

	// Open all registered traces
	for name := range c.Traces {
		// Create a real-time logfile structure
		logfile := advapi32pkg.EventTraceLogfile{}

		// We need to create a new struct for each trace or use struct embedding
		// This is a simplification for now
		namePtr, err := syscall.UTF16PtrFromString(name)
		if err != nil {
			return fmt.Errorf("failed to convert trace name to UTF16: %w", err)
		}
		logfile.LoggerName = namePtr

		// Set up for real-time processing
		logfile.BufferSize = 64
		logfile.Filled = 0
		logfile.EventsLost = 0
		logfile.IsKernelTrace = 0
		logfile.Callback = syscall.NewCallback(c.eventCallback)
		logfile.CurrentTime = 0
		logfile.BuffersRead = 0
		logfile.Union1 = 0
		logfile.Context = 0

		// Set up for real-time mode
		logfile.LogfileHeader.LogFileMode = advapi32pkg.EVENT_TRACE_REAL_TIME_MODE
		logfile.Union1 = uint32(advapi32pkg.PROCESS_TRACE_MODE_REAL_TIME)
		logfile.LogfileHeader.BufferSize = 64
		logfile.LogfileHeader.VersionUnion = 0x00000002 // V2 format
		logfile.LogfileHeader.ProviderVersion = 0
		logfile.LogfileHeader.NumberOfProcessors = 0
		logfile.LogfileHeader.EndTime = 0
		logfile.LogfileHeader.TimerResolution = 0
		logfile.LogfileHeader.MaximumFileSize = 0
		logfile.LogfileHeader.BuffersWritten = 0
		logfile.LogfileHeader.LoggerName = namePtr
		logfile.LogfileHeader.LogFileName = nil
		logfile.LogfileHeader.ReservedFlags = 0
		logfile.LogfileHeader.BuffersLost = 0

		traceHandle, err := advapi32pkg.OpenTrace(&logfile)
		if err != nil {
			return fmt.Errorf("failed to open trace %s: %w", name, err)
		}

		c.traceHandles = append(c.traceHandles, traceHandle)
	}

	// Process the traces using the appropriate function
	// In a real implementation, we would pass all handles here
	if len(c.traceHandles) > 0 {
		for _, handle := range c.traceHandles {
			fmt.Printf("Debug: Processing trace handle: %d\n", handle)
		}
		// ProcessTrace expects a slice of handles
		fmt.Printf("Debug: Processing %d traces\n", len(c.traceHandles))
		err := advapi32pkg.ProcessTrace(c.traceHandles)
		if err != nil {
			fmt.Printf("Debug: ProcessTrace failed: %v\n", err)
			c.lastError = fmt.Errorf("ProcessTrace failed: %w", err)
		}
	}

	return nil
}

// Stop stops the consumer
func (c *Consumer) Stop(ctx context.Context) error {

	// Close all traces
	var lastErr error
	for _, h := range c.traceHandles {
		if err := syscall.CloseHandle(h); err != nil {
			lastErr = err
		}
	}

	// Wait for processing to complete
	c.Wait()

	// Close the events channel
	close(c.Events)
	c.closed = true

	return lastErr
}

// Err returns the last error encountered by the consumer
func (c *Consumer) Err() error {
	return c.lastError
}
