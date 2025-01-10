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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		shouldError bool
	}{
		{
			name: "Valid Config",
			config: Config{
				SampleInterval:   time.Minute,
				MaxWindowAge:     time.Hour,
				ZScoreThreshold:  3.0,
				MADThreshold:     3.5,
				EmergencyMaxSize: 1000,
			},
			shouldError: false,
		},
		{
			name: "Invalid Sample Interval - Too Short",
			config: Config{
				SampleInterval:   30 * time.Second,
				MaxWindowAge:     time.Hour,
				ZScoreThreshold:  3.0,
				MADThreshold:     3.5,
				EmergencyMaxSize: 1000,
			},
			shouldError: true,
		},
		{
			name: "Invalid Window Age - Too Short",
			config: Config{
				SampleInterval:   time.Minute,
				MaxWindowAge:     30 * time.Minute,
				ZScoreThreshold:  3.0,
				MADThreshold:     3.5,
				EmergencyMaxSize: 1000,
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDetector_CountLogs(t *testing.T) {
	config := createDefaultConfig().(*Config)
	logger := zap.NewNop()
	nextConsumer := &consumertest.LogsSink{}

	detector := newDetector(config, logger, nextConsumer)

	tests := []struct {
		name     string
		logs     func() plog.Logs
		expected int64
	}{
		{
			name: "Empty Logs",
			logs: func() plog.Logs {
				return plog.NewLogs()
			},
			expected: 0,
		},
		{
			name: "Single Log Record",
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				rl := logs.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()
				sl.LogRecords().AppendEmpty()
				return logs
			},
			expected: 1,
		},
		{
			name: "Multiple Log Records",
			logs: func() plog.Logs {
				logs := plog.NewLogs()
				rl := logs.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()
				sl.LogRecords().AppendEmpty()
				sl.LogRecords().AppendEmpty()
				return logs
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := detector.countLogs(tt.logs())
			assert.Equal(t, tt.expected, count)
		})
	}
}

func TestDetector_TakeSample(t *testing.T) {
	config := &Config{
		SampleInterval:   time.Minute,
		MaxWindowAge:     time.Hour,
		ZScoreThreshold:  3.0,
		MADThreshold:     3.5,
		EmergencyMaxSize: 1000,
	}
	logger := zap.NewNop()
	nextConsumer := &consumertest.LogsSink{}

	detector := newDetector(config, logger, nextConsumer)

	// Set initial state
	now := time.Now()
	detector.currentBucket.start = now.Add(-2 * time.Minute)
	detector.currentBucket.count = 120 // 60 logs per minute

	detector.takeSample(now)

	assert.Equal(t, 1, len(detector.rateHistory))
	assert.InDelta(t, 60.0, detector.rateHistory[0].rate, 0.1) // Expect ~60 logs/minute
	assert.Equal(t, int64(0), detector.currentBucket.count)    // Bucket should be reset
	assert.Equal(t, now, detector.currentBucket.start)         // New bucket should start
}

func TestDetector_CheckForAnomaly(t *testing.T) {
	config := &Config{
		SampleInterval:   time.Minute,
		MaxWindowAge:     time.Hour,
		ZScoreThreshold:  3.0,
		MADThreshold:     3.5,
		EmergencyMaxSize: 1000,
	}
	logger := zap.NewNop()
	nextConsumer := &consumertest.LogsSink{}

	detector := newDetector(config, logger, nextConsumer)

	// Setup baseline data (normal rate around 100 logs/minute)
	baselineTime := time.Now().Add(-30 * time.Minute)
	for i := 0; i < 29; i++ {
		detector.rateHistory = append(detector.rateHistory, Sample{
			timestamp: baselineTime.Add(time.Duration(i) * time.Minute),
			rate:      100.0 + float64(i%5), // Small variations
		})
	}

	tests := []struct {
		name          string
		currentRate   float64
		expectAnomaly bool
		anomalyType   string
	}{
		{
			name:          "Normal Rate",
			currentRate:   102.0,
			expectAnomaly: false,
		},
		{
			name:          "Spike Anomaly",
			currentRate:   500.0,
			expectAnomaly: true,
			anomalyType:   "Spike",
		},
		{
			name:          "Drop Anomaly",
			currentRate:   10.0,
			expectAnomaly: true,
			anomalyType:   "Drop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add current sample
			detector.rateHistory = append(detector.rateHistory, Sample{
				timestamp: time.Now(),
				rate:      tt.currentRate,
			})

			anomaly := detector.checkForAnomaly()
			if tt.expectAnomaly {
				require.NotNil(t, anomaly)
				assert.Equal(t, tt.anomalyType, anomaly.anomalyType)
			} else {
				assert.Nil(t, anomaly)
			}

			// Remove the test sample for next iteration
			detector.rateHistory = detector.rateHistory[:len(detector.rateHistory)-1]
		})
	}
}

func TestCalculateStatistics(t *testing.T) {
	tests := []struct {
		name   string
		rates  []float64
		expect func(*testing.T, Statistics)
	}{
		{
			name:  "Empty Rates",
			rates: []float64{},
			expect: func(t *testing.T, stats Statistics) {
				assert.Equal(t, 0.0, stats.mean)
				assert.Equal(t, 0.0, stats.stdDev)
			},
		},
		{
			name:  "Normal Distribution",
			rates: []float64{1, 2, 3, 4, 5},
			expect: func(t *testing.T, stats Statistics) {
				assert.Equal(t, 3.0, stats.mean)
				assert.InDelta(t, 1.414, stats.stdDev, 0.01)
				assert.Equal(t, 3.0, stats.median)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := calculateStatistics(tt.rates)
			tt.expect(t, stats)
		})
	}
}

func TestDetector_Integration(t *testing.T) {
	config := &Config{
		SampleInterval:   time.Minute,
		MaxWindowAge:     time.Hour,
		ZScoreThreshold:  3.0,
		MADThreshold:     3.5,
		EmergencyMaxSize: 1000,
	}

	nextConsumer := &consumertest.LogsSink{}
	logger := zap.NewNop()
	detector := newDetector(config, logger, nextConsumer)

	err := detector.Start(context.Background(), nil)
	require.NoError(t, err)
	defer detector.Shutdown(context.Background())

	baseTime := time.Now()

	// Helper function to simulate log ingestion and force sampling
	simulateLogIngestion := func(logCount int, timestamp time.Time) {
		detector.stateLock.Lock()
		detector.lastSampleTime = timestamp.Add(-2 * time.Minute)
		detector.currentBucket.start = timestamp.Add(-time.Minute)
		detector.stateLock.Unlock()

		// Send logs through the pipeline
		logs := generateTestLogs(logCount)
		err := detector.ConsumeLogs(context.Background(), logs)
		require.NoError(t, err)

		// Force sample taking
		detector.stateLock.Lock()
		detector.takeSample(timestamp)
		detector.stateLock.Unlock()
	}

	// Record initial number of logs
	initialLogCount := nextConsumer.LogRecordCount()

	// Simulate baseline traffic (100 logs/minute for 30 minutes)
	for i := 0; i < 30; i++ {
		simulateLogIngestion(100, baseTime.Add(time.Duration(i)*time.Minute))
	}

	// Verify no anomalies during baseline (should only see the original logs passing through)
	baselineLogCount := nextConsumer.LogRecordCount() - initialLogCount
	assert.Equal(t, 3000, baselineLogCount, "Should have only the baseline logs")

	// Reset the log sink to only catch anomalies
	nextConsumer.Reset()

	// Simulate a sudden spike (1000 logs/minute)
	simulateLogIngestion(1000, baseTime.Add(31*time.Minute))

	// Give a small amount of time for async processing
	time.Sleep(100 * time.Millisecond)

	// We expect to see the 1000 logs plus 1 anomaly log
	assert.Equal(t, 1001, nextConsumer.LogRecordCount(), "Should have spike logs plus one anomaly")

	// Verify anomaly content
	logs := nextConsumer.AllLogs()
	var anomalyLog plog.Logs
	for _, log := range logs {
		rl := log.ResourceLogs()
		if rl.Len() > 0 {
			sl := rl.At(0).ScopeLogs()
			if sl.Len() > 0 {
				lr := sl.At(0).LogRecords()
				if lr.Len() > 0 {
					// Check if this is an anomaly log by looking for anomaly attributes
					attrs := lr.At(0).Attributes()
					if _, exists := attrs.Get("anomaly.type"); exists {
						anomalyLog = log
						break
					}
				}
			}
		}
	}

	require.NotNil(t, anomalyLog, "Should have found an anomaly log")

	// Verify anomaly attributes
	record := anomalyLog.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := record.Attributes()

	anomalyType, exists := attrs.Get("anomaly.type")
	require.True(t, exists)
	assert.Equal(t, "Spike", anomalyType.Str())

	currentRate, exists := attrs.Get("anomaly.current_rate")
	require.True(t, exists)
	assert.InDelta(t, 1000.0, currentRate.Double(), 0.1)
}

// Helper to verify anomaly record
func verifyAnomalyRecord(t *testing.T, record plog.LogRecord, expectedType string, expectedRate float64) {
	attrs := record.Attributes()

	anomalyType, exists := attrs.Get("anomaly.type")
	require.True(t, exists, "anomaly.type should exist")
	assert.Equal(t, expectedType, anomalyType.Str())

	currentRate, exists := attrs.Get("anomaly.current_rate")
	require.True(t, exists, "anomaly.current_rate should exist")
	assert.InDelta(t, expectedRate, currentRate.Double(), 0.1)

	// Verify other required attributes exist
	_, exists = attrs.Get("anomaly.z_score")
	require.True(t, exists, "anomaly.z_score should exist")

	_, exists = attrs.Get("anomaly.mad_score")
	require.True(t, exists, "anomaly.mad_score should exist")

	_, exists = attrs.Get("baseline.mean")
	require.True(t, exists, "baseline.mean should exist")
}

// Helper to get the first log record from LogsSink
func getFirstLogRecord(t *testing.T, sink *consumertest.LogsSink) plog.LogRecord {
	logs := sink.AllLogs()
	require.Equal(t, 1, len(logs), "Should have exactly one batch of logs")

	rl := logs[0].ResourceLogs()
	require.Equal(t, 1, rl.Len(), "Should have exactly one resource log")

	sl := rl.At(0).ScopeLogs()
	require.Equal(t, 1, sl.Len(), "Should have exactly one scope log")

	lr := sl.At(0).LogRecords()
	require.Equal(t, 1, lr.Len(), "Should have exactly one log record")

	return lr.At(0)
}

// Helper function to generate test logs
func generateTestLogs(count int) plog.Logs {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()

	for i := 0; i < count; i++ {
		sl.LogRecords().AppendEmpty()
	}

	return logs
}
