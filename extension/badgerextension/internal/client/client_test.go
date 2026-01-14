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

package client

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with default options", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.Close(context.Background())
		require.NoError(t, err)
	})

	t.Run("creates client with SyncWrites enabled", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{
			SyncWrites: true,
		}, zap.NewNop())
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.Close(context.Background())
		require.NoError(t, err)
	})

	t.Run("creates client with MemTableSize set", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{
			MemTableSize: 16 << 20, // 16MB
		}, zap.NewNop())
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.Close(context.Background())
		require.NoError(t, err)
	})

	t.Run("creates client with BlockCacheSize set", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{
			BlockCacheSize: 32 << 20, // 32MB
		}, zap.NewNop())
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.Close(context.Background())
		require.NoError(t, err)
	})

	t.Run("creates client with all options set", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{
			SyncWrites:     true,
			MemTableSize:   16 << 20,
			BlockCacheSize: 32 << 20,
		}, zap.NewNop())
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.Close(context.Background())
		require.NoError(t, err)
	})
}

func TestClient_Get(t *testing.T) {
	t.Run("returns nil for non-existent key", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		val, err := c.Get(context.Background(), "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("returns value for existing key", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		// Set a value first
		err = c.Set(context.Background(), "testkey", []byte("testvalue"))
		require.NoError(t, err)

		// Get the value
		val, err := c.Get(context.Background(), "testkey")
		require.NoError(t, err)
		assert.Equal(t, []byte("testvalue"), val)
	})
}

func TestClient_Set(t *testing.T) {
	t.Run("sets value successfully", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		err = c.Set(context.Background(), "key1", []byte("value1"))
		require.NoError(t, err)

		// Verify the value was set
		val, err := c.Get(context.Background(), "key1")
		require.NoError(t, err)
		assert.Equal(t, []byte("value1"), val)
	})

	t.Run("overwrites existing value", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		err = c.Set(context.Background(), "key1", []byte("value1"))
		require.NoError(t, err)

		err = c.Set(context.Background(), "key1", []byte("value2"))
		require.NoError(t, err)

		val, err := c.Get(context.Background(), "key1")
		require.NoError(t, err)
		assert.Equal(t, []byte("value2"), val)
	})

	t.Run("sets empty value", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		err = c.Set(context.Background(), "emptykey", []byte{})
		require.NoError(t, err)

		val, err := c.Get(context.Background(), "emptykey")
		require.NoError(t, err)
		assert.Nil(t, val)
	})
}

func TestClient_Delete(t *testing.T) {
	t.Run("deletes existing key", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		// Set a value
		err = c.Set(context.Background(), "deletekey", []byte("deletevalue"))
		require.NoError(t, err)

		// Verify it exists
		val, err := c.Get(context.Background(), "deletekey")
		require.NoError(t, err)
		assert.Equal(t, []byte("deletevalue"), val)

		// Delete it
		err = c.Delete(context.Background(), "deletekey")
		require.NoError(t, err)

		// Verify it's gone
		val, err = c.Get(context.Background(), "deletekey")
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("deleting non-existent key succeeds", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		// Delete a key that doesn't exist - should not error
		err = c.Delete(context.Background(), "nonexistent")
		require.NoError(t, err)
	})
}

