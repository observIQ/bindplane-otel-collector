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

package blobconsume //import "github.com/observiq/bindplane-otel-collector/internal/blobconsume"

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// NDJSONLogsConsumer consumes newline-delimited JSON content and converts each line into a log record.
type NDJSONLogsConsumer struct {
	nextConsumer consumer.Logs
	logger       *zap.Logger
}

// NewNDJSONLogsConsumer creates a new NDJSON logs consumer
func NewNDJSONLogsConsumer(nextConsumer consumer.Logs, logger *zap.Logger) *NDJSONLogsConsumer {
	return &NDJSONLogsConsumer{
		nextConsumer: nextConsumer,
		logger:       logger,
	}
}

// Consume splits entityContent by newlines, parses each line as JSON,
// and sends the resulting log records to the next consumer.
func (n *NDJSONLogsConsumer) Consume(ctx context.Context, entityContent []byte) error {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecords := scopeLogs.LogRecords()

	now := pcommon.NewTimestampFromTime(time.Now())
	lines := bytes.Split(entityContent, []byte("\n"))
	skipped := 0

	for i, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var parsed map[string]any
		if err := json.Unmarshal(line, &parsed); err != nil {
			n.logger.Warn("Skipping malformed JSON line",
				zap.Int("line_number", i+1),
				zap.Error(err))
			skipped++
			continue
		}

		tempBody := pcommon.NewMap()
		if err := tempBody.FromRaw(parsed); err != nil {
			n.logger.Warn("Skipping line, failed to set body",
				zap.Int("line_number", i+1),
				zap.Error(err))
			skipped++
			continue
		}

		record := logRecords.AppendEmpty()
		record.SetObservedTimestamp(now)
		tempBody.CopyTo(record.Body().SetEmptyMap())
	}

	if skipped > 0 {
		n.logger.Warn("Skipped malformed lines during NDJSON parsing", zap.Int("skipped_count", skipped))
	}

	if logRecords.Len() == 0 {
		return nil
	}

	if err := n.nextConsumer.ConsumeLogs(ctx, logs); err != nil {
		return fmt.Errorf("ndjson consume: %w", err)
	}
	return nil
}
