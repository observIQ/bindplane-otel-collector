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

package throughputmeasurementprocessor

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/observiq/bindplane-agent/internal/measurements"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type throughputMeasurementProcessor struct {
	logger              *zap.Logger
	enabled             bool
	measurements        *measurements.ThroughputMeasurements
	samplingCutOffRatio float64
	processorID         component.ID
	bindplane           component.ID
	startOnce           sync.Once
}

func newThroughputMeasurementProcessor(logger *zap.Logger, mp metric.MeterProvider, cfg *Config, processorID component.ID) (*throughputMeasurementProcessor, error) {
	measurements, err := measurements.NewThroughputMeasurements(mp, processorID.String(), cfg.ExtraLabels)
	if err != nil {
		return nil, fmt.Errorf("create throughput measurements: %w", err)
	}

	return &throughputMeasurementProcessor{
		logger:              logger,
		enabled:             cfg.Enabled,
		measurements:        measurements,
		samplingCutOffRatio: cfg.SamplingRatio,
		processorID:         processorID,
		bindplane:           cfg.BindplaneExtension,
		startOnce:           sync.Once{},
	}, nil
}

func (tmp *throughputMeasurementProcessor) start(_ context.Context, host component.Host) error {
	var err error
	tmp.startOnce.Do(func() {
		registry, getRegErr := GetThroughputRegistry(host, tmp.bindplane)
		if getRegErr != nil {
			err = fmt.Errorf("get throughput registry: %w", getRegErr)
			return
		}

		if registry != nil {
			registerErr := registry.RegisterThroughputMeasurements(tmp.processorID.String(), tmp.measurements)
			if registerErr != nil {
				err = fmt.Errorf("register throughput measurements: %w", registerErr)
				return
			}
		}
	})

	return err
}

func (tmp *throughputMeasurementProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if tmp.enabled {
		//#nosec G404 -- randomly generated number is not used for security purposes. It's ok if it's weak
		if rand.Float64() <= tmp.samplingCutOffRatio {
			tmp.measurements.AddTraces(ctx, td)
		}
	}

	return td, nil
}

func (tmp *throughputMeasurementProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	if tmp.enabled {
		//#nosec G404 -- randomly generated number is not used for security purposes. It's ok if it's weak
		if rand.Float64() <= tmp.samplingCutOffRatio {
			tmp.measurements.AddLogs(ctx, ld)
		}
	}

	return ld, nil
}

func (tmp *throughputMeasurementProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if tmp.enabled {
		//#nosec G404 -- randomly generated number is not used for security purposes. It's ok if it's weak
		if rand.Float64() <= tmp.samplingCutOffRatio {
			tmp.measurements.AddMetrics(ctx, md)
		}
	}

	return md, nil
}

func (tmp *throughputMeasurementProcessor) shutdown(_ context.Context) error {
	unregisterProcessor(tmp.processorID)
	return nil
}
