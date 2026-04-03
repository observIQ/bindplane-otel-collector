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
	"context"
	"fmt"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
)

// RawTextLogsConsumer consumes raw text content and creates a single log record with the entire content as the body.
type RawTextLogsConsumer struct {
	nextConsumer consumer.Logs
}

// NewRawTextLogsConsumer creates a new raw text logs consumer
func NewRawTextLogsConsumer(nextConsumer consumer.Logs) *RawTextLogsConsumer {
	return &RawTextLogsConsumer{
		nextConsumer: nextConsumer,
	}
}

// Consume creates a single log record with the entire entityContent as a string body.
func (r *RawTextLogsConsumer) Consume(ctx context.Context, entityContent []byte) error {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	record := scopeLogs.LogRecords().AppendEmpty()
	record.Body().SetStr(string(entityContent))

	if err := r.nextConsumer.ConsumeLogs(ctx, logs); err != nil {
		return fmt.Errorf("text consume: %w", err)
	}
	return nil
}
