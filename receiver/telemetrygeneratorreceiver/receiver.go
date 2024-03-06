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
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"

	"go.uber.org/zap"
)

// telemetryProducer is an interface for producing telemetry used by the specific receivers
type telemetryProducer interface {
	// produce should generate telemetry from each generator, update the timestamps to now, and send it to the next consumer
	produce() error
}
type telemetryGeneratorReceiver struct {
	logger     *zap.Logger
	cfg        *Config
	doneChan   chan struct{}
	ctx        context.Context
	cancelFunc context.CancelCauseFunc
	producer   telemetryProducer
}

// newTelemetryGeneratorReceiver creates a new telemetry generator receiver
func newTelemetryGeneratorReceiver(ctx context.Context, logger *zap.Logger, cfg *Config, tp telemetryProducer) telemetryGeneratorReceiver {
	ctx, cancel := context.WithCancelCause(ctx)

	return telemetryGeneratorReceiver{
		logger:     logger,
		cfg:        cfg,
		doneChan:   make(chan struct{}),
		ctx:        ctx,
		cancelFunc: cancel,
		producer:   tp,
	}
}

// Shutdown shuts down the telemetry generator receiver
func (r *telemetryGeneratorReceiver) Shutdown(ctx context.Context) error {
	r.cancelFunc(errors.New("shutdown"))
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-r.doneChan:
	}

	return err
}

// Start starts the telemetryGeneratorReceiver receiver
func (r *telemetryGeneratorReceiver) Start(_ context.Context, _ component.Host) error {

	go func() {
		defer close(r.doneChan)

		ticker := time.NewTicker(time.Second / time.Duration(r.cfg.PayloadsPerSecond))
		defer ticker.Stop()

		err := r.producer.produce()
		if err != nil {
			r.logger.Error("Error generating telemetry", zap.Error(err))
		}
		for {
			select {
			case <-r.ctx.Done():
				return
			case <-ticker.C:
				err = r.producer.produce()
				if err != nil {
					r.logger.Error("Error generating telemetry", zap.Error(err))
				}
			}
		}
	}()
	return nil
}
