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
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/extension/xextension/storage"
)

func TestGet(t *testing.T) {
	client, err := NewClient(t.TempDir(), &Options{
		Sync: true,
	})
	require.NoError(t, err)

	v, err := client.Get(context.Background(), "test")
	require.NoError(t, err)
	require.Nil(t, v)
}

func TestSet(t *testing.T) {
	client, err := NewClient(t.TempDir(), &Options{
		Sync: true,
	})
	require.NoError(t, err)

	err = client.Set(context.Background(), "test", []byte("test"))
	require.NoError(t, err)

	v, err := client.Get(context.Background(), "test")
	require.NoError(t, err)
	require.Equal(t, []byte("test"), v)
}

func TestDelete(t *testing.T) {
	client, err := NewClient(t.TempDir(), &Options{
		Sync: true,
	})
	require.NoError(t, err)

	err = client.Set(context.Background(), "test", []byte("test"))
	require.NoError(t, err)

	err = client.Delete(context.Background(), "test")
	require.NoError(t, err)

	v, err := client.Get(context.Background(), "test")
	require.NoError(t, err)
	require.Nil(t, v)
}

func TestBatch(t *testing.T) {
	client, err := NewClient(t.TempDir(), &Options{
		Sync: true,
	})
	require.NoError(t, err)

	err = client.Batch(t.Context(), storage.SetOperation("test", []byte("test")))
	require.NoError(t, err)

	v, err := client.Get(t.Context(), "test")
	require.NoError(t, err)
	require.Equal(t, []byte("test"), v)

	getOp := storage.GetOperation("test")
	err = client.Batch(t.Context(), getOp, storage.DeleteOperation("test"))
	require.NoError(t, err)

	require.NoError(t, err)
	require.Equal(t, []byte("test"), getOp.Value)
	v, err = client.Get(t.Context(), "test")
	require.NoError(t, err)
	require.Nil(t, v)
}

func TestClose(t *testing.T) {
	client, err := NewClient(t.TempDir(), &Options{
		Sync: true,
	})
	require.NoError(t, err)

	err = client.Close(t.Context())
	require.NoError(t, err)
}

func TestDoneProcessing(t *testing.T) {
	c, err := NewClient(t.TempDir(), &Options{
		Sync: true,
	})
	require.NoError(t, err)

	internalClient, ok := c.(*client)
	require.True(t, ok)
	require.NoError(t, internalClient.Start(t.Context(), nil))

	err = internalClient.Set(t.Context(), "test", []byte("test"))
	require.NoError(t, err)

	require.NoError(t, internalClient.Close(t.Context()))

	val, err := internalClient.Get(t.Context(), "test")
	require.ErrorContains(t, err, "client is closing")
	require.Nil(t, val)
}
