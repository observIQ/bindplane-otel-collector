// // Copyright observIQ, Inc.
// //
// // Licensed under the Apache License, Version 2.0 (the "License");
// // you may not use this file except in compliance with the License.
// // You may obtain a copy of the License at
// //
// //     http://www.apache.org/licenses/LICENSE-2.0
// //
// // Unless required by applicable law or agreed to in writing, software
// // distributed under the License is distributed on an "AS IS" BASIS,
// // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// // See the License for the specific language governing permissions and
// // limitations under the License.

package throughputanomalyconnector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				AnalysisInterval: time.Minute,
				MaxWindowAge:     time.Hour,
				ZScoreThreshold:  3.0,
				MADThreshold:     3.0,
			},
			expectError: false,
		},
		{
			name: "invalid sample interval - too small",
			config: Config{
				AnalysisInterval: time.Second * 30,
				MaxWindowAge:     time.Hour,
				ZScoreThreshold:  3.0,
				MADThreshold:     3.0,
			},
			expectError: true,
		},
		{
			name: "invalid max window age - too small relative to sample interval",
			config: Config{
				AnalysisInterval: time.Minute,
				MaxWindowAge:     time.Minute * 5,
				ZScoreThreshold:  3.0,
				MADThreshold:     3.0,
			},
			expectError: true,
		},
		{
			name: "invalid threshold - negative z-score",
			config: Config{
				AnalysisInterval: time.Minute,
				MaxWindowAge:     time.Hour,
				ZScoreThreshold:  -1.0,
				MADThreshold:     3.0,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDetector_StatisticsCalculation(t *testing.T) {
	tests := []struct {
		name     string
		counts   []int64
		expected Statistics
	}{
		{
			name:   "empty counts",
			counts: []int64{},
			expected: Statistics{
				mean:    0,
				stdDev:  0,
				median:  0,
				mad:     0,
				samples: []int64(nil),
			},
		},
		{
			name:   "single count",
			counts: []int64{100},
			expected: Statistics{
				mean:    100,
				stdDev:  0,
				median:  100,
				mad:     0,
				samples: []int64{100},
			},
		},
		{
			name:   "multiple counts",
			counts: []int64{100, 200, 300, 400, 500},
			expected: Statistics{
				mean:    300,
				stdDev:  141.42135623731,
				median:  300,
				mad:     148.26,
				samples: []int64{100, 200, 300, 400, 500},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := calculateStatistics(tt.counts)
			assert.InDelta(t, tt.expected.mean, stats.mean, 0.001)
			assert.InDelta(t, tt.expected.stdDev, stats.stdDev, 0.001)
			assert.InDelta(t, tt.expected.median, stats.median, 0.001)
			assert.InDelta(t, tt.expected.mad, stats.mad, 0.001)
			assert.Equal(t, tt.expected.samples, stats.samples)
		})
	}
}

// mockConsumer implements consumer.Logs for testing
type mockConsumer struct {
	consumedLogs []plog.Logs
	consumeError error
}

func (m *mockConsumer) ConsumeLogs(_ context.Context, logs plog.Logs) error {
	if m.consumeError != nil {
		return m.consumeError
	}
	m.consumedLogs = append(m.consumedLogs, logs)
	return nil
}

// Add Capabilities method to satisfy consumer.Logs interface
func (m *mockConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func TestDetector_AnomalyDetection(t *testing.T) {
	config := &Config{
		AnalysisInterval: time.Minute,
		MaxWindowAge:     time.Hour,
		ZScoreThreshold:  2.0,
		MADThreshold:     2.0,
	}

	logger := zap.NewNop()
	mockConsumer := &mockConsumer{}

	t.Run("detect spike", func(t *testing.T) {
		detector := newDetector(config, logger, mockConsumer)

		// Manually set up the counts slice with baseline data
		detector.counts = []int64{100, 100, 100, 100, 100, 1000} // Last value is the spike

		// Run anomaly detection directly
		anomaly := detector.checkForAnomaly()

		require.NotNil(t, anomaly, "Expected to detect an anomaly")
		assert.Equal(t, "Spike", anomaly.anomalyType)
		assert.Equal(t, int64(1000), anomaly.currentCount)
		assert.Equal(t, float64(100), anomaly.baselineStats.mean)
	})

	t.Run("detect drop", func(t *testing.T) {
		detector := newDetector(config, logger, mockConsumer)

		// Manually set up the counts slice with baseline data
		detector.counts = []int64{100, 100, 100, 100, 100, 10} // Last value is the drop

		// Run anomaly detection directly
		anomaly := detector.checkForAnomaly()

		require.NotNil(t, anomaly, "Expected to detect an anomaly")
		assert.Equal(t, "Drop", anomaly.anomalyType)
		assert.Equal(t, int64(10), anomaly.currentCount)
		assert.Equal(t, float64(100), anomaly.baselineStats.mean)
	})

	t.Run("no anomaly for normal variation", func(t *testing.T) {
		detector := newDetector(config, logger, mockConsumer)

		// Manually set up the counts slice with normal variation
		detector.counts = []int64{100, 95, 105, 98, 102, 103}

		// Run anomaly detection directly
		anomaly := detector.checkForAnomaly()

		assert.Nil(t, anomaly, "Expected no anomaly for normal variation")
	})

	t.Run("handle empty history", func(t *testing.T) {
		detector := newDetector(config, logger, mockConsumer)

		// Empty counts slice
		detector.counts = []int64{}

		// Run anomaly detection directly
		anomaly := detector.checkForAnomaly()

		assert.Nil(t, anomaly, "Expected no anomaly for empty history")
	})

	t.Run("handle single data point", func(t *testing.T) {
		detector := newDetector(config, logger, mockConsumer)

		// Single data point
		detector.counts = []int64{100}

		// Run anomaly detection directly
		anomaly := detector.checkForAnomaly()

		assert.Nil(t, anomaly, "Expected no anomaly for single data point")
	})

	t.Run("validate anomaly statistics", func(t *testing.T) {
		detector := newDetector(config, logger, mockConsumer)

		// Set up known data pattern
		detector.counts = []int64{100, 100, 100, 100, 100, 200} // 100% increase

		// Run anomaly detection directly
		anomaly := detector.checkForAnomaly()

		require.NotNil(t, anomaly, "Expected to detect an anomaly")
		assert.Equal(t, "Spike", anomaly.anomalyType)
		assert.Equal(t, float64(100), anomaly.baselineStats.mean)
		assert.Equal(t, float64(0), anomaly.baselineStats.stdDev) // All baseline values are the same
		assert.Equal(t, float64(100), anomaly.percentageDiff)
	})
}

func TestDetector_Shutdown(t *testing.T) {
	config := &Config{
		AnalysisInterval: time.Minute,
		MaxWindowAge:     time.Hour,
		ZScoreThreshold:  3.0,
		MADThreshold:     3.0,
	}

	logger := zap.NewNop()
	mockConsumer := &mockConsumer{}
	detector := newDetector(config, logger, mockConsumer)

	// Initialize the log channel with proper capacity
	detector.logChan = make(chan logBatch, 1)

	// Start the detector
	err := detector.Start(context.Background(), nil)
	require.NoError(t, err)

	// Allow some time for goroutine to start
	time.Sleep(100 * time.Millisecond)

	// Test clean shutdown
	ctx := context.Background()
	err = detector.Shutdown(ctx)
	assert.NoError(t, err)

	// Verify shutdown behavior
	select {
	case _, ok := <-detector.logChan:
		assert.False(t, ok, "logChan should be closed")
	default:
		// Channel is already closed
	}

	// Test shutdown with canceled context
	detector = newDetector(config, logger, mockConsumer)
	detector.logChan = make(chan logBatch, 1) // Initialize channel for new detector
	err = detector.Start(context.Background(), nil)
	require.NoError(t, err)

	// Allow some time for goroutine to start
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = detector.Shutdown(ctx)
	assert.Error(t, err)
}
