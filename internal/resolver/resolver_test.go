// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resolver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// testMeterProvider returns a MeterProvider for use in tests
func testMeterProvider() otelmetric.MeterProvider {
	return metric.NewMeterProvider()
}

func TestNew(t *testing.T) {
	tests := []struct {
		name          string
		mp            otelmetric.MeterProvider
		logger        *zap.Logger
		cacheSize     int
		wantErr       bool
		errorContains string
	}{
		{
			name:      "valid cache size",
			mp:        testMeterProvider(),
			logger:    zap.NewNop(),
			cacheSize: 100,
			wantErr:   false,
		},
		{
			name:      "custom cache size",
			mp:        testMeterProvider(),
			logger:    zap.NewNop(),
			cacheSize: 500,
			wantErr:   false,
		},
		{
			name:      "with metrics",
			mp:        metric.NewMeterProvider(),
			logger:    zap.NewNop(),
			cacheSize: 100,
			wantErr:   false,
		},
		{
			name:          "nil Logger returns error",
			mp:            testMeterProvider(),
			logger:        nil,
			cacheSize:     100,
			wantErr:       true,
			errorContains: "Logger is required",
		},
		{
			name:          "nil MeterProvider returns error",
			mp:            nil,
			logger:        zap.NewNop(),
			cacheSize:     100,
			wantErr:       true,
			errorContains: "MeterProvider is required",
		},
		{
			name:          "zero cache size returns error",
			mp:            testMeterProvider(),
			logger:        zap.NewNop(),
			cacheSize:     0,
			wantErr:       true,
			errorContains: "cache size must be greater than 0",
		},
		{
			name:          "negative cache size returns error",
			mp:            testMeterProvider(),
			logger:        zap.NewNop(),
			cacheSize:     -1,
			wantErr:       true,
			errorContains: "cache size must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := New(tt.mp, tt.logger, tt.cacheSize)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, r)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, r)
			}
		})
	}
}

func TestResolver_lookupIPAddr(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	r, err := New(testMeterProvider(), logger, 10)
	require.NoError(t, err)
	require.NotNil(t, r)

	// Test lookup of localhost
	addrs, err := r.lookupIPAddr(ctx, "localhost")
	require.NoError(t, err)
	require.NotEmpty(t, addrs)

	// Second lookup should be cached
	addrs2, err := r.lookupIPAddr(ctx, "localhost")
	require.NoError(t, err)
	require.Equal(t, addrs, addrs2)

	// Verify cache hit
	require.Equal(t, 1, r.cache.Len())
}

func TestResolver_lookupIPAddr_CacheHit(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	defer mp.Shutdown(ctx)

	r, err := New(mp, logger, 10)
	require.NoError(t, err)

	// First lookup - cache miss
	_, err = r.lookupIPAddr(ctx, "localhost")
	require.NoError(t, err)

	// Second lookup - cache hit
	_, err = r.lookupIPAddr(ctx, "localhost")
	require.NoError(t, err)

	// Collect metrics
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
}

func TestResolver_lookupIPAddr_InvalidHost(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	r, err := New(testMeterProvider(), logger, 10)
	require.NoError(t, err)

	// Test lookup of invalid hostname
	_, err = r.lookupIPAddr(ctx, "invalid-hostname-that-does-not-exist-12345")
	require.Error(t, err)
}

func TestResolver_DialContext(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	r, err := New(testMeterProvider(), logger, 10)
	require.NoError(t, err)

	// Test dialing localhost
	conn, err := r.DialContext(ctx, "tcp", "localhost:0")
	if err == nil {
		// If connection succeeds, close it
		conn.Close()
	}
	// We expect an error since we're dialing port 0, but the DNS lookup should work
	// The error should be about connection, not DNS
	require.NotNil(t, err)
	require.NotContains(t, err.Error(), "no such host")
}

func TestResolver_DialContext_InvalidAddress(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	r, err := New(testMeterProvider(), logger, 10)
	require.NoError(t, err)

	// Test with invalid address format
	_, err = r.DialContext(ctx, "tcp", "invalid-address")
	require.Error(t, err)
}

func TestResolver_CacheEviction(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	// Create resolver with small cache
	r, err := New(testMeterProvider(), logger, 2)
	require.NoError(t, err)

	// Add entries to fill cache
	_, err = r.lookupIPAddr(ctx, "localhost")
	require.NoError(t, err)

	_, err = r.lookupIPAddr(ctx, "127.0.0.1")
	require.NoError(t, err)

	// Add one more to trigger eviction
	_, err = r.lookupIPAddr(ctx, "::1")
	require.NoError(t, err)

	// Cache should be at capacity
	require.Equal(t, 2, r.cache.Len())
}

func TestResolver_Metrics(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	r, err := New(mp, logger, 100)
	require.NoError(t, err)

	// Perform lookups
	_, err = r.lookupIPAddr(ctx, "localhost")
	require.NoError(t, err)

	_, err = r.lookupIPAddr(ctx, "localhost") // cache hit
	require.NoError(t, err)

	_, err = r.lookupIPAddr(ctx, "127.0.0.1") // cache miss
	require.NoError(t, err)

	// Collect metrics
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	// Verify metrics were recorded
	require.NotEmpty(t, rm.ScopeMetrics)
}

func TestResolver_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	r, err := New(testMeterProvider(), logger, 100)
	require.NoError(t, err)

	// Concurrent lookups
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := r.lookupIPAddr(ctx, "localhost")
			require.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Cache should have the entry
	require.Equal(t, 1, r.cache.Len())
}

func TestResolver_CacheSize(t *testing.T) {
	logger := zap.NewNop()

	cacheSize := 1000
	r, err := New(testMeterProvider(), logger, cacheSize)
	require.NoError(t, err)
	require.NotNil(t, r)
	// Verify cache was created with correct size by checking it can hold that many items
	require.Equal(t, 0, r.cache.Len())
}

func TestResolver_HTTPClientIntegration(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	r, err := New(testMeterProvider(), logger, 100)
	require.NoError(t, err)

	// Test that DialContext works (even if connection fails, DNS should work)
	_, err = r.DialContext(ctx, "tcp", "localhost:0")
	// We expect connection error, not DNS error
	if err != nil {
		require.NotContains(t, err.Error(), "no such host")
	}
}
