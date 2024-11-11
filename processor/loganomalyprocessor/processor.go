package loganomalyprocessor

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

var _ processor.Logs = (*Processor)(nil)

type Sample struct {
	timestamp time.Time
	rate      float64
}

type Statistics struct {
	mean    float64
	stdDev  float64
	median  float64
	mad     float64
	samples []float64
}

type AnomalyStat struct {
	anomalyType    string
	baselineStats  Statistics
	currentRate    float64
	zScore         float64
	madScore       float64
	percentageDiff float64
}

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

func (p *Processor) Start(_ context.Context, _ component.Host) error {
	ticker := time.NewTicker(p.config.SampleInterval)

	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				p.checkAndUpdateMetrics()

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

// takeSample calculates and stores a new rate sample
func (p *Processor) takeSample(now time.Time) {
	duration := now.Sub(p.currentBucket.start).Minutes()
	if duration < (1.0 / 60.0) {
		return
	}
	rate := float64(p.currentBucket.count) / duration

	p.rateHistory = append(p.rateHistory, Sample{
		timestamp: now,
		rate:      rate,
	})

	p.currentBucket.count = 0
	p.currentBucket.start = now
	p.lastSampleTime = now

	p.pruneLogs()
	if anomaly := p.checkForAnomaly(); anomaly != nil {
		p.logAnomaly(anomaly)
	}
}

// pruneLogs performs cleanup on log count buffer
func (p *Processor) pruneLogs() {
	if len(p.rateHistory) == 0 {
		return
	}

	cutoffTime := time.Now().Add(-p.config.MaxWindowAge)
	idx := sort.Search(len(p.rateHistory), func(i int) bool {
		return p.rateHistory[i].timestamp.After(cutoffTime)
	})
	if idx > 0 {
		p.rateHistory = p.rateHistory[idx:]
	}

	// in the case we have more logs than specified for our buffer
	if len(p.rateHistory) > p.config.EmergencyMaxSize {
		excess := len(p.rateHistory) - p.config.EmergencyMaxSize
		p.rateHistory = p.rateHistory[excess:]
		p.logger.Warn("emergency max buffer was exceeded, purge was performed",
			zap.Int("samples_removed", excess))
	}
}

// Calculate statistics for the current window
func calculateStatistics(rates []float64) Statistics {
	if len(rates) == 0 {
		return Statistics{}
	}

	// Calculate mean
	var sum float64
	for _, rate := range rates {
		sum += rate
	}
	mean := sum / float64(len(rates))

	// Calculate standard deviation
	var sumSquaredDiff float64
	for _, rate := range rates {
		diff := rate - mean
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(len(rates)))

	// Calculate median
	sortedRates := make([]float64, len(rates))
	copy(sortedRates, rates)
	sort.Float64s(sortedRates)
	median := sortedRates[len(sortedRates)/2]

	// Calculate MAD
	deviations := make([]float64, len(rates))
	for i, rate := range rates {
		deviations[i] = math.Abs(rate - median)
	}
	sort.Float64s(deviations)
	mad := deviations[len(deviations)/2] * 1.4826

	return Statistics{
		mean:    mean,
		stdDev:  stdDev,
		median:  median,
		mad:     mad,
		samples: rates,
	}
}

// checkForAnomaly performs the anomaly detection
func (p *Processor) checkForAnomaly() *AnomalyStat {
	if len(p.rateHistory) < 1 { // central limit theorem here :) Im proud of this
		return nil
	}

	currentRate := p.rateHistory[len(p.rateHistory)-1].rate

	rates := make([]float64, len(p.rateHistory)-1)
	for i, sample := range p.rateHistory[:len(p.rateHistory)-1] {
		rates[i] = sample.rate
	}

	stats := calculateStatistics(rates)
	if stats.stdDev == 0 || stats.mad == 0 {
		return nil
	}

	zScore := (currentRate - stats.mean) / stats.stdDev
	madScore := (currentRate - stats.median) / stats.mad
	percentageDiff := ((currentRate - stats.mean) / stats.mean) * 100

	// Check for anomaly using both Z-score and MAD
	if math.Abs(zScore) > p.config.ZScoreThreshold || math.Abs(madScore) > p.config.MADThreshold {
		anomalyType := "Drop"
		if currentRate > stats.mean {
			anomalyType = "Spike"
		}

		return &AnomalyStat{
			anomalyType:    anomalyType,
			baselineStats:  stats,
			currentRate:    currentRate,
			zScore:         zScore,
			madScore:       madScore,
			percentageDiff: math.Abs(percentageDiff),
		}
	}

	return nil
}

// checkAndUpdateMetrics runs periodically to check for anomalies even when no logs are received
func (p *Processor) checkAndUpdateMetrics() {
	p.stateLock.Lock()
	defer p.stateLock.Unlock()

	now := time.Now()
	if now.Sub(p.lastSampleTime) >= p.config.SampleInterval {
		p.takeSample(now)
	}
}

// logAnomaly logs detected anomalies
func (p *Processor) logAnomaly(anomaly *AnomalyStat) {
	if anomaly == nil {
		return
	}

	icon := "📈"
	if anomaly.anomalyType == "Drop" {
		icon = "📉"
	}

	p.logger.Info("Log anomaly detected",
		zap.String("anomaly_type", icon+" "+anomaly.anomalyType),
		zap.Float64("current_rate", anomaly.currentRate),
		zap.Float64("baseline_mean", anomaly.baselineStats.mean),
		zap.Float64("baseline_median", anomaly.baselineStats.median),
		zap.Float64("z_score", anomaly.zScore),
		zap.Float64("mad_score", anomaly.madScore),
		zap.Float64("deviation_percentage", anomaly.percentageDiff),
	)
	p.logger.Sync()
}
