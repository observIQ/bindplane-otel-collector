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

package badgerextension

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/observiq/bindplane-otel-collector/extension/badgerextension/internal/client/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewBadgerExtension(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		Directory: &DirectoryConfig{
			Path: t.TempDir(),
		},
	}

	ext := newBadgerExtension(logger, cfg)
	require.NotNil(t, ext)

	badgerExt, ok := ext.(*badgerExtension)
	require.True(t, ok)
	assert.Equal(t, logger, badgerExt.logger)
	assert.Equal(t, cfg, badgerExt.cfg)
	assert.NotNil(t, badgerExt.clients)
	assert.Empty(t, badgerExt.clients)
}

func TestBadgerExtension_GetClient(t *testing.T) {
	t.Run("creates client with empty name", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		client, err := ext.GetClient(
			context.Background(),
			component.KindReceiver,
			component.MustNewID("test"),
			"",
		)
		require.NoError(t, err)
		require.NotNil(t, client)

		// Verify the client was stored
		assert.Len(t, ext.clients, 1)

		// Cleanup
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})

	t.Run("creates client with name", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		client, err := ext.GetClient(
			context.Background(),
			component.KindReceiver,
			component.MustNewID("test"),
			"myqueue",
		)
		require.NoError(t, err)
		require.NotNil(t, client)

		// Verify the client was stored
		assert.Len(t, ext.clients, 1)

		// Cleanup
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})

	t.Run("returns existing client", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		// Get client first time
		client1, err := ext.GetClient(
			context.Background(),
			component.KindReceiver,
			component.MustNewID("test"),
			"",
		)
		require.NoError(t, err)
		require.NotNil(t, client1)

		// Get same client second time
		client2, err := ext.GetClient(
			context.Background(),
			component.KindReceiver,
			component.MustNewID("test"),
			"",
		)
		require.NoError(t, err)
		require.NotNil(t, client2)

		// Should be the same client
		assert.Equal(t, client1, client2)
		assert.Len(t, ext.clients, 1)

		// Cleanup
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})

	t.Run("creates different clients for different components", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		client1, err := ext.GetClient(
			context.Background(),
			component.KindReceiver,
			component.MustNewID("test1"),
			"",
		)
		require.NoError(t, err)
		require.NotNil(t, client1)

		client2, err := ext.GetClient(
			context.Background(),
			component.KindExporter,
			component.MustNewID("test2"),
			"",
		)
		require.NoError(t, err)
		require.NotNil(t, client2)

		assert.NotEqual(t, client1, client2)
		assert.Len(t, ext.clients, 2)

		// Cleanup
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})

	t.Run("creates client with named component ID", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		client, err := ext.GetClient(
			context.Background(),
			component.KindProcessor,
			component.MustNewIDWithName("batch", "myinstance"),
			"queue",
		)
		require.NoError(t, err)
		require.NotNil(t, client)

		// Verify the client was stored with the correct key
		assert.Len(t, ext.clients, 1)

		// Cleanup
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})
}

func TestBadgerExtension_Start(t *testing.T) {
	t.Run("starts without garbage collection", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
			BlobGarbageCollection: nil,
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		err := ext.Start(context.Background(), nil)
		require.NoError(t, err)

		// No GC context should be set
		assert.Nil(t, ext.gcContextCancel)

		// Cleanup
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})

	t.Run("starts with garbage collection", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
			BlobGarbageCollection: &BlobGarbageCollectionConfig{
				Interval:     100 * time.Millisecond,
				DiscardRatio: 0.5,
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		err := ext.Start(context.Background(), nil)
		require.NoError(t, err)

		// GC context should be set
		assert.NotNil(t, ext.gcContextCancel)

		// Cleanup
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})
}

func TestBadgerExtension_Shutdown(t *testing.T) {
	t.Run("shutdown without clients", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		err := ext.Shutdown(context.Background())
		require.NoError(t, err)
	})

	t.Run("shutdown with clients", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		// Create some clients
		_, err := ext.GetClient(
			context.Background(),
			component.KindReceiver,
			component.MustNewID("test1"),
			"",
		)
		require.NoError(t, err)

		_, err = ext.GetClient(
			context.Background(),
			component.KindExporter,
			component.MustNewID("test2"),
			"",
		)
		require.NoError(t, err)

		assert.Len(t, ext.clients, 2)

		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})

	t.Run("shutdown cancels garbage collection", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
			BlobGarbageCollection: &BlobGarbageCollectionConfig{
				Interval:     100 * time.Millisecond,
				DiscardRatio: 0.5,
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		err := ext.Start(context.Background(), nil)
		require.NoError(t, err)
		assert.NotNil(t, ext.gcContextCancel)

		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})

	t.Run("shutdown returns error when client close fails", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		// Inject a mock client that returns an error on Close
		mockClient := mocks.NewClient(t)
		mockClient.EXPECT().Close(mock.Anything).Return(errors.New("close error"))

		ext.clients["test_client"] = mockClient

		err := ext.Shutdown(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to close badger client")
	})
}

