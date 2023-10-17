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

package azureblobrehydrationreceiver //import "github.com/observiq/bindplane-agent/receiver/azureblobrehydrationreceiver"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// blobConsumer is responsible for turning blobs into OTLP data and sending to the next consumer.
//
//go:generate mockery --name blobConsumer --output ./internal/mocks --with-expecter --filename mock_blob_consumer.go --structname MockBlobConsumer
type blobConsumer interface {
	// Consume consumes blob contents at the path and unmarshals it.
	Consume(ctx context.Context, blobContent []byte) error
}

// metricsConsumer
type metricsConsumer struct {
	nextConsumer consumer.Metrics

	unmarshaler *pmetric.JSONUnmarshaler
}

// newMetricsConsumer creates a new metrics consumer
func newMetricsConsumer(nextConsumer consumer.Metrics) *metricsConsumer {
	return &metricsConsumer{
		nextConsumer: nextConsumer,
		unmarshaler:  &pmetric.JSONUnmarshaler{},
	}
}

// Consume unmarshals blobContent into pmetrics and consumes it
func (m *metricsConsumer) Consume(ctx context.Context, blobContent []byte) error {
	payload, err := m.unmarshaler.UnmarshalMetrics(blobContent)
	if err != nil {
		return fmt.Errorf("metrics consume: %w", err)
	}

	return m.nextConsumer.ConsumeMetrics(ctx, payload)
}

// logsConsumer
type logsConsumer struct {
	nextConsumer consumer.Logs

	unmarshaler *plog.JSONUnmarshaler
}

// newLogsConsumer creates a new logs consumer
func newLogsConsumer(nextConsumer consumer.Logs) *logsConsumer {
	return &logsConsumer{
		nextConsumer: nextConsumer,
		unmarshaler:  &plog.JSONUnmarshaler{},
	}
}

// Consume unmarshals blobContent into plogs and consumes it
func (l *logsConsumer) Consume(ctx context.Context, blobContent []byte) error {
	payload, err := l.unmarshaler.UnmarshalLogs(blobContent)
	if err != nil {
		return fmt.Errorf("logs consume: %w", err)
	}

	return l.nextConsumer.ConsumeLogs(ctx, payload)
}

// tracesConsumer
type tracesConsumer struct {
	nextConsumer consumer.Traces

	unmarshaler *ptrace.JSONUnmarshaler
}

// newTracesConsumer creates a new trace consumer
func newTracesConsumer(nextConsumer consumer.Traces) *tracesConsumer {
	return &tracesConsumer{
		nextConsumer: nextConsumer,
		unmarshaler:  &ptrace.JSONUnmarshaler{},
	}
}

// Consume unmarshals blobContent into ptrace and consumes it
func (l *tracesConsumer) Consume(ctx context.Context, blobContent []byte) error {
	payload, err := l.unmarshaler.UnmarshalTraces(blobContent)
	if err != nil {
		return fmt.Errorf("traces consume: %w", err)
	}

	return l.nextConsumer.ConsumeTraces(ctx, payload)
}
