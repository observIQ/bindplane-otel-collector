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
	"fmt"
	"math"
	"sort"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

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
	timestamp      time.Time
}

// takeSample calculates and stores a new rate sample
func (d *Detector) takeSample(now time.Time) {
	duration := now.Sub(d.currentBucket.start).Minutes()
	if duration < (1.0 / 60.0) {
		return
	}
	rate := float64(d.currentBucket.count) / duration

	d.rateHistory = append(d.rateHistory, Sample{
		timestamp: now,
		rate:      rate,
	})

	d.currentBucket.count = 0
	d.currentBucket.start = now
	d.lastSampleTime = now

	d.pruneLogs()
	if anomaly := d.checkForAnomaly(); anomaly != nil {
		d.logAnomaly(anomaly)
	}
}

// pruneLogs performs cleanup on log count buffer
func (d *Detector) pruneLogs() {
	if len(d.rateHistory) == 0 {
		return
	}

	cutoffTime := time.Now().Add(-d.config.MaxWindowAge)
	idx := sort.Search(len(d.rateHistory), func(i int) bool {
		return d.rateHistory[i].timestamp.After(cutoffTime)
	})
	if idx > 0 {
		d.rateHistory = d.rateHistory[idx:]
	}

	// in the case we have more logs than specified for our buffer
	if len(d.rateHistory) > d.config.EmergencyMaxSize {
		excess := len(d.rateHistory) - d.config.EmergencyMaxSize
		d.rateHistory = d.rateHistory[excess:]
		d.logger.Warn("emergency max buffer was exceeded, purge was performed",
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
func (d *Detector) checkForAnomaly() *AnomalyStat {
	if len(d.rateHistory) < 1 {
		return nil
	}

	currentRate := d.rateHistory[len(d.rateHistory)-1].rate

	rates := make([]float64, len(d.rateHistory)-1)
	for i, sample := range d.rateHistory[:len(d.rateHistory)-1] {
		rates[i] = sample.rate
	}

	stats := calculateStatistics(rates)
	if stats.stdDev == 0 || stats.mad == 0 {
		return nil
	}

	zScore := (currentRate - stats.mean) / stats.stdDev
	madScore := (currentRate - stats.median) / stats.mad
	percentageDiff := ((currentRate - stats.mean) / stats.mean) * 100

	anomalyTimestamp := d.rateHistory[len(d.rateHistory)-1].timestamp

	// Check for anomaly using both Z-score and MAD
	if math.Abs(zScore) > d.config.ZScoreThreshold || math.Abs(madScore) > d.config.MADThreshold {
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
			timestamp:      anomalyTimestamp,
		}
	}

	return nil
}

// logAnomaly logs detected anomalies
func (d *Detector) logAnomaly(anomaly *AnomalyStat) {
	if anomaly == nil {
		return
	}

	icon := "📈"
	if anomaly.anomalyType == "Drop" {
		icon = "📉"
	}

	// Create a new Logs instance
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()

	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(anomaly.timestamp))

	logRecord.SetSeverityText("WARNING")
	logRecord.SetSeverityNumber(plog.SeverityNumberWarn)

	logRecord.Body().SetStr(fmt.Sprintf("%s Log Anomaly Detected: %s", icon, anomaly.anomalyType))

	// Add all anomaly data as attributes
	attrs := logRecord.Attributes()
	attrs.PutStr("anomaly.type", anomaly.anomalyType)
	attrs.PutDouble("anomaly.current_rate", anomaly.currentRate)
	attrs.PutDouble("anomaly.z_score", anomaly.zScore)
	attrs.PutDouble("anomaly.mad_score", anomaly.madScore)
	attrs.PutDouble("anomaly.percentage_diff", anomaly.percentageDiff)
	attrs.PutDouble("baseline.mean", anomaly.baselineStats.mean)
	attrs.PutDouble("baseline.std_dev", anomaly.baselineStats.stdDev)
	attrs.PutDouble("baseline.median", anomaly.baselineStats.median)
	attrs.PutDouble("baseline.mad", anomaly.baselineStats.mad)

	d.logger.Info("Log anomaly detected",
		zap.String("anomaly_type", icon+" "+anomaly.anomalyType),
		zap.Float64("current_rate", anomaly.currentRate),
		zap.Float64("baseline_mean", anomaly.baselineStats.mean),
		zap.Float64("baseline_median", anomaly.baselineStats.median),
		zap.Float64("z_score", anomaly.zScore),
		zap.Float64("mad_score", anomaly.madScore),
		zap.Float64("deviation_percentage", anomaly.percentageDiff),
	)
	// Create a new context with timeout for exporting
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := d.nextConsumer.ConsumeLogs(ctx, logs); err != nil {
		d.logger.Error("Failed to export anomaly log", zap.Error(err))
	}
	// p.anomalyBuffer = append(p.anomalyBuffer, &logs)
	// if len(p.anomalyBuffer) > p.config.EmergencyMaxSize {
	// 	// Remove oldest anomaly when buffer is full
	// 	p.anomalyBuffer = p.anomalyBuffer[1:]
	// }

	// if len(p.anomalyBuffer) == 0 {
	// 	p.stateLock.Lock()
	// 	defer p.stateLock.Unlock()
	// 	return
	// }

	// for _, anomalyLog := range p.anomalyBuffer {
	// 	if err := p.nextConsumer.ConsumeLogs(p.ctx, *anomalyLog); err != nil {
	// 		p.logger.Error("Failed to export anomaly log", zap.Error(err))
	// 	}
	// }

	// p.anomalyBuffer = nil

	// p.stateLock.Lock()
	// defer p.stateLock.Unlock()
}
