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

// Package collector presents an interface that wraps the OTel Collector Core
package collector

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/observiq/bindplane-otel-collector/factories"
	"github.com/observiq/bindplane-otel-collector/internal/measurements"
	"github.com/observiq/bindplane-otel-collector/internal/topology"
	"go.opentelemetry.io/collector/otelcol"
	"go.uber.org/zap"
)

// The timeout for how long to wait on shutting down the collector before bailing out during a Restart.
var collectorRestartTimeout = 10 * time.Second

// Collector is an interface for running the open telemetry collector.
//
//go:generate mockery --name Collector --filename mock_collector.go --structname MockCollector
type Collector interface {
	Run(context.Context) error
	Stop(context.Context)
	Restart(context.Context) error
	SetLoggingOpts([]zap.Option)
	GetLoggingOpts() []zap.Option
	Status() <-chan *Status
}

// collector is the standard implementation of the Collector interface.
type collector struct {
	configPaths []string
	version     string
	loggingOpts []zap.Option
	// factories for modifying in test
	factories  otelcol.Factories
	mux        sync.Mutex
	svc        *otelcol.Collector
	statusChan chan *Status
	wg         *sync.WaitGroup

	// collectorCtx is the context that is fed into collector.Run
	// Cancelling it will force shutdown
	collectorCtx       context.Context
	collectorCtxCancel func()
}

// New returns a new collector.
func New(configPaths []string, version string, loggingOpts []zap.Option) (Collector, error) {
	factories, err := factories.DefaultFactories()
	if err != nil {
		return nil, fmt.Errorf("error while setting up default factories: %w", err)
	}

	return &collector{
		configPaths: configPaths,
		version:     version,
		loggingOpts: loggingOpts,
		statusChan:  make(chan *Status, 10),
		wg:          &sync.WaitGroup{},
		factories:   factories,
	}, nil
}

// GetLoggingOpts returns the current logging options
func (c *collector) GetLoggingOpts() []zap.Option {
	return c.loggingOpts
}

// SetLoggingOpts sets the loggings options. These will take effect on next restart
func (c *collector) SetLoggingOpts(opts []zap.Option) {
	c.loggingOpts = opts
}

// Run will run the collector. This function will return an error
// if the collector was unable to startup.
func (c *collector) Run(ctx context.Context) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.svc != nil {
		return errors.New("service already running")
	}

	// The OT collector only supports using settings once during the lifetime
	// of a single collector instance. We must remake the settings on each startup.
	settings, err := NewSettings(c.configPaths, c.version, c.loggingOpts, c.factories)
	if err != nil {
		return err
	}

	// The OT collector only supports calling run once during the lifetime
	// of a service. We must make a new instance each time we run the collector.
	svc, err := otelcol.NewCollector(*settings)
	if err != nil {
		err := fmt.Errorf("failed to create service: %w", err)
		c.sendStatus(false, false, err)
		return err
	}

	startupErr := make(chan error, 1)

	// Note: This doesn't provide any timeout mechanism if the incoming context is cancelled.
	// If the context passed to Start is cancelled, shutdown could take a very long time (due to e.g. pipeline draining).
	// This is because the collector passes the background context if the start context is cancelled before shutdown is called.
	collectorCtx, collectorCtxCancel := context.WithCancel(ctx)
	c.collectorCtx = collectorCtx
	c.collectorCtxCancel = collectorCtxCancel
	wg := sync.WaitGroup{}
	wg.Add(1)

	c.svc = svc
	c.wg = &wg

	go func() {
		defer wg.Done()

		// Ensure the collectorCtx context is cancelled after the service is done running, even if we cancelled it in Stop.
		defer collectorCtxCancel()

		// Catch panic
		defer func() {
			if r := recover(); r != nil {
				var panicErr error
				panicStack := string(debug.Stack())
				switch v := r.(type) {
				case error:
					panicErr = fmt.Errorf("collector panicked with error: %w. Panic stacktrace: %s", v, panicStack)
				case string:
					panicErr = fmt.Errorf("collector panicked with error: %s. Panic stacktrace: %s", v, panicStack)
				default:
					panicErr = fmt.Errorf("collector panicked with error: %v. Panic stacktrace: %s", v, panicStack)
				}

				c.sendStatus(false, true, panicErr)

				// Send error to startup channel so it doesn't wait for a timeout if a panic occurs.
				startupErr <- panicErr
			}
		}()

		err := svc.Run(collectorCtx)
		c.sendStatus(false, false, err)

		// The error may be nil;
		// We want to signal even in this case, because otherwise waitForStartup could keep waiting
		// for the collector startup, even though the collector will never start up.
		// This can occur if an asynchronous error occurs quickly after collector startup.
		startupErr <- err
	}()

	// A race condition exists in the OT collector where the shutdown channel
	// is not guaranteed to be initialized before the shutdown function is called.
	// We protect against this by waiting for startup to finish before unlocking the mutex.
	return c.waitForStartup(ctx, startupErr)
}

// Stop will stop the collector.
func (c *collector) Stop(ctx context.Context) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.svc == nil {
		return
	}

	c.svc.Shutdown()

	shutdownCompleteChan := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			// Cancel the start context if the stop context is cancelled.
			// Ideally, we'd be able to pass a context into shutdown, but the OTEL collector
			// doesn't support that.
			c.collectorCtxCancel()
		case <-shutdownCompleteChan: // shutdown before context cancellation
		}
	}()

	c.wg.Wait()
	close(shutdownCompleteChan)

	c.svc = nil

	// After shutting down, we reset the registries so they're fresh for the next collector startup.
	measurements.BindplaneAgentThroughputMeasurementsRegistry.Reset()
	topology.BindplaneAgentTopologyRegistry.Reset()
}

// Restart will restart the collector. It will also reset the status channel.
// After calling restart call Status() to get a handle to the new channel.
func (c *collector) Restart(ctx context.Context) error {
	// We stop with a timeout, because we don't want the collector to hang when restarting.
	timeoutCtx, cancel := context.WithTimeout(ctx, collectorRestartTimeout)
	defer cancel()
	c.Stop(timeoutCtx)

	// Reset status channel so it's not polluted by the collector shutting down and restarting
	c.statusChan = make(chan *Status, 10)
	return c.Run(ctx)
}

// waitForStartup waits for the service to startup before exiting.
func (c *collector) waitForStartup(ctx context.Context, startupErr chan error) error {
	ticker := time.NewTicker(time.Millisecond * 250)
	defer ticker.Stop()

	for {
		if c.svc.GetState() == otelcol.StateRunning {
			c.sendStatus(true, false, nil)
			return nil
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			c.svc.Shutdown()
			return ctx.Err()
		case err := <-startupErr:
			if err == nil {
				// We want to report an error here, even if the error is nil, because we did not observe
				// the collector actually start.
				return fmt.Errorf("collector failed to start, and no error was returned")
			}
			return err
		}
	}
}

// Status will return the status of the collector.
func (c *collector) Status() <-chan *Status {
	return c.statusChan
}

// sendStatus will set the status of the collector
func (c *collector) sendStatus(running, panicked bool, err error) {
	select {
	case c.statusChan <- &Status{running, panicked, err}:
	default:
	}
}

// Status is the status of a collector.
type Status struct {
	Running  bool
	Panicked bool
	Err      error
}
