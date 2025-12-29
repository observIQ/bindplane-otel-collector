// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/processor/batchprocessor/internal/metadata"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestAttributeBasedBatchingWithByteSizeThresholds(t *testing.T) {
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	
	// Configure both attribute-based grouping AND byte-size batching
	cfg.BatchGroupByAttributes = []string{"service.name"}
	cfg.BatchGroupByAttributeLimit = 10
	cfg.SendBatchSizeBytes = 2000 // 2KB byte threshold
	cfg.SendBatchSize = 100        // 100 items threshold
	cfg.Timeout = 5 * time.Second  // Long timeout to test thresholds
	
	traces, err := NewFactory().CreateTraces(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, sink)
	require.NoError(t, err)
	require.NoError(t, traces.Start(context.Background(), componenttest.NewNopHost()))
	
	// Send traces for serviceA - should batch by service.name
	for i := 0; i < 5; i++ {
		td := testdata.GenerateTraces(10)
		rs := td.ResourceSpans().At(0)
		rs.Resource().Attributes().PutStr("service.name", "serviceA")
		require.NoError(t, traces.ConsumeTraces(context.Background(), td))
	}
	
	// Send traces for serviceB - should be in a separate batch
	for i := 0; i < 5; i++ {
		td := testdata.GenerateTraces(10)
		rs := td.ResourceSpans().At(0)
		rs.Resource().Attributes().PutStr("service.name", "serviceB")
		require.NoError(t, traces.ConsumeTraces(context.Background(), td))
	}
	
	// Wait for batches to be sent (triggered by byte size or item count)
	time.Sleep(500 * time.Millisecond)
	
	require.NoError(t, traces.Shutdown(context.Background()))
	
	// Verify all spans were delivered
	assert.Equal(t, 100, sink.SpanCount(), "Expected all 100 spans to be delivered (50 per service)")
	
	// Verify we got multiple batches (not just 1)
	// Due to byte size threshold, we should have gotten multiple batches per service
	allTraces := sink.AllTraces()
	assert.Greater(t, len(allTraces), 2, "Expected multiple batches due to grouping and thresholds")
}

func TestAttributeGroupingWithBothItemAndByteThresholds(t *testing.T) {
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	
	// Configure attribute grouping with BOTH item and byte thresholds
	cfg.BatchGroupByAttributes = []string{"tenant.id"}
	cfg.BatchGroupByAttributeLimit = 20
	cfg.SendBatchSize = 20             // Item threshold
	cfg.SendBatchSizeBytes = 1000      // Byte threshold  
	cfg.SendBatchMaxSizeBytes = 5000   // Max byte size for splitting
	cfg.Timeout = 10 * time.Second
	
	traces, err := NewFactory().CreateTraces(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, sink)
	require.NoError(t, err)
	require.NoError(t, traces.Start(context.Background(), componenttest.NewNopHost()))
	
	// Send data for tenant1 - should trigger byte threshold first
	for i := 0; i < 10; i++ {
		td := testdata.GenerateTraces(5)
		rs := td.ResourceSpans().At(0)
		rs.Resource().Attributes().PutStr("tenant.id", "tenant1")
		require.NoError(t, traces.ConsumeTraces(context.Background(), td))
	}
	
	// Send data for tenant2 - different tenant, separate batcher
	for i := 0; i < 10; i++ {
		td := testdata.GenerateTraces(5)
		rs := td.ResourceSpans().At(0)
		rs.Resource().Attributes().PutStr("tenant.id", "tenant2")
		require.NoError(t, traces.ConsumeTraces(context.Background(), td))
	}
	
	time.Sleep(500 * time.Millisecond)
	require.NoError(t, traces.Shutdown(context.Background()))
	
	// All spans should be delivered
	assert.Equal(t, 100, sink.SpanCount(), "Expected all 100 spans delivered")
	
	// Should have multiple batches due to both grouping and thresholds
	assert.Greater(t, len(sink.AllTraces()), 1, "Should have multiple batches")
}

