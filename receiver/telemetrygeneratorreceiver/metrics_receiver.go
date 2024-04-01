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
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type metricsGeneratorReceiver struct {
	telemetryGeneratorReceiver
	nextConsumer consumer.Metrics
	generators   []metricGenerator
}

// newMetricsReceiver creates a new metrics specific receiver.
func newMetricsReceiver(ctx context.Context, logger *zap.Logger, cfg *Config, nextConsumer consumer.Metrics) (*metricsGeneratorReceiver, error) {
	mr := &metricsGeneratorReceiver{
		nextConsumer: nextConsumer,
	}
	r := newTelemetryGeneratorReceiver(ctx, logger, cfg, mr)

	mr.telemetryGeneratorReceiver = r

	var err error
	mr.generators, err = newMetricsGenerators(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new metrics generators: %w", err)
	}

	return mr, nil
}

// produce generates metrics from each generator and sends them to the next consumer
func (r *metricsGeneratorReceiver) produce() error {
	metrics := pmetric.NewMetrics()
	for _, g := range r.generators {
		m := g.generateMetrics()
		for i := 0; i < m.ResourceMetrics().Len(); i++ {
			src := m.ResourceMetrics().At(i)
			src.CopyTo(metrics.ResourceMetrics().AppendEmpty())
		}
	}
	return r.nextConsumer.ConsumeMetrics(r.ctx, metrics)
}
