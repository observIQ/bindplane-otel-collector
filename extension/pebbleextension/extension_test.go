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

package pebbleextension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

func TestNewPebbleExtension(t *testing.T) {
	logger := zap.NewNop()
	cfg := &Config{
		Directory: &DirectoryConfig{
			Path: t.TempDir(),
		},
	}

	ext, err := newPebbleExtension(logger, cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)
	assert.Equal(t, logger, ext.logger)
	assert.Equal(t, cfg, ext.cfg)
	assert.NotNil(t, ext.clients)
	assert.Empty(t, ext.clients)
}

func TestPebbleExtension_GetClient(t *testing.T) {
	t.Run("creates client with empty name", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext, err := newPebbleExtension(logger, cfg)
		require.NoError(t, err)
		defer func() { _ = ext.Shutdown(context.Background()) }()

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
	})

	t.Run("creates client with name", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext, err := newPebbleExtension(logger, cfg)
		require.NoError(t, err)
		require.NoError(t, ext.Start(t.Context(), nil))
		defer func() { require.NoError(t, ext.Shutdown(t.Context())) }()

		client, err := ext.GetClient(
			context.Background(),
			component.KindReceiver,
			component.MustNewID("test"),
			"myqueue",
		)
		require.NoError(t, err)
		require.NotNil(t, client)

		assert.Len(t, ext.clients, 1)
	})

	t.Run("returns existing client", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext, err := newPebbleExtension(logger, cfg)
		require.NoError(t, err)

		ext.Start(t.Context(), nil)
		defer func() { _ = ext.Shutdown(t.Context()) }()

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
	})

	t.Run("creates different clients for different components", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext, err := newPebbleExtension(logger, cfg)
		require.NoError(t, err)
		ext.Start(t.Context(), nil)
		defer func() { _ = ext.Shutdown(t.Context()) }()

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
	})

	t.Run("creates client with named component ID", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext, err := newPebbleExtension(logger, cfg)
		require.NoError(t, err)
		ext.Start(t.Context(), nil)
		defer func() { _ = ext.Shutdown(t.Context()) }()

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
	})
}

func TestPebbleExtension_Start(t *testing.T) {
	t.Run("starts successfully", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext, err := newPebbleExtension(logger, cfg)
		require.NoError(t, err)

		err = ext.Start(context.Background(), nil)
		require.NoError(t, err)

		// Cleanup
		err = ext.Shutdown(context.Background())
		require.NoError(t, err)
	})
}

func TestPebbleExtension_Shutdown(t *testing.T) {
	t.Run("shutdown without clients", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext, err := newPebbleExtension(logger, cfg)
		require.NoError(t, err)

		err = ext.Shutdown(t.Context())
		require.NoError(t, err)
	})

	t.Run("shutdown with clients", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext, err := newPebbleExtension(logger, cfg)
		require.NoError(t, err)

		require.NoError(t, ext.Start(t.Context(), nil))
		defer func() { require.NoError(t, ext.Shutdown(t.Context())) }()

		// Create some clients
		_, err = ext.GetClient(
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

func TestPebbleExtension_FullLifecycle(t *testing.T) {
	t.Run("full lifecycle with storage operations", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
			Sync: true,
		}

		ext, err := newPebbleExtension(logger, cfg)
		require.NoError(t, err)

		// Start
		err = ext.Start(context.Background(), nil)
		require.NoError(t, err)
		defer func() { require.NoError(t, ext.Shutdown(context.Background())) }()

		// Get a client
		client, err := ext.GetClient(
			context.Background(),
			component.KindReceiver,
			component.MustNewID("otlp"),
			"persistent_queue",
		)
		require.NoError(t, err)
		ctx := t.Context()
		err = client.Set(ctx, "test_key", []byte("test_value"))
		require.NoError(t, err)

		val, err := client.Get(ctx, "test_key")
		require.NoError(t, err)
		assert.Equal(t, []byte("test_value"), val)

		err = client.Delete(ctx, "test_key")
		require.NoError(t, err)

		val, err = client.Get(ctx, "test_key")
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("multiple clients lifecycle", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Directory: &DirectoryConfig{
				Path: t.TempDir(),
			},
		}

		ext, err := newPebbleExtension(logger, cfg)
		require.NoError(t, err)

		err = ext.Start(context.Background(), nil)
		require.NoError(t, err)
		defer func() { require.NoError(t, ext.Shutdown(t.Context())) }()

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

		err = client1.Set(context.Background(), "key1", []byte("value1"))
		require.NoError(t, err)

		err = client2.Set(context.Background(), "key2", []byte("value2"))
		require.NoError(t, err)

		val, err := client1.Get(context.Background(), "key1")
		require.NoError(t, err)
		assert.Equal(t, []byte("value1"), val)

		val, err = client1.Get(context.Background(), "key2")
		require.NoError(t, err)
		assert.Nil(t, val) // Should not find key2 in client1's storage

		val, err = client2.Get(context.Background(), "key2")
		require.NoError(t, err)
		assert.Equal(t, []byte("value2"), val)
	})
}
