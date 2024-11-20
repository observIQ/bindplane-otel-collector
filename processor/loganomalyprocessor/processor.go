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

package loganomalyprocessor

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

var _ processor.Logs = (*Processor)(nil)

type Processor struct {
	ctx    context.Context
	cancel context.CancelFunc
	logger *zap.Logger

	stateLock sync.Mutex
	config    *Config

	// Rolling window of rate samples
	rateHistory []Sample

	// Current bucket for accumulating logs
	currentBucket struct {
		count int64
		start time.Time
	}
	lastSampleTime time.Time

	// Buffer for storing recent anomalies
	anomalyBuffer     []*plog.Logs
	anomalyBufferSize int // Maximum number of anomalies to store

	nextConsumer consumer.Logs
}

func newProcessor(config *Config, logger *zap.Logger, nextConsumer consumer.Logs) *Processor {
	ctx, cancel := context.WithCancel(context.Background())

	logger = logger.WithOptions(zap.Development())

	return &Processor{

		ctx:          ctx,
		cancel:       cancel,
		logger:       logger,
		config:       config,
		stateLock:    sync.Mutex{},
		rateHistory:  make([]Sample, 0, config.MaxWindowAge/config.SampleInterval),
		nextConsumer: nextConsumer,
	}
}

func (p *Processor) Start(_ context.Context, host component.Host) error {
	ticker := time.NewTicker(p.config.SampleInterval)

	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				p.checkAndUpdateAnomalies()
				if err := p.exportAnomalies(p.ctx); err != nil {
					p.logger.Error("Failed to export anomalies", zap.Error(err))
				}
			}
		}
	}()

	return nil
}

func (p *Processor) Shutdown(_ context.Context) error {
	p.cancel()
	return nil
}

func (p *Processor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true} // i should prob change this to false TODO
}

func (p *Processor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	p.stateLock.Lock()
	defer p.stateLock.Unlock()

	logCount := p.countLogs(ld)

	if p.currentBucket.start.IsZero() {
		p.currentBucket.start = time.Now()
	}

	p.currentBucket.count += logCount

	now := time.Now()
	if now.Sub(p.lastSampleTime) >= p.config.SampleInterval {
		p.takeSample(now)
	}

	return p.nextConsumer.ConsumeLogs(ctx, ld)
}

// countLogs counts the number of log records in the input
func (p *Processor) countLogs(ld plog.Logs) int64 {
	var count int64
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		sls := rls.At(i).ScopeLogs()
		for j := 0; j < sls.Len(); j++ {
			count += int64(sls.At(j).LogRecords().Len())
		}
	}
	return count
}

// checkAndUpdateMetrics runs periodically to check for anomalies even when no logs are received
func (p *Processor) checkAndUpdateAnomalies() {
	p.stateLock.Lock()
	defer p.stateLock.Unlock()

	now := time.Now()
	if now.Sub(p.lastSampleTime) >= p.config.SampleInterval {
		p.takeSample(now)
	}
}

// exportAnomalies sends the logs that are in the anomaly buffer to the next consumer
func (p *Processor) exportAnomalies(ctx context.Context) error {
	p.stateLock.Lock()
	defer p.stateLock.Unlock()

	if len(p.anomalyBuffer) == 0 {
		return nil
	}

	for _, anomalyLog := range p.anomalyBuffer {
		if err := p.nextConsumer.ConsumeLogs(ctx, *anomalyLog); err != nil {
			p.logger.Error("Failed to export anomaly log", zap.Error(err))
			return err
		}
	}

	p.anomalyBuffer = nil

	return nil
}
