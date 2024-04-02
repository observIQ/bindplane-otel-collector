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

package pluginreceiver

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/otelcol"
	"go.uber.org/zap"
)

// Receiver is a receiver that runs an embedded open telemetry config
// as an internal service.
type Receiver struct {
	plugin         *Plugin
	renderedCfg    *RenderedConfig
	emitterFactory exporter.Factory
	logger         *zap.Logger
	createService  createServiceFunc
	service        Service

	// serviceErrChan gets the error from the service once it stops running.
	// If there is no error, `nil` is placed on the channel.
	// It will be closed after the collector service shuts down.
	serviceErrChan chan error
}

// NewReceiver creates a new plugin receiver
func NewReceiver(
	plugin *Plugin,
	renderedConfig *RenderedConfig,
	emitterFactory exporter.Factory,
	logger *zap.Logger,
) *Receiver {
	return &Receiver{
		plugin:         plugin,
		renderedCfg:    renderedConfig,
		emitterFactory: emitterFactory,
		logger:         logger,
		createService:  createService,
		serviceErrChan: make(chan error, 1),
	}
}

// Start starts the receiver's internal service
func (r *Receiver) Start(ctx context.Context, host component.Host) error {
	r.logger.Info("Starting plugin...", zap.String("plugin", r.plugin.Title), zap.String("plugin-version", r.plugin.Version))

	factories, err := r.renderedCfg.GetRequiredFactories(host, r.emitterFactory)
	if err != nil {
		return fmt.Errorf("failed to get factories from factory provider: %w", err)
	}

	cfgProvider, err := r.renderedCfg.GetConfigProvider()
	if err != nil {
		return fmt.Errorf("failed to get config provider: %w", err)
	}

	service, err := r.createService(*factories, cfgProvider, r.logger)
	if err != nil {
		return fmt.Errorf("failed to create internal service: %w", err)
	}
	r.service = service

	if err := r.startService(ctx, service); err != nil {
		return fmt.Errorf("failed to start internal service: %w", err)
	}

	return nil
}

// Shutdown stops the receiver's internal service
func (r *Receiver) Shutdown(ctx context.Context) error {
	if r.service != nil {
		r.service.Shutdown()

		// Wait for service to actually shutdown, since shutdown is asynchronous
		select {
		case <-ctx.Done():
			r.logger.Warn("Context done before service properly shut down.", zap.Error(ctx.Err()))
		case err := <-r.serviceErrChan:
			if err != nil {
				return fmt.Errorf("shutdown service: %w", err)
			}
		}
	}

	return nil
}

// startService starts the provided service
func (r *Receiver) startService(ctx context.Context, svc Service) error {
	go func() {
		r.serviceErrChan <- svc.Run(ctx)
		close(r.serviceErrChan)
	}()

	ticker := time.NewTicker(time.Millisecond * 250)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-r.serviceErrChan:
			return err
		case <-ticker.C:
			if svc.GetState() == otelcol.StateRunning {
				return nil
			}
		}
	}
}
