// Copyright  observIQ, Inc.
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

package googlecloudexporter

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// hostname is the name of the current host
var hostname = getHostname()

// googlecloudExporter is a google cloud googlecloudExporter wrapped with additional functionality
type googlecloudExporter struct {
	appendHost bool

	metricsProcessors []component.MetricsProcessor
	metricsExporter   exporter.Metrics
	metricsConsumer   consumer.Metrics

	logsProcessors []component.LogsProcessor
	logsExporter   exporter.Logs
	logsConsumer   consumer.Logs

	tracesProcessors []component.TracesProcessor
	tracesExporter   exporter.Traces
	tracesConsumer   consumer.Traces
}

// ConsumeMetrics consumes metrics
func (e *googlecloudExporter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	if e.appendHost {
		e.appendMetricHost(&md)
	}

	if e.metricsConsumer == nil {
		return nil
	}

	return e.metricsConsumer.ConsumeMetrics(ctx, md)
}

// ConsumeTraces consumes traces
func (e *googlecloudExporter) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	if e.appendHost {
		e.appendTraceHost(&td)
	}

	if e.tracesConsumer == nil {
		return nil
	}

	return e.tracesConsumer.ConsumeTraces(ctx, td)
}

// ConsumeLogs consumes logs
func (e *googlecloudExporter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	if e.appendHost {
		e.appendLogHost(&ld)
	}

	if e.logsConsumer == nil {
		return nil
	}

	return e.logsConsumer.ConsumeLogs(ctx, ld)
}

// Capabilities returns the capabilities of the exporter
func (e *googlecloudExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// Start starts the exporter
func (e *googlecloudExporter) Start(ctx context.Context, host component.Host) error {
	if e.tracesExporter != nil {
		if err := e.tracesExporter.Start(ctx, host); err != nil {
			return fmt.Errorf("failed to start traces exporter: %w", err)
		}
	}

	if e.logsExporter != nil {
		if err := e.logsExporter.Start(ctx, host); err != nil {
			return fmt.Errorf("failed to start logs exporter: %w", err)
		}
	}

	if e.metricsExporter != nil {
		if err := e.metricsExporter.Start(ctx, host); err != nil {
			return fmt.Errorf("failed to start metrics exporter: %w", err)
		}
	}

	for _, processor := range e.tracesProcessors {
		if err := processor.Start(ctx, host); err != nil {
			return fmt.Errorf("failed to start traces processor: %w", err)
		}
	}

	for _, processor := range e.logsProcessors {
		if err := processor.Start(ctx, host); err != nil {
			return fmt.Errorf("failed to start logs processor: %w", err)
		}
	}

	for _, processor := range e.metricsProcessors {
		if err := processor.Start(ctx, host); err != nil {
			return fmt.Errorf("failed to start metrics processor: %w", err)
		}
	}

	return nil
}

// Shutdown will shutdown the exporter
func (e *googlecloudExporter) Shutdown(ctx context.Context) error {
	for i := len(e.tracesProcessors) - 1; i >= 0; i-- {
		if err := e.tracesProcessors[i].Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown traces processor: %w", err)
		}
	}

	for i := len(e.logsProcessors) - 1; i >= 0; i-- {
		if err := e.logsProcessors[i].Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown logs processor: %w", err)
		}
	}

	for i := len(e.metricsProcessors) - 1; i >= 0; i-- {
		if err := e.metricsProcessors[i].Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown metrics processor: %w", err)
		}
	}

	if e.tracesExporter != nil {
		if err := e.tracesExporter.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown traces exporter: %w", err)
		}
	}

	if e.logsExporter != nil {
		if err := e.logsExporter.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown logs exporter: %w", err)
		}
	}

	if e.metricsExporter != nil {
		if err := e.metricsExporter.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown metrics exporter: %w", err)
		}
	}

	return nil
}

// appendMetricHost appends hostname to metrics if not already present
func (e *googlecloudExporter) appendMetricHost(md *pmetric.Metrics) {
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		resourceAttrs := md.ResourceMetrics().At(i).Resource().Attributes()
		_, hostNameExists := resourceAttrs.Get(string(semconv.HostNameKey))
		_, hostIDExists := resourceAttrs.Get(string(semconv.HostIDKey))
		if !hostNameExists && !hostIDExists {
			resourceAttrs.PutStr(string(semconv.HostNameKey), hostname)
		}
	}
}

// appendLogHost appends hostname to logs if not already present
func (e *googlecloudExporter) appendLogHost(ld *plog.Logs) {
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceAttrs := ld.ResourceLogs().At(i).Resource().Attributes()
		_, hostNameExists := resourceAttrs.Get(string(semconv.HostNameKey))
		_, hostIDExists := resourceAttrs.Get(string(semconv.HostIDKey))
		if !hostNameExists && !hostIDExists {
			resourceAttrs.PutStr(string(semconv.HostNameKey), hostname)
		}
	}
}

// appendTraceHost appends hostname to traces if not already present
func (e *googlecloudExporter) appendTraceHost(td *ptrace.Traces) {
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resourceAttrs := td.ResourceSpans().At(i).Resource().Attributes()
		_, hostNameExists := resourceAttrs.Get(string(semconv.HostNameKey))
		_, hostIDExists := resourceAttrs.Get(string(semconv.HostIDKey))
		if !hostNameExists && !hostIDExists {
			resourceAttrs.PutStr(string(semconv.HostNameKey), hostname)
		}
	}
}

// getHostname returns the current hostname or "unknown" if not found
func getHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return "unknown"
}