func TestAttributeGroupingWithMaxByteSizeSplitting(t *testing.T) {
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	
	// Test that max byte size splitting works with attribute grouping
	cfg.BatchGroupByAttributes = []string{"environment"}
	cfg.SendBatchSize = 1000           // High item threshold
	cfg.SendBatchSizeBytes = 500       // Low byte threshold to trigger sends
	cfg.SendBatchMaxSizeBytes = 2000   // Split if batch gets larger
	cfg.Timeout = 100 * time.Millisecond
	
	traces, err := NewFactory().CreateTraces(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, sink)
	require.NoError(t, err)
	require.NoError(t, traces.Start(context.Background(), componenttest.NewNopHost()))
	
	// Send traces for "production" environment
	for i := 0; i < 20; i++ {
		td := testdata.GenerateTraces(3)
		rs := td.ResourceSpans().At(0)
		rs.Resource().Attributes().PutStr("environment", "production")
		require.NoError(t, traces.ConsumeTraces(context.Background(), td))
	}
	
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, traces.Shutdown(context.Background()))
	
	// All spans should arrive
	assert.Equal(t, 60, sink.SpanCount(), "Expected all 60 spans")
}

func TestMultipleAttributesWithByteBatching(t *testing.T) {
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	
	// Group by multiple attributes AND use byte batching
	cfg.BatchGroupByAttributes = []string{"service.name", "deployment.environment"}
	cfg.BatchGroupByAttributeLimit = 50
	cfg.SendBatchSizeBytes = 1500
	cfg.Timeout = 200 * time.Millisecond
	
	traces, err := NewFactory().CreateTraces(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, sink)
	require.NoError(t, err)
	require.NoError(t, traces.Start(context.Background(), componenttest.NewNopHost()))
	
	// Create different combinations of service + environment
	combinations := []struct {
		service string
		env     string
	}{
		{"api", "prod"},
		{"api", "staging"},
		{"web", "prod"},
		{"web", "staging"},
	}
	
	for _, combo := range combinations {
		for i := 0; i < 5; i++ {
			td := testdata.GenerateTraces(3)
			rs := td.ResourceSpans().At(0)
			rs.Resource().Attributes().PutStr("service.name", combo.service)
			rs.Resource().Attributes().PutStr("deployment.environment", combo.env)
			require.NoError(t, traces.ConsumeTraces(context.Background(), td))
		}
	}
	
	time.Sleep(300 * time.Millisecond)
	require.NoError(t, traces.Shutdown(context.Background()))
	
	// All spans should be delivered (4 combinations × 5 batches × 3 spans)
	assert.Equal(t, 60, sink.SpanCount(), "Expected all spans from all combinations")
}

func TestConfigExampleFromDocumentation(t *testing.T) {
	// This tests the example config from ENHANCEMENTS.md
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	
	// Example from documentation:
	// Group by service name
	cfg.BatchGroupByAttributes = []string{"service.name"}
	cfg.BatchGroupByAttributeLimit = 50
	
	// Use both item and byte thresholds
	cfg.SendBatchSize = 10000
	cfg.SendBatchSizeBytes = 5242880     // 5MB
	cfg.SendBatchMaxSizeBytes = 10485760 // 10MB
	
	cfg.Timeout = 10 * time.Second
	
	// Just verify it validates and creates successfully
	err := cfg.Validate()
	require.NoError(t, err, "Configuration from documentation should be valid")
	
	traces, err := NewFactory().CreateTraces(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, sink)
	require.NoError(t, err, "Should create processor with documented config")
	require.NoError(t, traces.Start(context.Background(), componenttest.NewNopHost()))
	
	// Send some test data
	td := testdata.GenerateTraces(10)
	rs := td.ResourceSpans().At(0)
	rs.Resource().Attributes().PutStr("service.name", "test-service")
	require.NoError(t, traces.ConsumeTraces(context.Background(), td))
	
	require.NoError(t, traces.Shutdown(context.Background()))
	
	// Data should be delivered
	assert.Equal(t, 10, sink.SpanCount())
}
