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

package bindplaneextension

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang/snappy"
	"github.com/observiq/bindplane-agent/internal/measurements"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type bindplaneExtension struct {
	logger                  *zap.Logger
	cfg                     *Config
	ctmr                    *measurements.ResettableThroughputMeasurementsRegistry
	customCapabilityHandler opampcustommessages.CustomCapabilityHandler
	doneChan                chan struct{}
	wg                      *sync.WaitGroup
}

func newBindplaneExtension(logger *zap.Logger, cfg *Config) *bindplaneExtension {
	return &bindplaneExtension{
		logger:   logger,
		cfg:      cfg,
		ctmr:     measurements.NewResettableThroughputMeasurementsRegistry(false),
		doneChan: make(chan struct{}),
		wg:       &sync.WaitGroup{},
	}
}

func (b *bindplaneExtension) Start(_ context.Context, host component.Host) error {
	var emptyComponentID component.ID

	// Set up measurements if enabled
	if b.cfg.OpAMP != emptyComponentID && b.cfg.MeasurementsInterval > 0 {
		err := b.setupCustomCapabilities(host)
		if err != nil {
			return fmt.Errorf("setup capability handler: %w", err)
		}

		b.wg.Add(1)
		go b.reportMetricsLoop()
	}

	return nil
}

func (b *bindplaneExtension) RegisterThroughputMeasurements(processorID string, measurements *measurements.ThroughputMeasurements) error {
	return b.ctmr.RegisterThroughputMeasurements(processorID, measurements)
}

func (b *bindplaneExtension) setupCustomCapabilities(host component.Host) error {
	ext, ok := host.GetExtensions()[b.cfg.OpAMP]
	if !ok {
		return fmt.Errorf("opamp extension %q does not exist", b.cfg.OpAMP)
	}

	registry, ok := ext.(opampcustommessages.CustomCapabilityRegistry)
	if !ok {
		return fmt.Errorf("extension %q is not an custom message registry", b.cfg.OpAMP)
	}

	var err error
	b.customCapabilityHandler, err = registry.Register(measurements.ReportMeasurementsV1Capability)
	if err != nil {
		return fmt.Errorf("register custom capability: %w", err)
	}

	return nil
}

func (b *bindplaneExtension) Dependencies() []component.ID {
	var emptyComponentID component.ID
	if b.cfg.OpAMP == emptyComponentID {
		return nil
	}

	return []component.ID{b.cfg.OpAMP}
}

func (b *bindplaneExtension) reportMetricsLoop() {
	defer b.wg.Done()

	t := time.NewTicker(b.cfg.MeasurementsInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			err := b.reportMetrics()
			if err != nil {
				b.logger.Error("Failed to report throughput metrics.", zap.Error(err))
			}
		case <-b.doneChan:
			return
		}
	}
}

func (b *bindplaneExtension) reportMetrics() error {
	m := b.ctmr.OTLPMeasurements(b.cfg.ExtraMeasurementsAttributes)

	// Send metrics as snappy-encoded otlp proto
	marshaller := pmetric.ProtoMarshaler{}
	marshalled, err := marshaller.MarshalMetrics(m)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}

	encoded := snappy.Encode(nil, marshalled)
	for {
		sendingChannel, err := b.customCapabilityHandler.SendMessage(measurements.ReportMeasurementsType, encoded)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, types.ErrCustomMessagePending):
			<-sendingChannel
			continue
		default:
			return fmt.Errorf("send custom message: %w", err)
		}
	}
}

func (b *bindplaneExtension) Shutdown(ctx context.Context) error {
	close(b.doneChan)

	waitgroupDone := make(chan struct{})
	go func() {
		defer close(waitgroupDone)
		b.wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-waitgroupDone: // OK
	}

	if b.customCapabilityHandler != nil {
		b.customCapabilityHandler.Unregister()
	}

	return nil
}
