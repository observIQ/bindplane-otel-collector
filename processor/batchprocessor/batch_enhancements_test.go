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
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/processor/batchprocessor/internal/metadata"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestBatchProcessorByteBasedBatching(t *testing.T) {
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	cfg.SendBatchSizeBytes = 1000 // 1KB threshold
	cfg.Timeout = 100 * time.Millisecond
	
	traces, err := NewFactory().CreateTraces(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, sink)
	require.NoError(t, err)
	require.NoError(t, traces.Start(context.Background(), componenttest.NewNopHost()))
	
	// Send some traces
	for i := 0; i < 10; i++ {
		td := testdata.GenerateTraces(5)
		require.NoError(t, traces.ConsumeTraces(context.Background(), td))
	}
	
	// Wait for batches to be sent
	time.Sleep(200 * time.Millisecond)
	
	require.NoError(t, traces.Shutdown(context.Background()))
	
	// Verify we received the data
	assert.Greater(t, sink.SpanCount(), 0, "Expected spans to be delivered")
	assert.Equal(t, 50, sink.SpanCount(), "Expected all 50 spans to be delivered")
}

func TestBatchProcessorAttributeBasedGrouping(t *testing.T) {
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	cfg.SendBatchSize = 10
	cfg.Timeout = 100 * time.Millisecond
	cfg.BatchGroupByAttributes = []string{"service.name"}
	cfg.BatchGroupByAttributeLimit = 10
	
	traces, err := NewFactory().CreateTraces(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, sink)
	require.NoError(t, err)
	require.NoError(t, traces.Start(context.Background(), componenttest.NewNopHost()))
	
	// Send traces with different service names
	for i := 0; i < 3; i++ {
		td := testdata.GenerateTraces(5)
		// Add service.name attribute to resource
		rs := td.ResourceSpans().At(0)
		rs.Resource().Attributes().PutStr("service.name", "service"+string(rune('A'+i)))
		require.NoError(t, traces.ConsumeTraces(context.Background(), td))
	}
	
	// Wait for batches to be sent
	time.Sleep(200 * time.Millisecond)
	
	require.NoError(t, traces.Shutdown(context.Background()))
	
	// Verify all spans were delivered
	assert.Equal(t, 15, sink.SpanCount(), "Expected all 15 spans to be delivered")
}

func TestBatchProcessorAttributeGroupingValidation(t *testing.T) {
	// Test that duplicate attributes are rejected
	cfg := &Config{
		BatchGroupByAttributes: []string{"service.name", "service.name"},
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate entry")
	
	// Test that metadata_keys and batch_group_by_attributes cannot both be set
	cfg = &Config{
		MetadataKeys:           []string{"tenant_id"},
		BatchGroupByAttributes: []string{"service.name"},
	}
	err = cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot both be set")
}

func TestBatchProcessorByteBasedConfigValidation(t *testing.T) {
	// Test that max must be >= min
	cfg := &Config{
		SendBatchSizeBytes:    1000,
		SendBatchMaxSizeBytes: 500,
	}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send_batch_max_size_bytes must be greater or equal")
}

func TestAttributeExtraction(t *testing.T) {
	t.Run("extract from traces", func(t *testing.T) {
		td := ptrace.NewTraces()
		rs := td.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr("resource.attr", "resource-value")
		ss := rs.ScopeSpans().AppendEmpty()
		ss.Scope().Attributes().PutStr("scope.attr", "scope-value")
		span := ss.Spans().AppendEmpty()
		span.Attributes().PutStr("span.attr", "span-value")
		
		// Extract span attribute (highest priority)
		attrSet := extractAttributesFromTraces(td, []string{"span.attr"})
		attrs := attrSet.ToSlice()
		require.Len(t, attrs, 1)
		assert.Equal(t, "span.attr", string(attrs[0].Key))
		assert.Equal(t, "span-value", attrs[0].Value.AsString())
		
		// Extract resource attribute when span attribute doesn't exist
		attrSet = extractAttributesFromTraces(td, []string{"resource.attr"})
		attrs = attrSet.ToSlice()
		require.Len(t, attrs, 1)
		assert.Equal(t, "resource.attr", string(attrs[0].Key))
		assert.Equal(t, "resource-value", attrs[0].Value.AsString())
		
		// Extract missing attribute (should return empty)
		attrSet = extractAttributesFromTraces(td, []string{"missing.attr"})
		attrs = attrSet.ToSlice()
		require.Len(t, attrs, 1)
		assert.Equal(t, "missing.attr", string(attrs[0].Key))
		assert.Equal(t, "", attrs[0].Value.AsString())
	})
}

func TestByteThresholdsAndItemThresholdsTogether(t *testing.T) {
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	cfg.SendBatchSize = 100        // 100 items
	cfg.SendBatchSizeBytes = 500   // 500 bytes - whichever comes first
	cfg.Timeout = 5 * time.Second  // Long timeout so we test the thresholds
	
	traces, err := NewFactory().CreateTraces(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, sink)
	require.NoError(t, err)
	require.NoError(t, traces.Start(context.Background(), componenttest.NewNopHost()))
	
	// Send a single large batch that should trigger byte threshold
	td := testdata.GenerateTraces(10)
	require.NoError(t, traces.ConsumeTraces(context.Background(), td))
	
	// Give it time to process
	time.Sleep(100 * time.Millisecond)
	
	require.NoError(t, traces.Shutdown(context.Background()))
	
	// We should have received the data (either threshold would work)
	assert.Equal(t, 10, sink.SpanCount())
}

func TestPcommonValueConversion(t *testing.T) {
	tests := []struct {
		name     string
		setValue func(pcommon.Value)
		expected string
	}{
		{
			name:     "string",
			setValue: func(v pcommon.Value) { v.SetStr("test") },
			expected: "test",
		},
		{
			name:     "int",
			setValue: func(v pcommon.Value) { v.SetInt(42) },
			expected: "42",
		},
		{
			name:     "bool",
			setValue: func(v pcommon.Value) { v.SetBool(true) },
			expected: "true",
		},
		{
			name:     "double",
			setValue: func(v pcommon.Value) { v.SetDouble(3.14) },
			expected: "3.14",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := pcommon.NewValueEmpty()
			tt.setValue(val)
			
			attr := pcommonValueToOtelAttribute("test.key", val)
			assert.Equal(t, "test.key", string(attr.Key))
		})
	}
}
