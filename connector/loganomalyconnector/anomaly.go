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

// Package loganomalyconnector provides an OpenTelemetry collector connector that detects
// anomalies in log throughput using statistical analysis. It monitors the rate of incoming
// logs and generates alerts when significant deviations from the baseline are detected,
// using both Z-score and Median Absolute Deviation (MAD) methods for anomaly detection.
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

// Statistics holds statistical measures calculated from a set of log rate samples.
// These statistics are used to establish a baseline and detect anomalies in log throughput.
type Statistics struct {
	mean    float64
	stdDev  float64
	median  float64
	mad     float64
	samples []int64
}

// AnomalyStat contains information about a detected log throughput anomaly,
// including the type of anomaly, baseline statistics, and deviation metrics.
type AnomalyStat struct {
	anomalyType    string
	baselineStats  Statistics
	currentCount   int64
	zScore         float64
	madScore       float64
	percentageDiff float64
	timestamp      time.Time
}

// Calculate statistics for the current window
func calculateStatistics(counts []int64) Statistics {
	if len(counts) == 0 {
		return Statistics{}
	}

	// Calculate mean
	var sum float64
	for _, count := range counts {
		sum += float64(count)
	}

	mean := sum / float64(len(counts))

	// Calculate standard deviation
	var sumSquaredDiff float64
	for _, count := range counts {
		diff := float64(count) - mean
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(len(counts)))

	// Calculate median
	sortedCounts := make([]int64, len(counts))
	copy(sortedCounts, counts)
	sort.Slice(sortedCounts, func(i int, j int) bool {
		return sortedCounts[i] < sortedCounts[j]
	})
	median := float64(sortedCounts[len(sortedCounts)/2])

	// Calculate MAD
	deviations := make([]float64, len(counts))
	for i, count := range counts {
		deviations[i] = math.Abs(float64(count) - median)
	}
	sort.Float64s(deviations)
	mad := deviations[len(deviations)/2] * 1.4826

	return Statistics{
		mean:    mean,
		stdDev:  stdDev,
		median:  median,
		mad:     mad,
		samples: counts,
	}
}

// checkForAnomaly performs the anomaly detection
func (d *Detector) checkForAnomaly() *AnomalyStat {
	if len(d.counts) < 2 {
		return nil
	}

	currentCount := d.counts[len(d.counts)-1]
	historicalCounts := make([]int64, len(d.counts)-1)
	copy(historicalCounts, d.counts[:len(d.counts)-1])

	stats := calculateStatistics(historicalCounts)

	if stats.stdDev == 0 {
		if float64(currentCount) != stats.mean {
			percentageDiff := ((float64(currentCount) - stats.mean) / stats.mean) * 100
			anomalyType := "Drop"
			if float64(currentCount) > stats.mean {
				anomalyType = "Spike"
			}

			return &AnomalyStat{
				anomalyType:    anomalyType,
				baselineStats:  stats,
				currentCount:   currentCount,
				zScore:         0, // Not meaningful when stdDev is 0
				madScore:       0, // Not meaningful when MAD is 0
				percentageDiff: math.Abs(percentageDiff),
				timestamp:      d.lastWindowEndTime,
			}
		}
		return nil
	}
	zScore := (float64(currentCount) - stats.mean) / stats.stdDev
	madScore := (float64(currentCount) - stats.median) / stats.mad
	percentageDiff := ((float64(currentCount) - stats.mean) / stats.mean) * 100

	// Check for anomaly using both Z-score and MAD
	if math.Abs(zScore) > d.config.ZScoreThreshold || math.Abs(madScore) > d.config.MADThreshold {
		anomalyType := "Drop"
		if float64(currentCount) > stats.mean {
			anomalyType = "Spike"
		}

		return &AnomalyStat{
			anomalyType:    anomalyType,
			baselineStats:  stats,
			currentCount:   currentCount,
			zScore:         zScore,
			madScore:       madScore,
			percentageDiff: math.Abs(percentageDiff),
			timestamp:      d.lastWindowEndTime,
		}
	}

	return nil
}

// logAnomaly logs detected anomalies
func (d *Detector) logAnomaly(anomaly *AnomalyStat) {
	if anomaly == nil {
		return
	}

	// Create a new Logs instance
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()

	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(anomaly.timestamp))

	logRecord.SetSeverityText("WARNING")
	logRecord.SetSeverityNumber(plog.SeverityNumberWarn)

	logRecord.Body().SetStr(fmt.Sprintf("Log Anomaly Detected: %s", anomaly.anomalyType))

	// Add all anomaly data as attributes
	attrs := logRecord.Attributes()
	attrs.PutStr("anomaly.type", anomaly.anomalyType)
	attrs.PutDouble("anomaly.current_count", float64(anomaly.currentCount))
	attrs.PutDouble("anomaly.z_score", anomaly.zScore)
	attrs.PutDouble("anomaly.mad_score", anomaly.madScore)
	attrs.PutDouble("anomaly.percentage_diff", anomaly.percentageDiff)
	attrs.PutDouble("baseline.mean", anomaly.baselineStats.mean)
	attrs.PutDouble("baseline.std_dev", anomaly.baselineStats.stdDev)
	attrs.PutDouble("baseline.median", anomaly.baselineStats.median)
	attrs.PutDouble("baseline.mad", anomaly.baselineStats.mad)

	d.logger.Info("Log anomaly detected",
		zap.String("anomaly_type", anomaly.anomalyType),
		zap.Float64("current_count", float64(anomaly.currentCount)),
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
}
