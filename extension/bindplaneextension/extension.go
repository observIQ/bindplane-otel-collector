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
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang/snappy"
	"github.com/observiq/bindplane-otel-collector/internal/measurements"
	"github.com/observiq/bindplane-otel-collector/internal/topology"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type bindplaneExtension struct {
	logger                            *zap.Logger
	cfg                               *Config
	measurementsRegistry              *measurements.ResettableThroughputMeasurementsRegistry
	topologyRegistry                  *topology.ResettableTopologyRegistry
	customCapabilityHandlerThroughput opampcustommessages.CustomCapabilityHandler
	customCapabilityHandlerTopology   opampcustommessages.CustomCapabilityHandler

	doneChan chan struct{}
	wg       *sync.WaitGroup
}

func newBindplaneExtension(logger *zap.Logger, cfg *Config) *bindplaneExtension {
	return &bindplaneExtension{
		logger:               logger,
		cfg:                  cfg,
		measurementsRegistry: measurements.NewResettableThroughputMeasurementsRegistry(false),
		topologyRegistry:     topology.NewResettableTopologyRegistry(),
		doneChan:             make(chan struct{}),
		wg:                   &sync.WaitGroup{},
	}
}

func (b *bindplaneExtension) Start(_ context.Context, host component.Host) error {
	var emptyComponentID component.ID

	// Set up custom capabilities if enabled
	if b.cfg.OpAMP != emptyComponentID {
		err := b.setupCustomCapabilities(host)
		if err != nil {
			return fmt.Errorf("setup capability handler: %w", err)
		}

		if b.cfg.MeasurementsInterval > 0 {
			b.wg.Add(1)
			go b.reportMetricsLoop()
		}

		b.wg.Add(1)
		go b.reportTopologyLoop()
	}

	return nil
}

func (b *bindplaneExtension) RegisterThroughputMeasurements(processorID string, measurements *measurements.ThroughputMeasurements) error {
	return b.measurementsRegistry.RegisterThroughputMeasurements(processorID, measurements)
}

func (b *bindplaneExtension) RegisterTopologyState(processorID string, topology *topology.TopoState) error {
	return b.topologyRegistry.RegisterTopologyState(processorID, topology)
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
	if b.cfg.MeasurementsInterval > 0 {
		b.customCapabilityHandlerThroughput, err = registry.Register(measurements.ReportMeasurementsV1Capability)
		if err != nil {
			return fmt.Errorf("register custom measurements capability: %w", err)
		}
	}

	b.customCapabilityHandlerTopology, err = registry.Register(topology.ReportTopologyCapability)
	if err != nil {
		return fmt.Errorf("register custom topology capability: %w", err)
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
	m := b.measurementsRegistry.OTLPMeasurements(b.cfg.ExtraMeasurementsAttributes)

	// Send metrics as snappy-encoded otlp proto
	marshaller := pmetric.ProtoMarshaler{}
	marshalled, err := marshaller.MarshalMetrics(m)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}

	encoded := snappy.Encode(nil, marshalled)
	for {
		sendingChannel, err := b.customCapabilityHandlerThroughput.SendMessage(measurements.ReportMeasurementsType, encoded)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, types.ErrCustomMessagePending):
			<-sendingChannel
			continue
		default:
			return fmt.Errorf("send custom throughput message: %w", err)
		}
	}
}

func (b *bindplaneExtension) reportTopologyLoop() {
	defer b.wg.Done()

	var topologyInterval time.Duration
	select {
	case <-b.doneChan:
		return
	case topologyInterval = <-b.topologyRegistry.SetIntervalChan():
		if topologyInterval <= 0 {
			return
		}
	}

	t := time.NewTicker(topologyInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			err := b.reportTopology()
			if err != nil {
				b.logger.Error("Failed to report topology.", zap.Error(err))
			}
		case <-b.doneChan:
			return
		}
	}
}

func (b *bindplaneExtension) reportTopology() error {
	ts := b.topologyRegistry.TopologyInfos()

	// Send topology state snappy-encoded
	marshalled, err := json.Marshal(ts)
	if err != nil {
		return fmt.Errorf("marshal topology state: %w", err)
	}

	encoded := snappy.Encode(nil, marshalled)
	for {
		sendingChannel, err := b.customCapabilityHandlerTopology.SendMessage(topology.ReportTopologyType, encoded)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, types.ErrCustomMessagePending):
			<-sendingChannel
			continue
		default:
			return fmt.Errorf("send custom topology message: %w", err)
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

	if b.customCapabilityHandlerThroughput != nil {
		b.customCapabilityHandlerThroughput.Unregister()
	}

	if b.customCapabilityHandlerTopology != nil {
		b.customCapabilityHandlerTopology.Unregister()
	}

	return nil
}
