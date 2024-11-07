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

type AnomalyStat struct {
	anomalyType    string
	baselineRate   float64
	currentRate    float64
	percentageDiff float64
}

type Processor struct {
	ctx    context.Context
	cancel context.CancelFunc
	logger *zap.Logger

	stateLock sync.Mutex

	config *Config

	anomalyFormatter    *AnomalyFormatter
	currentWindowCount  int64
	baselineWindowCount int64
	logTimestamps       []time.Time

	startTime   time.Time
	checkTicker *time.Ticker

	nextConsumer consumer.Logs
}

func newProcessor(config *Config, log *zap.Logger, nextConsumer consumer.Logs) *Processor {
	ctx, cancel := context.WithCancel(context.Background())

	log = log.WithOptions(zap.Development())

	return &Processor{
		ctx:    ctx,
		cancel: cancel,
		logger: log,

		stateLock: sync.Mutex{},

		config: config,

		startTime: time.Now(),

		checkTicker: nil,

		nextConsumer:     nextConsumer,
		anomalyFormatter: newAnomalyFormatter(),
	}
}

func (p *Processor) Start(_ context.Context, _ component.Host) error {
	p.checkTicker = time.NewTicker(p.config.ComparisonWindows.CurrentWindow)
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case <-p.checkTicker.C:
				p.noLogAnomalyCheck()
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
	return consumer.Capabilities{MutatesData: true}
}

func (p *Processor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	p.stateLock.Lock()
	defer p.stateLock.Unlock()

	currentTime := time.Now()

	resLogs := ld.ResourceLogs()
	for i := 0; i < resLogs.Len(); i++ {
		resLog := resLogs.At(i)
		scopeLogs := resLog.ScopeLogs()

		for j := 0; j < scopeLogs.Len(); j++ {
			scopeLog := scopeLogs.At(j)
			logs := scopeLog.LogRecords()

			for k := 0; k < logs.Len(); k++ {

				logTime := logs.At(k).Timestamp().AsTime()

				if currentTime.Sub(logTime) <= p.config.ComparisonWindows.CurrentWindow {
					p.currentWindowCount++
				}
				if currentTime.Sub(logTime) < p.config.ComparisonWindows.BaselineWindow {
					p.baselineWindowCount++
				}

				p.logTimestamps = append(p.logTimestamps, logTime)
			}
		}
	}

	p.pruneLogs(currentTime)

	return p.nextConsumer.ConsumeLogs(ctx, ld)
}

func (p *Processor) pruneLogs(currentTime time.Time) {
	if len(p.logTimestamps) == 0 {
		return
	}

	if currentTime.Sub(p.logTimestamps[0]) <= p.config.ComparisonWindows.BaselineWindow {
		return
	}

	// binary search to find the earliest timestamp that we can prune
	baselineLimit := currentTime.Add(-p.config.ComparisonWindows.BaselineWindow)
	cutoffIndex := sort.Search(len(p.logTimestamps), func(i int) bool {
		return p.logTimestamps[i].After(baselineLimit)
	})

	if cutoffIndex == len(p.logTimestamps) {
		p.logTimestamps = p.logTimestamps[:0]
		p.currentWindowCount = 0
		p.baselineWindowCount = 0
		return
	}

	// Update timestamps slice
	p.logTimestamps = p.logTimestamps[cutoffIndex:]

	// Reset counters
	p.baselineWindowCount = int64(len(p.logTimestamps))

	// Count current window logs
	currentLimit := currentTime.Add(-p.config.ComparisonWindows.CurrentWindow)
	currentIndex := sort.Search(len(p.logTimestamps), func(i int) bool {
		return p.logTimestamps[i].After(currentLimit)
	})
	p.currentWindowCount = int64(len(p.logTimestamps) - currentIndex)

}

func (p *Processor) checkForAnomaly() *AnomalyStat {
	if p.baselineWindowCount == 0 && p.currentWindowCount == 0 {
		return nil
	}

	if time.Since(p.startTime) < p.config.ComparisonWindows.BaselineWindow {
		return nil
	}

	baselineRate := float64(p.baselineWindowCount) / p.config.ComparisonWindows.BaselineWindow.Minutes()
	currentRate := float64(p.currentWindowCount) / p.config.ComparisonWindows.CurrentWindow.Minutes()
	percentageDiff := ((currentRate - baselineRate) / baselineRate) * 100

	if math.Abs(percentageDiff) <= p.config.DeviationThreshold {
		return nil
	}

	anomalyType := "Drop"
	if percentageDiff > 0 {
		anomalyType = "Spike"
	}

	return &AnomalyStat{
		baselineRate:   baselineRate,
		currentRate:    currentRate,
		anomalyType:    anomalyType,
		percentageDiff: math.Abs(percentageDiff),
	}
}

// noLogAnomalyCheck runs at a set interval defined in the config currentWindow, it acts to check in the case no logs come through.
func (p *Processor) noLogAnomalyCheck() {
	p.stateLock.Lock()
	defer p.stateLock.Unlock()

	p.pruneLogs(time.Now())
	if anomaly := p.checkForAnomaly(); anomaly != nil {
		p.logAnomaly(anomaly)
	}
}

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
		zap.Float64("baseline_rate (logs/min)", anomaly.baselineRate),
		zap.Float64("current_rate (logs/min)", anomaly.currentRate),
		zap.Float64("deviation_percentage", anomaly.percentageDiff),
	)
	p.logger.Sync()
}
