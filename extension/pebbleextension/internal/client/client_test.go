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
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"
)

func TestGet(t *testing.T) {
	client, err := NewClient(t.TempDir(), zap.NewNop(), &Options{
		Sync: true,
	})
	require.NoError(t, err)

	v, err := client.Get(t.Context(), "test")
	require.NoError(t, err)
	require.Nil(t, v)

	require.NoError(t, client.Close(t.Context()))
}

func TestSet(t *testing.T) {
	client, err := NewClient(t.TempDir(), zap.NewNop(), &Options{
		Sync: true,
	})
	require.NoError(t, err)

	err = client.Set(t.Context(), "test", []byte("test"))
	require.NoError(t, err)

	v, err := client.Get(t.Context(), "test")
	require.NoError(t, err)
	require.Equal(t, []byte("test"), v)

	require.NoError(t, client.Close(t.Context()))
}

func TestDelete(t *testing.T) {
	client, err := NewClient(t.TempDir(), zap.NewNop(), &Options{
		Sync: true,
	})
	require.NoError(t, err)

	err = client.Set(t.Context(), "test", []byte("test"))
	require.NoError(t, err)

	err = client.Delete(t.Context(), "test")
	require.NoError(t, err)

	v, err := client.Get(t.Context(), "test")
	require.NoError(t, err)
	require.Nil(t, v)

	require.NoError(t, client.Close(t.Context()))
}

func TestBatch(t *testing.T) {
	client, err := NewClient(t.TempDir(), zap.NewNop(), &Options{
		Sync: true,
	})
	require.NoError(t, err)

	err = client.Batch(t.Context(), storage.SetOperation("test0", []byte("test0")), storage.SetOperation("test1", []byte("test1")))
	require.NoError(t, err)

	// validating that the Set was performed
	v, err := client.Get(t.Context(), "test0")
	require.NoError(t, err)
	require.Equal(t, []byte("test0"), v)

	// batch deleting the items and getting a non-existent key
	err = client.Batch(t.Context(), storage.DeleteOperation("test0"), storage.DeleteOperation("test1"), storage.GetOperation("test2"))
	require.NoError(t, err)

	v, err = client.Get(t.Context(), "test0")
	require.NoError(t, err)
	require.Nil(t, v)

	v, err = client.Get(t.Context(), "test1")
	require.NoError(t, err)
	require.Nil(t, v)

	require.NoError(t, client.Close(t.Context()))
}

func TestClose(t *testing.T) {
	client, err := NewClient(t.TempDir(), zap.NewNop(), &Options{
		Sync: true,
	})
	require.NoError(t, err)

	err = client.Close(t.Context())
	require.NoError(t, err)
}

func TestCompaction(t *testing.T) {
	testCases := []struct {
		name string
		sync bool
	}{
		{name: "sync", sync: true},
		{name: "async", sync: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(t.TempDir(), zap.NewNop(), &Options{
				Sync: tt.sync,
			})
			require.NoError(t, err)
			err = client.Compact(true)
			require.NoError(t, err)
			require.NoError(t, client.Close(t.Context()))
		})
	}
}
