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

package loganomalyconnector

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// Detector implements the log anomaly detection connector.
// It maintains a rolling window of log rate samples and uses statistical analysis
// to detect anomalies in log throughput. The detector processes incoming logs,
// calculates rates, and generates alerts when anomalous patterns are detected.
type Detector struct {
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

	nextConsumer consumer.Logs
}

func newDetector(config *Config, logger *zap.Logger, nextConsumer consumer.Logs) *Detector {
	ctx, cancel := context.WithCancel(context.Background())

	logger = logger.WithOptions(zap.Development())

	return &Detector{
		ctx:          ctx,
		cancel:       cancel,
		logger:       logger,
		config:       config,
		stateLock:    sync.Mutex{},
		rateHistory:  make([]Sample, 0, config.MaxWindowAge/config.SampleInterval),
		nextConsumer: nextConsumer,
	}
}

// Start begins the anomaly detection process, sampling at intervals specified in the config.
// It launches a background goroutine that periodically checks for and updates anomalies.
func (d *Detector) Start(_ context.Context, _ component.Host) error {
	ticker := time.NewTicker(d.config.SampleInterval)

	go func() {
		for {
			select {
			case <-d.ctx.Done():
				return
			case <-ticker.C:
				d.checkAndUpdateAnomalies()
			}
		}
	}()

	return nil
}

// Shutdown stops the detector's operations by canceling its context.
// It cleans up any resources and stops the background sampling process.
func (d *Detector) Shutdown(_ context.Context) error {
	d.cancel()
	return nil
}

// Capabilities returns the consumer capabilities of the detector.
// It indicates that this detector does not mutate the data it processes.
func (d *Detector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeLogs processes incoming log data, counting the logs and maintaining time-based sampling buckets.
// The logs are then forwarded to the next consumer in the pipeline.
func (d *Detector) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	d.stateLock.Lock()
	defer d.stateLock.Unlock()

	logCount := d.countLogs(ld)

	if d.currentBucket.start.IsZero() {
		d.currentBucket.start = time.Now()
	}

	d.currentBucket.count += logCount

	now := time.Now()
	if now.Sub(d.lastSampleTime) >= d.config.SampleInterval {
		d.takeSample(now)
	}

	return d.nextConsumer.ConsumeLogs(ctx, ld)
}

// countLogs counts the number of log records in the input
func (d *Detector) countLogs(ld plog.Logs) int64 {
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
func (d *Detector) checkAndUpdateAnomalies() {
	d.stateLock.Lock()
	defer d.stateLock.Unlock()

	now := time.Now()
	if now.Sub(d.lastSampleTime) >= d.config.SampleInterval {
		d.takeSample(now)
	}
}
