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
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	advapi32pkg "github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
)

var (
	rtLostEventGuid = "{6A399AE0-4BC6-4DE9-870B-3657F8947E7E}"
)

// Consumer handles consuming ETW events from sessions
type Consumer struct {
	logger       *zap.Logger
	traceHandles []syscall.Handle
	lastError    error
	closed       bool

	eventCallback  func(eventRecord *advapi32pkg.EventRecord) uintptr
	bufferCallback func(buffer *advapi32pkg.EventTraceLogfile) uintptr

	// Maps trace names to their status
	Traces map[string]*Session

	// Channel for received events
	Events chan *Event

	LostEvents uint64
	Skipped    uint64

	doneChan chan struct{}
	wg       *sync.WaitGroup
}

// NewRealTimeConsumer creates a new Consumer to consume ETW in RealTime mode
func NewRealTimeConsumer(_ context.Context, logger *zap.Logger) *Consumer {
	c := &Consumer{
		traceHandles: make([]syscall.Handle, 0, 64),
		Traces:       make(map[string]*Session),
		Events:       make(chan *Event),
		wg:           &sync.WaitGroup{},
		doneChan:     make(chan struct{}),
		logger:       logger,
	}
	c.eventCallback = c.defaultEventCallback
	c.bufferCallback = c.defaultBufferCallback
	return c
}

// eventCallback is called for each event
func (c *Consumer) defaultEventCallback(eventRecord *advapi32pkg.EventRecord) (rc uintptr) {
	c.logger.Debug("Event callback called",
		zap.Uint16("EventID", eventRecord.EventHeader.EventDescriptor.Id),
		zap.Uint8("Version", eventRecord.EventHeader.EventDescriptor.Version),
		zap.Uint8("Channel", eventRecord.EventHeader.EventDescriptor.Channel),
		zap.Uint8("Level", eventRecord.EventHeader.EventDescriptor.Level),
		zap.Uint8("Opcode", eventRecord.EventHeader.EventDescriptor.Opcode),
		zap.Int64("Timestamp", eventRecord.EventHeader.TimeStamp))

	if eventRecord.EventHeader.ProviderId.String() == rtLostEventGuid {
		c.LostEvents++
		return 1
	}

	helper, err := newEventRecordHelper(eventRecord)
	if err != nil {
		c.logger.Error("Failed to create event record helper", zap.Error(err))
		c.LostEvents++
		rc = 1
		return
	}

	helper.initialize()

	if err := helper.prepareProperties(); err != nil {
		c.logger.Error("Failed to prepare properties", zap.Error(err))
		c.LostEvents++
		rc = 1
		return
	}

	event, err := helper.buildEvent()
	if err != nil {
		c.logger.Info("Failed to build event", zap.Error(err))
		c.LostEvents++
		rc = 1
		return
	}

	select {
	case c.Events <- event:
		rc = 1
	case <-c.doneChan:
		rc = 0
		return
	}
}

func (c *Consumer) defaultBufferCallback(buffer *advapi32pkg.EventTraceLogfile) uintptr {
	if _, ok := <-c.doneChan; ok {
		return 1
	}
	return 0
}

// FromSessions initializes the consumer from sessions
func (c *Consumer) FromSessions(session *Session) *Consumer {
	c.Traces[session.name] = session
	return c
}

// Start starts consuming events from all registered traces
func (c *Consumer) Start(ctx context.Context) error {
	if len(c.traceHandles) == 0 {
		c.traceHandles = make([]syscall.Handle, 0, len(c.Traces))
	}

	// persisting the logfile to avoid memory reallocation
	logfile := advapi32pkg.EventTraceLogfile{}
	for name := range c.Traces {
		logfile = advapi32pkg.EventTraceLogfile{}
		c.logger.Info("starting trace for session", zap.String("session", name))

		// logfile.BufferSize = 64
		// logfile.LogfileHeader = advapi32pkg.TraceLogfileHeader{VersionUnion: 0}
		logfile.SetProcessTraceMode(PROCESS_TRACE_MODE_REAL_TIME | PROCESS_TRACE_MODE_EVENT_RECORD)
		logfile.BufferCallback = syscall.NewCallbackCDecl(c.bufferCallback)
		logfile.Callback = syscall.NewCallbackCDecl(c.eventCallback)
		loggerName, err := syscall.UTF16PtrFromString(name)
		if err != nil {
			c.logger.Error("Failed to convert logger name to UTF-16", zap.Error(err))
			return err
		}
		logfile.LoggerName = loggerName

		traceHandle, err := advapi32pkg.OpenTrace(&logfile)
		if err != nil {
			c.logger.Error("Failed to open trace", zap.Error(err))
			return err
		}

		c.traceHandles = append(c.traceHandles, traceHandle)
	}

	// Process the traces using the appropriate function
	if len(c.traceHandles) > 0 {
		// Add a health check routine that logs a test event occasionally
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()

			for {
				select {
				case <-c.doneChan:
					return
				case <-ticker.C:
					c.logger.Debug("ETW consumer health check",
						zap.Int("trace_handles", len(c.traceHandles)),
						zap.Uint64("lost_events", c.LostEvents),
						zap.Uint64("skipped_events", c.Skipped),
						zap.Int("events", len(c.Events)))
				}
			}
		}()

		for i := range c.traceHandles {
			handle := c.traceHandles[i]
			if handle == syscall.InvalidHandle {
				c.logger.Error("Invalid handle", zap.Uintptr("handle", uintptr(handle)))
				return fmt.Errorf("invalid handle")
			}
			c.logger.Debug("Adding trace handle to consumer", zap.Uintptr("handle", uintptr(handle)))
			c.wg.Add(1)

			// persisting the consumer to avoid memory reallocation
			go func(handle syscall.Handle, logfile advapi32pkg.EventTraceLogfile) {
				defer c.wg.Done()
				defer func() {
					if r := recover(); r != nil {
						c.lastError = fmt.Errorf("ProcessTrace panic: %v", r)
					}
				}()

				for {
					select {
					case <-c.doneChan:
						return
					default:
						// Process trace is a blocking call that will continue to process events until the trace is closed
						if err := advapi32pkg.ProcessTrace(&handle); err != nil {
							c.logger.Error("ProcessTrace failed", zap.Error(err))
						} else {
							c.logger.Info("ProcessTrace completed successfully")
						}
					}
				}

			}(handle, logfile)
		}
	}
	return nil
}

// Stop stops the consumer
func (c *Consumer) Stop(ctx context.Context) error {
	close(c.doneChan)

	// Close all traces
	var lastErr error
	for _, h := range c.traceHandles {
		c.logger.Debug("Closing trace", zap.Uintptr("handle", uintptr(h)))
		err := advapi32pkg.CloseTrace(h)
		if err != nil {
			c.logger.Error("CloseTrace failed", zap.Error(err))
		}
	}

	c.logger.Debug("Waiting for processing to complete", zap.Time("start", time.Now()))
	// Wait for processing to complete
	c.wg.Wait()
	c.logger.Debug("Processing complete", zap.Time("end", time.Now()))
	// Close the events channel
	close(c.Events)
	c.closed = true
	return lastErr
}

// Err returns the last error encountered by the consumer
func (c *Consumer) Err() error {
	return c.lastError
}
