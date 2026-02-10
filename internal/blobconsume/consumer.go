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
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Consumer is responsible for turning entities into OTLP data and sending to the next consumer.
//
//go:generate mockery --name Consumer --inpackage --with-expecter --filename mock_consumer.go --structname MockConsumer
type Consumer interface {
	// Consume consumes entity contents at the path and unmarshals it.
	Consume(ctx context.Context, entityContent []byte) error
}

// MetricsConsumer consumes rehydrated metric entities and marshals them into pdata structures
type MetricsConsumer struct {
	nextConsumer consumer.Metrics

	unmarshaler *pmetric.JSONUnmarshaler
}

// NewMetricsConsumer creates a new metrics consumer
func NewMetricsConsumer(nextConsumer consumer.Metrics) *MetricsConsumer {
	return &MetricsConsumer{
		nextConsumer: nextConsumer,
		unmarshaler:  &pmetric.JSONUnmarshaler{},
	}
}

// Consume unmarshals entityContent into pmetrics and consumes it
func (m *MetricsConsumer) Consume(ctx context.Context, entityContent []byte) error {
	payload, err := m.unmarshaler.UnmarshalMetrics(entityContent)
	if err != nil {
		return fmt.Errorf("metrics consume: %w", err)
	}

	return m.nextConsumer.ConsumeMetrics(ctx, payload)
}

// LogsConsumer consumes rehydrated log entities and marshals them into pdata structures
type LogsConsumer struct {
	nextConsumer consumer.Logs

	unmarshaler *plog.JSONUnmarshaler
}

// NewLogsConsumer creates a new logs consumer
func NewLogsConsumer(nextConsumer consumer.Logs) *LogsConsumer {
	return &LogsConsumer{
		nextConsumer: nextConsumer,
		unmarshaler:  &plog.JSONUnmarshaler{},
	}
}

// Consume unmarshals entityContent into plogs and consumes it
func (l *LogsConsumer) Consume(ctx context.Context, entityContent []byte) error {
	payload, err := l.unmarshaler.UnmarshalLogs(entityContent)
	if err != nil {
		return fmt.Errorf("logs consume: %w", err)
	}

	return l.nextConsumer.ConsumeLogs(ctx, payload)
}

// TracesConsumer consumes rehydrated trace entities and marshals them into pdata structures
type TracesConsumer struct {
	nextConsumer consumer.Traces

	unmarshaler *ptrace.JSONUnmarshaler
}

// NewTracesConsumer creates a new trace consumer
func NewTracesConsumer(nextConsumer consumer.Traces) *TracesConsumer {
	return &TracesConsumer{
		nextConsumer: nextConsumer,
		unmarshaler:  &ptrace.JSONUnmarshaler{},
	}
}

// Consume unmarshals entityContent into ptrace and consumes it
func (l *TracesConsumer) Consume(ctx context.Context, entityContent []byte) error {
	payload, err := l.unmarshaler.UnmarshalTraces(entityContent)
	if err != nil {
		return fmt.Errorf("traces consume: %w", err)
	}

	return l.nextConsumer.ConsumeTraces(ctx, payload)
}