func TestClient_Batch(t *testing.T) {
	t.Run("batch set operations", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		ops := []*storage.Operation{
			{Type: storage.Set, Key: "batch1", Value: []byte("value1")},
			{Type: storage.Set, Key: "batch2", Value: []byte("value2")},
			{Type: storage.Set, Key: "batch3", Value: []byte("value3")},
		}

		err = c.Batch(context.Background(), ops...)
		require.NoError(t, err)

		// Verify all values were set
		for i, op := range ops {
			val, err := c.Get(context.Background(), op.Key)
			require.NoError(t, err)
			assert.Equal(t, []byte("value"+string(rune('1'+i))), val)
		}
	})

	t.Run("batch get operations", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		// Set some values first
		err = c.Set(context.Background(), "getkey1", []byte("getvalue1"))
		require.NoError(t, err)
		err = c.Set(context.Background(), "getkey2", []byte("getvalue2"))
		require.NoError(t, err)

		ops := []*storage.Operation{
			{Type: storage.Get, Key: "getkey1"},
			{Type: storage.Get, Key: "getkey2"},
		}

		err = c.Batch(context.Background(), ops...)
		require.NoError(t, err)

		assert.Equal(t, []byte("getvalue1"), ops[0].Value)
		assert.Equal(t, []byte("getvalue2"), ops[1].Value)
	})

	t.Run("batch delete operations", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		// Set some values first
		err = c.Set(context.Background(), "delkey1", []byte("delvalue1"))
		require.NoError(t, err)
		err = c.Set(context.Background(), "delkey2", []byte("delvalue2"))
		require.NoError(t, err)

		ops := []*storage.Operation{
			{Type: storage.Delete, Key: "delkey1"},
			{Type: storage.Delete, Key: "delkey2"},
		}

		err = c.Batch(context.Background(), ops...)
		require.NoError(t, err)

		// Verify values are deleted
		val, err := c.Get(context.Background(), "delkey1")
		require.NoError(t, err)
		assert.Nil(t, val)

		val, err = c.Get(context.Background(), "delkey2")
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("batch mixed operations", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		// Set initial values
		err = c.Set(context.Background(), "mixkey1", []byte("mixvalue1"))
		require.NoError(t, err)
		err = c.Set(context.Background(), "mixkey2", []byte("mixvalue2"))
		require.NoError(t, err)

		ops := []*storage.Operation{
			{Type: storage.Set, Key: "mixkey3", Value: []byte("mixvalue3")},
			{Type: storage.Get, Key: "mixkey1"},
			{Type: storage.Delete, Key: "mixkey2"},
		}

		err = c.Batch(context.Background(), ops...)
		require.NoError(t, err)

		// Verify the get operation populated the value
		assert.Equal(t, []byte("mixvalue1"), ops[1].Value)

		// Verify mixkey3 was set
		val, err := c.Get(context.Background(), "mixkey3")
		require.NoError(t, err)
		assert.Equal(t, []byte("mixvalue3"), val)

		// Verify mixkey2 was deleted
		val, err = c.Get(context.Background(), "mixkey2")
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("batch with invalid operation type", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		ops := []*storage.Operation{
			{Type: storage.OpType(999), Key: "invalidop"},
		}

		err = c.Batch(context.Background(), ops...)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "wrong operation type")
	})

	t.Run("batch with no operations", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		err = c.Batch(context.Background())
		require.NoError(t, err)
	})

	t.Run("batch get only operations (no write batch)", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		// Set a value first
		err = c.Set(context.Background(), "readonlykey", []byte("readonlyvalue"))
		require.NoError(t, err)

		ops := []*storage.Operation{
			{Type: storage.Get, Key: "readonlykey"},
			{Type: storage.Get, Key: "nonexistent"},
		}

		err = c.Batch(context.Background(), ops...)
		require.NoError(t, err)

		assert.Equal(t, []byte("readonlyvalue"), ops[0].Value)
		assert.Nil(t, ops[1].Value)
	})
}

func TestClient_Close(t *testing.T) {
	t.Run("closes successfully", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)

		err = c.Close(context.Background())
		require.NoError(t, err)
	})
}

func TestClient_RunValueLogGC(t *testing.T) {
	t.Run("runs value log GC", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.db")

		c, err := NewClient(path, &Options{}, zap.NewNop())
		require.NoError(t, err)
		defer func() { _ = c.Close(context.Background()) }()

		// RunValueLogGC typically returns an error when there's nothing to GC
		// This is expected behavior
		err = c.RunValueLogGC(0.5)
		// We just verify it doesn't panic - it may or may not error depending on state
		_ = err
	})
}

func TestTypeString(t *testing.T) {
	tests := []struct {
		name     string
		opType   storage.OpType
		expected string
	}{
		{
			name:     "set operation",
			opType:   storage.Set,
			expected: "set",
		},
		{
			name:     "get operation",
			opType:   storage.Get,
			expected: "get",
		},
		{
			name:     "delete operation",
			opType:   storage.Delete,
			expected: "delete",
		},
		{
			name:     "unknown operation",
			opType:   storage.OpType(999),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeString(tt.opType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBadgerNopLogger(t *testing.T) {
	logger := zap.NewNop().Sugar()
	bnl := &badgerNopLogger{logger: logger}

	t.Run("Warningf does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			bnl.Warningf("warning message: %s", "test")
		})
	})

	t.Run("Debugf does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			bnl.Debugf("debug message: %s", "test")
		})
	})

	t.Run("Infof does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			bnl.Infof("info message: %s", "test")
		})
	})

	t.Run("Errorf does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			bnl.Errorf("error message: %s", "test")
		})
	})
}

func TestClientInterface(t *testing.T) {
	// Verify that client implements the Client interface
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	c, err := NewClient(path, &Options{}, zap.NewNop())
	require.NoError(t, err)
	defer func() { _ = c.Close(context.Background()) }()

	// Assert that the client implements the interface
	var _ = Client(c)
	var _ = storage.Client(c)
}
