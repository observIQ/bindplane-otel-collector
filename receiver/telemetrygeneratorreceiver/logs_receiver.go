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
	"fmt"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type logsGeneratorReceiver struct {
	telemetryGeneratorReceiver
	nextConsumer consumer.Logs
	generators   []logGenerator
}

// newLogsReceiver creates a new logs specific receiver.
func newLogsReceiver(ctx context.Context, logger *zap.Logger, cfg *Config, nextConsumer consumer.Logs) (*logsGeneratorReceiver, error) {
	lr := &logsGeneratorReceiver{
		nextConsumer: nextConsumer,
	}
	r := newTelemetryGeneratorReceiver(ctx, logger, cfg, lr)

	lr.telemetryGeneratorReceiver = r

	var err error
	lr.generators, err = newLogsGenerators(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new logs generators: %w", err)
	}

	return lr, nil
}

// generateTelemetry
func (r *logsGeneratorReceiver) produce() error {
	logs := plog.NewLogs()
	for _, g := range r.generators {
		l := g.generateLogs()
		for i := 0; i < l.ResourceLogs().Len(); i++ {
			src := l.ResourceLogs().At(i)
			src.CopyTo(logs.ResourceLogs().AppendEmpty())
		}
	}
	return r.nextConsumer.ConsumeLogs(r.ctx, logs)
}
