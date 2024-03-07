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

package telemetrygeneratorreceiver //import "github.com/observiq/bindplane-agent/receiver/telemetrygeneratorreceiver"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type tracesGeneratorReceiver struct {
	telemetryGeneratorReceiver
	nextConsumer consumer.Traces
	generators   []traceGenerator
}

// newTracesReceiver creates a new traces specific receiver.
func newTracesReceiver(ctx context.Context, logger *zap.Logger, cfg *Config, nextConsumer consumer.Traces) *tracesGeneratorReceiver {
	tr := &tracesGeneratorReceiver{
		nextConsumer: nextConsumer,
	}

	r := newTelemetryGeneratorReceiver(ctx, logger, cfg, tr)

	tr.telemetryGeneratorReceiver = r
	tr.generators = newTraceGenerators(cfg, logger)

	return tr
}

// produce generates traces from each generator and sends them to the next consumer
func (r *tracesGeneratorReceiver) produce() error {
	traces := ptrace.NewTraces()
	for _, g := range r.generators {
		t := g.generateTraces()
		for i := 0; i < t.ResourceSpans().Len(); i++ {
			src := t.ResourceSpans().At(i)
			src.CopyTo(traces.ResourceSpans().AppendEmpty())
		}
	}
	return r.nextConsumer.ConsumeTraces(r.ctx, traces)
}
