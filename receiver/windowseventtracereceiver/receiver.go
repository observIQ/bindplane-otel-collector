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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/0xrawsec/golang-etw/etw"
)

type logsReceiver struct {
	cfg       *Config
	logger    *zap.Logger
	consumer  consumer.Logs
	stopFuncs []func() error

	wg       *sync.WaitGroup
	doneChan chan struct{}
}

var _ receiver.Logs = (*logsReceiver)(nil)

func newLogsReceiver(cfg *Config, c consumer.Logs, logger *zap.Logger) (*logsReceiver, error) {
	return &logsReceiver{cfg: cfg, consumer: c, logger: logger, wg: &sync.WaitGroup{}, doneChan: make(chan struct{})}, nil
}

func (lr *logsReceiver) Start(ctx context.Context, host component.Host) error {
	s := etw.NewRealTimeSession(lr.cfg.SessionName)
	lr.stopFuncs = append(lr.stopFuncs, s.Stop)

	// Enable all providers on this session
	for _, providerConfig := range lr.cfg.Providers {
		provider, err := etw.ParseProvider(providerConfig.Name)
		if err != nil {
			lr.logger.Error("failed to parse provider",
				zap.Error(err),
				zap.String("provider", providerConfig.Name),
				zap.String("session", lr.cfg.SessionName))
			return fmt.Errorf("failed to parse provider %s: %w", providerConfig.Name, err)
		}

		if err := s.EnableProvider(provider); err != nil {
			lr.logger.Error("failed to enable provider",
				zap.Error(err),
				zap.String("provider", providerConfig.Name))
			return fmt.Errorf("failed to enable provider %s: %w", providerConfig.Name, err)
		}

		lr.logger.Info("enabled provider subscription", zap.String("provider", provider.Name), zap.String("providerGuid", provider.GUID))
	}

	// Create a single consumer for the session with all providers
	eventConsumer := etw.NewRealTimeConsumer(ctx)
	eventConsumer = eventConsumer.FromSessions(s)

	// Start the consumer to begin receiving events
	err := eventConsumer.Start()
	if err != nil {
		lr.logger.Error("failed to start ETW consumer", zap.Error(err))
		return fmt.Errorf("failed to start ETW consumer: %w", err)
	}
	lr.stopFuncs = append(lr.stopFuncs, eventConsumer.Stop)

	// Start a single goroutine to listen for events
	lr.wg.Add(1)
	go lr.listenForEvents(ctx, eventConsumer)

	return nil
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
				lr.logger.Error("failed to parse logs", zap.Error(err))
				continue
			}

			err = lr.consumer.ConsumeLogs(ctx, logs)
			if err != nil {
				lr.logger.Error("failed to consume logs", zap.Error(err))
			}
		}
	}
}

// TODO think about bundling logs into resources
func (lr *logsReceiver) parseLogs(ctx context.Context, event *etw.Event) (plog.Logs, error) {
	logs := plog.NewLogs()
	resourceLog := logs.ResourceLogs().AppendEmpty()

	scopeLog := resourceLog.ScopeLogs().AppendEmpty()

	record := scopeLog.LogRecords().AppendEmpty()
	for key, value := range lr.cfg.Attributes {
		record.Attributes().PutStr(key, value)
	}

	lr.parseEventData(event, record)
	return logs, nil
}

// parseEventData parses the event data and sets the log record with that data
func (lr *logsReceiver) parseEventData(event *etw.Event, record plog.LogRecord) {
	record.SetTimestamp(pcommon.NewTimestampFromTime(event.System.TimeCreated.SystemTime))
	record.SetSeverityNumber(parseSeverity(event.System.Level.Name, strconv.FormatUint(uint64(event.System.Level.Value), 10)))

	record.Body().SetEmptyMap()
	record.Body().Map().PutStr("channel", event.System.Channel)
	record.Body().Map().PutStr("computer", event.System.Computer)

	if event.System.Execution.ThreadID != 0 {
		record.Body().Map().PutStr("thread_id", strconv.FormatUint(uint64(event.System.Execution.ThreadID), 10))
	}

	if event.System.Level.Name != "" {
		level := record.Body().Map().PutEmptyMap("level")
		level.PutStr("name", event.System.Level.Name)
		level.PutStr("value", strconv.FormatUint(uint64(event.System.Level.Value), 10))
	}

	if event.System.Opcode.Name != "" {
		opcode := record.Body().Map().PutEmptyMap("opcode")
		opcode.PutStr("name", event.System.Opcode.Name)
		opcode.PutStr("value", strconv.FormatUint(uint64(event.System.Opcode.Value), 10))
	}

	if event.System.Task.Name != "" {
		task := record.Body().Map().PutEmptyMap("task")
		task.PutStr("name", event.System.Task.Name)
		task.PutStr("value", strconv.FormatUint(uint64(event.System.Task.Value), 10))
	}

	if event.System.Provider.Name != "" {
		provider := record.Body().Map().PutEmptyMap("provider")
		provider.PutStr("name", event.System.Provider.Name)
		provider.PutStr("guid", event.System.Provider.Guid)
	}

	if len(event.EventData) > 0 {
		message := record.Body().Map().PutEmptyMap("event_data")
		for key, data := range event.EventData {
			message.PutStr(key, fmt.Sprintf("%v", data))
		}
	}

	if event.System.Keywords.Name != "" {
		keywords := record.Body().Map().PutEmptyMap("keywords")
		keywords.PutStr("name", event.System.Keywords.Name)
		keywords.PutStr("value", strconv.FormatUint(uint64(event.System.Keywords.Value), 10))
	}

	if event.System.EventID != 0 {
		eventID := record.Body().Map().PutEmptyMap("event_id")
		eventID.PutStr("guid", event.System.EventGuid)
		eventID.PutStr("id", strconv.FormatUint(uint64(event.System.EventID), 10))
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

	if event.System.Task.Name != "" {
		record.Body().Map().PutStr("task", event.System.Task.Name)
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
	lr.wg.Wait()
	for _, stopFunc := range lr.stopFuncs {
		if err := stopFunc(); err != nil {
			lr.logger.Error("failed to perform clean shutdown", zap.Error(err))
		}
	}
	return nil
}

// parseRenderedSeverity will parse the severity of the event.
func parseSeverity(levelName, levelValue string) plog.SeverityNumber {
	switch levelName {
	case "":
		switch levelValue {
		case "1":
			return plog.SeverityNumberFatal
		case "2":
			return plog.SeverityNumberError
		case "3":
			return plog.SeverityNumberWarn
		case "4":
			return plog.SeverityNumberInfo
		default:
			return plog.SeverityNumberInfo
		}
	case "Critical":
		return plog.SeverityNumberFatal
	case "Error":
		return plog.SeverityNumberError
	case "Warning":
		return plog.SeverityNumberWarn
	case "Information":
		return plog.SeverityNumberInfo
	default:
		return plog.SeverityNumberInfo
	}
}
