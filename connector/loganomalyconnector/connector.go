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

	// Buffer for storing recent anomalies
	anomalyBuffer []*plog.Logs

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

func (d *Detector) Start(_ context.Context, host component.Host) error {
	ticker := time.NewTicker(d.config.SampleInterval)

	go func() {
		for {
			select {
			case <-d.ctx.Done():
				return
			case <-ticker.C:
				d.checkAndUpdateAnomalies()
				// if err := p.exportAnomalies(ctx); err != nil {
				// 	p.logger.Error("Failed to export anomalies", zap.Error(err))
				// }
			}
		}
	}()

	return nil
}

func (d *Detector) Shutdown(_ context.Context) error {
	d.cancel()
	return nil
}

func (d *Detector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

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

// exportAnomalies sends the logs that are in the anomaly buffer to the next consumer
func (d *Detector) exportAnomalies(ctx context.Context) error {
	d.stateLock.Lock()
	defer d.stateLock.Unlock()

	if len(d.anomalyBuffer) == 0 {
		return nil
	}

	for _, anomalyLog := range d.anomalyBuffer {
		if err := d.nextConsumer.ConsumeLogs(ctx, *anomalyLog); err != nil {
			d.logger.Error("Failed to export anomaly log", zap.Error(err))
			return err
		}
	}

	d.anomalyBuffer = nil

	return nil
}
