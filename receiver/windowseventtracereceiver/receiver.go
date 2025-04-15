// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build windows

package windowseventtracereceiver

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw"
)

type logsReceiver struct {
	cfg       *Config
	logger    *zap.Logger
	consumer  consumer.Logs
	stopQueue []func(ctx context.Context) error

	session *etw.Session
	cancel  context.CancelFunc

	wg       *sync.WaitGroup
	doneChan chan struct{}
}

var _ receiver.Logs = (*logsReceiver)(nil)

func newLogsReceiver(cfg *Config, c consumer.Logs, logger *zap.Logger) (*logsReceiver, error) {
	return &logsReceiver{
		cfg:      cfg,
		consumer: c,
		logger:   logger,
		wg:       &sync.WaitGroup{},
		doneChan: make(chan struct{}),
	}, nil
}

func (lr *logsReceiver) Start(ctx context.Context, host component.Host) error {
	s := etw.NewRealTimeSession(lr.cfg.SessionName, lr.logger, lr.cfg.SessionBufferSize)

	if err := s.Start(ctx); err != nil {
		lr.logger.Error("Failed to start standard ETW session", zap.Error(err))
		return fmt.Errorf("failed to start ETW session: %w", err)
	}

	lr.session = s

	lr.wg.Add(1)
	cancelCtx, cancel := context.WithCancel(context.Background())
	lr.cancel = cancel

	// async start to not block starting more components
	go lr.initializeSubscriptions(cancelCtx)
	return nil
}

func (lr *logsReceiver) initializeSubscriptions(ctx context.Context) {
	defer lr.wg.Done()

	// Track successful providers
	successfulProviders := 0
	totalProviders := len(lr.cfg.Providers)

	// Enable all providers with a substantial delay between each
	for i, providerConfig := range lr.cfg.Providers {
		// setting a default level if not provided
		if providerConfig.Level == "" {
			providerConfig.Level = LevelInformational
		}

		err := lr.session.EnableProvider(
			providerConfig.Name,
			providerConfig.Level.toTraceLevel(),
			providerConfig.MatchAnyKeyword,
			providerConfig.MatchAllKeyword,
		)

		if err != nil {
			lr.logger.Error("Failed to enable provider",
				zap.String("provider", providerConfig.Name),
				zap.Error(err),
				zap.Int("index", i),
				zap.Int("total", totalProviders))
			continue
		}
		lr.logger.Info("Enabled provider", zap.String("provider", providerConfig.Name))
		successfulProviders++
	}

	if (successfulProviders == 0 && totalProviders > 0) && lr.cfg.RequireAllProviders {
		lr.logger.Error("Failed to enable any providers",
			zap.String("session", lr.cfg.SessionName),
			zap.Int("totalProviders", totalProviders))
		return
	}

	eventConsumer := etw.NewRealTimeConsumer(ctx, lr.logger, lr.session, lr.cfg.Raw)
	err := eventConsumer.Start(ctx)
	if err != nil {
		lr.logger.Error("Failed to start ETW consumer", zap.Error(err))
		return
	}
	lr.stopQueue = append(lr.stopQueue, func(ctx context.Context) error { return eventConsumer.Stop(ctx) })
	lr.wg.Add(1)
	go lr.listenForEvents(ctx, eventConsumer)
}

func (lr *logsReceiver) listenForEvents(ctx context.Context, eventConsumer *etw.Consumer) {
	defer lr.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-lr.doneChan:
			return
		case event, ok := <-eventConsumer.Events:
			if !ok {
				return
			}
			logs, err := lr.parseLogs(ctx, event)
			if err != nil {
				lr.logger.Error("Failed to parse logs", zap.Error(err))
				continue
			}

			err = lr.consumer.ConsumeLogs(ctx, logs)
			if err != nil {
				lr.logger.Error("Failed to consume logs", zap.Error(err))
			}
		}
	}
}

// TODO think about bundling logs into resources
func (lr *logsReceiver) parseLogs(ctx context.Context, event *etw.Event) (plog.Logs, error) {
	if lr.cfg.Raw {
		return lr.rawEvent(event)
	}
	return lr.parseEvent(event)
}

func (lr *logsReceiver) rawEvent(event *etw.Event) (plog.Logs, error) {
	logs := plog.NewLogs()
	resourceLog := logs.ResourceLogs().AppendEmpty()
	scopeLog := resourceLog.ScopeLogs().AppendEmpty()
	record := scopeLog.LogRecords().AppendEmpty()
	record.Body().SetStr(event.Raw)
	return logs, nil
}