func TestBadgerExtension_GarbageCollection(t *testing.T) {
	t.Run("runs garbage collection on clients", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
			BlobGarbageCollection: &BlobGarbageCollectionConfig{
				Interval:     50 * time.Millisecond,
				DiscardRatio: 0.5,
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		// Create a client first
		client, err := ext.GetClient(
			context.Background(),
			component.KindReceiver,
			component.MustNewID("test"),
			"",
		)
		require.NoError(t, err)
		require.NotNil(t, client)

		// Write some data
		err = client.Set(context.Background(), "key1", []byte("value1"))
		require.NoError(t, err)

		// Start the extension (which starts GC)
		err = ext.Start(context.Background(), nil)
		require.NoError(t, err)

		// Wait for at least one GC cycle
		time.Sleep(100 * time.Millisecond)

		// Shutdown
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})

	t.Run("logs error when garbage collection fails", func(t *testing.T) {
		// Create an observable logger to capture log messages
		core, logs := observer.New(zap.WarnLevel)
		logger := zap.New(core)

		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
			BlobGarbageCollection: &BlobGarbageCollectionConfig{
				Interval:     50 * time.Millisecond,
				DiscardRatio: 0.5,
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		// Inject a mock client that returns an error on RunValueLogGC
		mockClient := mocks.NewClient(t)
		mockClient.EXPECT().RunValueLogGC(0.5).Return(errors.New("gc error"))
		mockClient.EXPECT().Close(mock.Anything).Return(nil)

		ext.clients["test_client"] = mockClient

		// Start the extension (which starts GC)
		err := ext.Start(context.Background(), nil)
		require.NoError(t, err)

		// Wait for at least one GC cycle to execute
		time.Sleep(100 * time.Millisecond)

		// Shutdown
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)

		// Check that an error was logged
		// Note: Due to the goroutine-based error handling in runGC, the error may not be consistently logged
		// but we verify the mock was called
		mockClient.AssertCalled(t, "RunValueLogGC", 0.5)

		// Check logs if any error was recorded
		require.GreaterOrEqual(t, len(logs.All()), 1)
	})
}

func TestBadgerExtension_ClientOptions(t *testing.T) {
	t.Run("returns options with sync writes disabled", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
			SyncWrites: false,
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)
		opts := ext.clientOptions()

		assert.False(t, opts.SyncWrites)
	})

	t.Run("returns options with sync writes enabled", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
			SyncWrites: true,
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)
		opts := ext.clientOptions()

		assert.True(t, opts.SyncWrites)
	})
}

func TestKindString(t *testing.T) {
	tests := []struct {
		name     string
		kind     component.Kind
		expected string
	}{
		{
			name:     "receiver",
			kind:     component.KindReceiver,
			expected: "receiver",
		},
		{
			name:     "processor",
			kind:     component.KindProcessor,
			expected: "processor",
		},
		{
			name:     "exporter",
			kind:     component.KindExporter,
			expected: "exporter",
		},
		{
			name:     "extension",
			kind:     component.KindExtension,
			expected: "extension",
		},
		{
			name:     "connector",
			kind:     component.KindConnector,
			expected: "connector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := kindString(tt.kind)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBadgerExtension_FullLifecycle(t *testing.T) {
	t.Run("full lifecycle with storage operations", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
			SyncWrites: true,
			BlobGarbageCollection: &BlobGarbageCollectionConfig{
				Interval:     100 * time.Millisecond,
				DiscardRatio: 0.5,
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		// Start
		err := ext.Start(context.Background(), nil)
		require.NoError(t, err)

		// Get a client
		client, err := ext.GetClient(
			context.Background(),
			component.KindReceiver,
			component.MustNewID("otlp"),
			"persistent_queue",
		)
		require.NoError(t, err)

		// Perform storage operations
		ctx := context.Background()

		// Set
		err = client.Set(ctx, "test_key", []byte("test_value"))
		require.NoError(t, err)

		// Get
		val, err := client.Get(ctx, "test_key")
		require.NoError(t, err)
		assert.Equal(t, []byte("test_value"), val)

		// Delete
		err = client.Delete(ctx, "test_key")
		require.NoError(t, err)

		// Verify deleted
		val, err = client.Get(ctx, "test_key")
		require.NoError(t, err)
		assert.Nil(t, val)

		// Shutdown
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})

	t.Run("multiple clients lifecycle", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext := newBadgerExtension(logger, cfg).(*badgerExtension)

		// Start
		err := ext.Start(context.Background(), nil)
		require.NoError(t, err)

		// Get multiple clients
		client1, err := ext.GetClient(
			context.Background(),
			component.KindReceiver,
			component.MustNewID("otlp"),
			"queue1",
		)
		require.NoError(t, err)

		client2, err := ext.GetClient(
			context.Background(),
			component.KindExporter,
			component.MustNewID("googlecloud"),
			"queue2",
		)
		require.NoError(t, err)

		// Operations on client1
		err = client1.Set(context.Background(), "key1", []byte("value1"))
		require.NoError(t, err)

		// Operations on client2
		err = client2.Set(context.Background(), "key2", []byte("value2"))
		require.NoError(t, err)

		// Verify data isolation
		val, err := client1.Get(context.Background(), "key1")
		require.NoError(t, err)
		assert.Equal(t, []byte("value1"), val)

		val, err = client1.Get(context.Background(), "key2")
		require.NoError(t, err)
		assert.Nil(t, val) // Should not find key2 in client1's storage

		val, err = client2.Get(context.Background(), "key2")
		require.NoError(t, err)
		assert.Equal(t, []byte("value2"), val)

		// Shutdown
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})
}
