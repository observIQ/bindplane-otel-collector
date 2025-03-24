// Copyright observIQ, Inc.
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

package googlecloudstoragerehydrationreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/googlecloudstoragerehydrationreceiver"

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"
)

type mockStorageClient struct {
	listBlobsFunc    func(ctx context.Context, startTime, endTime time.Time) ([]*BlobInfo, error)
	downloadBlobFunc func(ctx context.Context, name string) ([]byte, error)
	deleteBlobFunc   func(ctx context.Context, name string) error
}

func (m *mockStorageClient) ListBlobs(ctx context.Context, startTime, endTime time.Time) ([]*BlobInfo, error) {
	if m.listBlobsFunc != nil {
		return m.listBlobsFunc(ctx, startTime, endTime)
	}
	return nil, nil
}

func (m *mockStorageClient) DownloadBlob(ctx context.Context, name string) ([]byte, error) {
	if m.downloadBlobFunc != nil {
		return m.downloadBlobFunc(ctx, name)
	}
	return nil, nil
}

func (m *mockStorageClient) DeleteBlob(ctx context.Context, name string) error {
	if m.deleteBlobFunc != nil {
		return m.deleteBlobFunc(ctx, name)
	}
	return nil
}

type mockConsumer struct {
	consumeFunc func(ctx context.Context, data []byte) error
}

func (m *mockConsumer) Consume(ctx context.Context, data []byte) error {
	if m.consumeFunc != nil {
		return m.consumeFunc(ctx, data)
	}
	return nil
}

type mockStorageExtension struct {
	storage.Extension
	getClientFunc func(ctx context.Context, kind component.Kind, id component.ID, signal string) (storage.Client, error)
}

func (m *mockStorageExtension) GetClient(ctx context.Context, kind component.Kind, id component.ID, signal string) (storage.Client, error) {
	if m.getClientFunc != nil {
		return m.getClientFunc(ctx, kind, id, signal)
	}
	return nil, nil
}

func TestNewMetricsReceiver(t *testing.T) {
	cfg := &Config{
		BucketName:    "test-bucket",
		StartingTime:  "2024-01-01T00:00:00Z",
		EndingTime:    "2024-01-02T00:00:00Z",
		BatchSize:     30,
		PageSize:      1000,
		DeleteOnRead:  false,
	}

	receiver, err := newMetricsReceiver(component.NewIDWithName("test", "test"), zap.NewNop(), cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, receiver)
}

func TestNewLogsReceiver(t *testing.T) {
	cfg := &Config{
		BucketName:    "test-bucket",
		StartingTime:  "2024-01-01T00:00:00Z",
		EndingTime:    "2024-01-02T00:00:00Z",
		BatchSize:     30,
		PageSize:      1000,
		DeleteOnRead:  false,
	}

	receiver, err := newLogsReceiver(component.NewIDWithName("test", "test"), zap.NewNop(), cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, receiver)
}

func TestNewTracesReceiver(t *testing.T) {
	cfg := &Config{
		BucketName:    "test-bucket",
		StartingTime:  "2024-01-01T00:00:00Z",
		EndingTime:    "2024-01-02T00:00:00Z",
		BatchSize:     30,
		PageSize:      1000,
		DeleteOnRead:  false,
	}

	receiver, err := newTracesReceiver(component.NewIDWithName("test", "test"), zap.NewNop(), cfg, consumertest.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, receiver)
}

func TestRehydrationReceiver_Start(t *testing.T) {
	cfg := &Config{
		BucketName:    "test-bucket",
		StartingTime:  "2024-01-01T00:00:00Z",
		EndingTime:    "2024-01-02T00:00:00Z",
		BatchSize:     30,
		PageSize:      1000,
		DeleteOnRead:  false,
		StorageID:     component.NewIDWithName("test", "storage"),
	}

	storageID := component.NewIDWithName("test", "storage")
	storageExt := &mockStorageExtension{
		getClientFunc: func(ctx context.Context, kind component.Kind, id component.ID, signal string) (storage.Client, error) {
			return &mockStorageClient{}, nil
		},
	}

	host := &mockHost{
		extensions: map[component.ID]component.Component{
			storageID: storageExt,
		},
	}

	receiver, err := newMetricsReceiver(storageID, zap.NewNop(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, receiver)

	err = receiver.Start(context.Background(), host)
	assert.NoError(t, err)
}

func TestRehydrationReceiver_Shutdown(t *testing.T) {
	cfg := &Config{
		BucketName:    "test-bucket",
		StartingTime:  "2024-01-01T00:00:00Z",
		EndingTime:    "2024-01-02T00:00:00Z",
		BatchSize:     30,
		PageSize:      1000,
		DeleteOnRead:  false,
	}

	receiver, err := newMetricsReceiver(component.NewIDWithName("test", "test"), zap.NewNop(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, receiver)

	err = receiver.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestRehydrationReceiver_ProcessBlob(t *testing.T) {
	cfg := &Config{
		BucketName:    "test-bucket",
		StartingTime:  "2024-01-01T00:00:00Z",
		EndingTime:    "2024-01-02T00:00:00Z",
		BatchSize:     30,
		PageSize:      1000,
		DeleteOnRead:  true,
	}

	blob := &BlobInfo{
		Name:         "test-blob",
		LastModified: time.Now(),
		Size:         100,
	}

	storageClient := &mockStorageClient{
		downloadBlobFunc: func(ctx context.Context, name string) ([]byte, error) {
			return []byte("test data"), nil
		},
		deleteBlobFunc: func(ctx context.Context, name string) error {
			return nil
		},
	}

	consumer := &mockConsumer{
		consumeFunc: func(ctx context.Context, data []byte) error {
			return nil
		},
	}

	receiver, err := newMetricsReceiver(component.NewID("test"), zap.NewNop(), cfg, consumer.NewNopMetrics())
	require.NoError(t, err)
	require.NotNil(t, receiver)

	receiver.storageClient = storageClient
	receiver.consumer = consumer

	err = receiver.processBlob(context.Background(), blob)
	assert.NoError(t, err)
}

type mockHost struct {
	component.Host
	extensions map[component.ID]component.Component
}

func (h *mockHost) GetExtensions() map[component.ID]component.Component {
	return h.extensions
} 