func (lr *logsReceiver) parseEvent(event *etw.Event) (plog.Logs, error) {
	logs := plog.NewLogs()
	resourceLog := logs.ResourceLogs().AppendEmpty()
	resourceLog.Resource().Attributes().PutStr("session", event.Session)
	resourceLog.Resource().Attributes().PutStr("provider", event.System.Provider.Name)
	resourceLog.Resource().Attributes().PutStr("provider_guid", event.System.Provider.GUID)
	resourceLog.Resource().Attributes().PutStr("computer", event.System.Computer)

	scopeLog := resourceLog.ScopeLogs().AppendEmpty()
	record := scopeLog.LogRecords().AppendEmpty()
	lr.parseEventData(event, record)
	return logs, nil
}

// parseEventData parses the event data and sets the log record with that data
func (lr *logsReceiver) parseEventData(event *etw.Event, record plog.LogRecord) {
	record.SetTimestamp(pcommon.NewTimestampFromTime(event.Timestamp))
	record.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	record.SetSeverityNumber(parseSeverity(event.System.Level))

	record.Body().SetEmptyMap()
	record.Body().Map().PutStr("opcode", event.System.Opcode)

	if event.System.Execution.ThreadID != 0 {
		record.Body().Map().PutStr("thread_id", strconv.FormatUint(uint64(event.System.Execution.ThreadID), 10))
	}

	if event.System.Task != "" {
		record.Body().Map().PutStr("task", event.System.Task)
	}

	if event.System.Provider.Name != "" {
		provider := record.Body().Map().PutEmptyMap("provider")
		provider.PutStr("name", event.System.Provider.Name)
		provider.PutStr("guid", event.System.Provider.GUID)
	}

	if len(event.EventData) > 0 {
		message := record.Body().Map().PutEmptyMap("event_data")
		for key, data := range event.EventData {
			message.PutStr(key, fmt.Sprintf("%v", data))
		}
	}

	if event.System.Keywords != "" {
		record.Body().Map().PutStr("keywords", event.System.Keywords)
	}

	if event.System.EventID != "" {
		eventID := record.Body().Map().PutEmptyMap("event_id")
		// eventID.PutStr("guid", event.System.EventGUID)
		eventID.PutStr("id", event.System.EventID)
	}

	if event.System.Execution.ProcessID != 0 {
		execution := record.Body().Map().PutEmptyMap("execution")
		execution.PutStr("process_id", strconv.FormatUint(uint64(event.System.Execution.ProcessID), 10))
		execution.PutStr("thread_id", strconv.FormatUint(uint64(event.System.Execution.ThreadID), 10))
	}

	if len(event.ExtendedData) > 0 {
		extendedData := record.Body().Map().PutEmptySlice("extended_data")
		for _, data := range event.ExtendedData {
			extendedData.AppendEmpty().SetStr(data)
		}
	}

	if len(event.UserData) > 0 {
		userData := record.Body().Map().PutEmptyMap("user_data")
		for key, data := range event.UserData {
			userData.PutStr(key, fmt.Sprintf("%v", data))
		}
	}
}

func (lr *logsReceiver) Shutdown(ctx context.Context) error {
	close(lr.doneChan)
	if lr.cancel != nil {
		lr.cancel()
	}

	for _, stopFunc := range lr.stopQueue {
		if err := stopFunc(ctx); err != nil {
			lr.logger.Error("Failed to perform clean shutdown", zap.Error(err))
		}
	}

	lr.wg.Wait()
	return nil
}

/*
Value	Semantics
LOG_ALWAYS (0)	Event bypasses level-based event filtering. Events should not use this level.
CRITICAL (1)	Critical error
ERROR (2)	Error
WARNING (3)	Warning
INFO (4)	Informational
VERBOSE (5)	Verbose
*/
func parseSeverity(level uint8) plog.SeverityNumber {
	switch level {
	case 0:
		return plog.SeverityNumberInfo
	case 1:
		return plog.SeverityNumberFatal
	case 2:
		return plog.SeverityNumberError
	case 3:
		return plog.SeverityNumberWarn
	case 4:
		return plog.SeverityNumberInfo
	case 5:
		return plog.SeverityNumberTrace
	default:
		return plog.SeverityNumberInfo
	}
}
