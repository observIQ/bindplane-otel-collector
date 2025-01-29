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

	counts            []int64
	lastWindowEndTime time.Time

	logChan      chan logBatch
	nextConsumer consumer.Logs
}

func newDetector(config *Config, logger *zap.Logger, nextConsumer consumer.Logs) *Detector {
	ctx, cancel := context.WithCancel(context.Background())
	numWindows := int(config.MaxWindowAge.Minutes())

	return &Detector{
		ctx:               ctx,
		cancel:            cancel,
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
func (d *Detector) Start(_ context.Context, _ component.Host) error {

	ticker := time.NewTicker(d.config.SampleInterval)

	go func() {
		for {
			select {
			case <-d.ctx.Done():
				return
			case batch := <-d.logChan:
				d.processLogBatch(batch)
			case <-ticker.C:
				d.checkForAnomaly() // check this
			}
		}
	}()

	return nil
}

func (d *Detector) processLogBatch(batch logBatch) {
	now := time.Now()

	// Update windows that have elapsed since last update
	for now.Sub(d.lastWindowEndTime).Minutes() >= 1 {
		// Shift windows and add empty window
		copy(d.counts, d.counts[1:])
		d.counts[len(d.counts)-1] = 0
		d.lastWindowEndTime = d.lastWindowEndTime.Add(time.Minute)
	}

	// Add counts to current window
	logCount := batch.logs.LogRecordCount()
	d.counts[len(d.counts)-1] += int64(logCount)

	// Check for anomalies using the fixed window counts
	if anomaly := d.checkForAnomaly(); anomaly != nil {
		d.logAnomaly(anomaly)
	}
}

// Shutdown stops the detector's operations by canceling its context.
// It cleans up any resources and stops the background sampling process.
func (d *Detector) Shutdown(ctx context.Context) error {
	d.cancel()

	// Wait for the main processing loop to finish
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		close(d.logChan)
		return nil
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
