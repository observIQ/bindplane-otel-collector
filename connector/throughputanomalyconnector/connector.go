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

package throughputanomalyconnector

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type logBatch struct {
	logs    plog.Logs
	ctx     context.Context
	errChan chan error
}

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

	counts       []int64
	currentCount atomic.Int64

	lastWindowEndTime time.Time

	logChan      chan logBatch
	nextConsumer consumer.Logs
	wg           sync.WaitGroup
}

func newDetector(config *Config, logger *zap.Logger, nextConsumer consumer.Logs) *Detector {
	numWindows := int(config.MaxWindowAge.Minutes())

	return &Detector{
		logger:            logger,
		config:            config,
		stateLock:         sync.Mutex{},
		nextConsumer:      nextConsumer,
		logChan:           make(chan logBatch, 1000),
		counts:            make([]int64, numWindows),
		lastWindowEndTime: time.Now().Truncate(time.Minute),
	}
}

// Start begins the anomaly detection process, sampling at intervals specified in the config.
// It launches a background goroutine that periodically checks for and updates anomalies.
func (d *Detector) Start(ctx context.Context, _ component.Host) error {
	d.ctx, d.cancel = context.WithCancel(ctx)

	ticker := time.NewTicker(d.config.SampleInterval)

	d.wg.Add(1)

	go func() {
		defer func() {
			ticker.Stop()
			d.wg.Done()
		}()

		for {
			select {
			case <-d.ctx.Done():
				return
			case batch := <-d.logChan:
				d.processLogBatch(batch)
			case <-ticker.C:
				d.analyzeTimeWindow()
			}
		}
	}()

	return nil
}

// Shutdown stops the detector's operations by canceling its context.
// It cleans up any resources and stops the background sampling process.
func (d *Detector) Shutdown(ctx context.Context) error {
	d.cancel()

	close(d.logChan)

	// Wait for goroutine to finish with timeout from context
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (d *Detector) processLogBatch(batch logBatch) {
	logCount := batch.logs.LogRecordCount()
	d.currentCount.Add(int64(logCount))

	err := d.nextConsumer.ConsumeLogs(batch.ctx, batch.logs)
	batch.errChan <- err
}

func (d *Detector) analyzeTimeWindow() {
	d.stateLock.Lock()
	defer d.stateLock.Unlock()

	now := time.Now()
	currentCount := d.currentCount.Swap(0)

	if len(d.counts) > 0 {
		d.counts[len(d.counts)-1] = currentCount
	}

	// drop any windows that are too old
	maxAge := d.config.MaxWindowAge
	cutoffTime := now.Add(-maxAge)

	// find the first window that is not too old
	var keepIndex int
	for i := range d.counts {
		windowTime := d.lastWindowEndTime.Add(-time.Duration(len(d.counts)-1-i) * time.Minute)
		if windowTime.After(cutoffTime) {
			keepIndex = i
			break
		}
	}

	if keepIndex > 0 {
		d.counts = d.counts[keepIndex:]
	}

	// add windows until we reach current time
	for d.lastWindowEndTime.Add(time.Minute).Before(now) {
		d.counts = append(d.counts, 0)
		d.lastWindowEndTime = d.lastWindowEndTime.Add(time.Minute)
	}

	// Check for anomalies using the fixed window counts
	if anomaly := d.checkForAnomaly(); anomaly != nil {
		d.logAnomaly(anomaly)
	}
}

// Capabilities returns the consumer capabilities of the detector.
// It indicates that this detector does not mutate the data it processes.
func (d *Detector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

// ConsumeLogs processes incoming log data, counting the logs and maintaining time-based sampling buckets.
func (d *Detector) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	errChan := make(chan error, 1)

	select {
	case d.logChan <- logBatch{logs: ld, ctx: ctx, errChan: errChan}:
		return <-errChan
	case <-ctx.Done():
		return ctx.Err()
	}

}
