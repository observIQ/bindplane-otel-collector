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

package azureblobpollingreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/azureblobpollingreceiver"

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/azureblob"
	"github.com/observiq/bindplane-otel-collector/internal/blobconsume"
	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pipeline"
	"go.uber.org/zap"
)

func TestPollingReceiver_runPoll(t *testing.T) {
	t.Run("First poll uses initial lookback", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Container:       "test-container",
			PollInterval:    1 * time.Minute,
			InitialLookback: 5 * time.Minute,
		}

		mockClient := new(azureblob.MockBlobClient)
		receiver := &pollingReceiver{
			logger:          logger,
			cfg:             cfg,
			azureClient:     mockClient,
			checkpoint:      NewPollingCheckpoint(),
			pollInterval:    cfg.PollInterval,
			initialLookback: cfg.InitialLookback,
			mut:             &sync.Mutex{},
			wg:              &sync.WaitGroup{},
		}

		// Verify that checkpoint has zero LastPollTime before first poll
		require.True(t, receiver.checkpoint.LastPollTime.IsZero())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Mock StreamBlobs to immediately send done signal
		mockClient.On("StreamBlobs", mock.Anything, "test-container", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				doneChan := args.Get(5).(chan struct{})
				close(doneChan)
			})

		receiver.runPoll(ctx)

		// Verify checkpoint was updated with poll time after first poll
		require.False(t, receiver.checkpoint.LastPollTime.IsZero())

		mockClient.AssertExpectations(t)
	})

	t.Run("Subsequent polls use last poll time", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Container:       "test-container",
			PollInterval:    1 * time.Minute,
			InitialLookback: 5 * time.Minute,
		}

		mockClient := new(azureblob.MockBlobClient)
		checkpoint := NewPollingCheckpoint()
		lastPollTime := time.Now().UTC().Add(-2 * time.Minute)
		checkpoint.UpdatePollTime(lastPollTime)

		receiver := &pollingReceiver{
			logger:          logger,
			cfg:             cfg,
			azureClient:     mockClient,
			checkpoint:      checkpoint,
			pollInterval:    cfg.PollInterval,
			initialLookback: cfg.InitialLookback,
			mut:             &sync.Mutex{},
			wg:              &sync.WaitGroup{},
		}

		// Verify checkpoint has the previous poll time
		require.Equal(t, lastPollTime, receiver.checkpoint.LastPollTime)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Mock StreamBlobs to immediately send done signal
		mockClient.On("StreamBlobs", mock.Anything, "test-container", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				doneChan := args.Get(5).(chan struct{})
				close(doneChan)
			})

		receiver.runPoll(ctx)

		// Verify checkpoint was updated with new poll time
		require.True(t, receiver.checkpoint.LastPollTime.After(lastPollTime))

		mockClient.AssertExpectations(t)
	})

	t.Run("Updates checkpoint after processing blobs", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Container:       "test-container",
			PollInterval:    1 * time.Minute,
			InitialLookback: 5 * time.Minute,
		}

		mockClient := new(azureblob.MockBlobClient)
		checkpoint := NewPollingCheckpoint()

		receiver := &pollingReceiver{
			logger:          logger,
			cfg:             cfg,
			azureClient:     mockClient,
			checkpoint:      checkpoint,
			pollInterval:    cfg.PollInterval,
			initialLookback: cfg.InitialLookback,
			mut:             &sync.Mutex{},
			wg:              &sync.WaitGroup{},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		now := time.Now().UTC()

		// Mock StreamBlobs to send some test blobs
		mockClient.On("StreamBlobs", mock.Anything, "test-container", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				blobChan := args.Get(4).(chan []*azureblob.BlobInfo)
				doneChan := args.Get(5).(chan struct{})

				// Send a batch of blobs
				blobs := []*azureblob.BlobInfo{
					{
						Name:         "year=2024/month=03/day=15/hour=14/test1.json",
						Size:         100,
						LastModified: now.Add(-1 * time.Hour),
					},
					{
						Name:         "year=2024/month=03/day=15/hour=15/test2.json",
						Size:         200,
						LastModified: now.Add(-30 * time.Minute),
					},
				}
				blobChan <- blobs
				close(doneChan)
			})

		beforePoll := receiver.checkpoint.LastPollTime
		receiver.runPoll(ctx)

		// Verify checkpoint poll time was updated
		require.True(t, receiver.checkpoint.LastPollTime.After(beforePoll))

		mockClient.AssertExpectations(t)
	})

	t.Run("Handles context cancellation", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Container:       "test-container",
			PollInterval:    1 * time.Minute,
			InitialLookback: 5 * time.Minute,
		}

		mockClient := new(azureblob.MockBlobClient)
		receiver := &pollingReceiver{
			logger:          logger,
			cfg:             cfg,
			azureClient:     mockClient,
			checkpoint:      NewPollingCheckpoint(),
			pollInterval:    cfg.PollInterval,
			initialLookback: cfg.InitialLookback,
			mut:             &sync.Mutex{},
			wg:              &sync.WaitGroup{},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Mock StreamBlobs to detect when called, then return from doneChan
		mockClient.On("StreamBlobs", mock.Anything, "test-container", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				doneChan := args.Get(5).(chan struct{})
				close(doneChan)
			})

		receiver.runPoll(ctx)

		mockClient.AssertExpectations(t)
	})

	t.Run("Handles error during poll", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Container:       "test-container",
			PollInterval:    1 * time.Minute,
			InitialLookback: 5 * time.Minute,
		}

		mockClient := new(azureblob.MockBlobClient)
		receiver := &pollingReceiver{
			logger:          logger,
			cfg:             cfg,
			azureClient:     mockClient,
			checkpoint:      NewPollingCheckpoint(),
			pollInterval:    cfg.PollInterval,
			initialLookback: cfg.InitialLookback,
			mut:             &sync.Mutex{},
			wg:              &sync.WaitGroup{},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Mock StreamBlobs to send an error
		mockClient.On("StreamBlobs", mock.Anything, "test-container", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				errChan := args.Get(3).(chan error)
				errChan <- context.Canceled
			})

		receiver.runPoll(ctx)

		mockClient.AssertExpectations(t)
	})
}

func TestPollingReceiver_GlobExpansion(t *testing.T) {
	t.Run("Glob root_folder expands to matched directories", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Container:              "test-container",
			RootFolder:             "linux/*",
			PollInterval:           1 * time.Minute,
			InitialLookback:        5 * time.Minute,
			UseTimePatternAsPrefix: true,
			TimePattern:            "{year}/{month}/{day}/{hour}",
		}

		mockClient := new(azureblob.MockBlobClient)
		receiver := &pollingReceiver{
			logger:          logger,
			cfg:             cfg,
			azureClient:     mockClient,
			checkpoint:      NewPollingCheckpoint(),
			pollInterval:    cfg.PollInterval,
			initialLookback: cfg.InitialLookback,
			mut:             &sync.Mutex{},
			wg:              &sync.WaitGroup{},
		}

		ctx := context.Background()

		// Mock ListPrefixes to return subdirectories
		mockClient.On("ListPrefixes", mock.Anything, "test-container", "linux/").
			Return([]string{"linux/auditd", "linux/logb", "linux/logc"}, nil)

		// Mock StreamBlobs for each matched directory's time prefixes
		mockClient.On("StreamBlobs", mock.Anything, "test-container", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				doneChan := args.Get(5).(chan struct{})
				close(doneChan)
			})

		receiver.runPoll(ctx)

		mockClient.AssertCalled(t, "ListPrefixes", mock.Anything, "test-container", "linux/")
		mockClient.AssertExpectations(t)
	})

	t.Run("No glob skips ListPrefixes API call", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Container:       "test-container",
			RootFolder:      "linux/auditd",
			PollInterval:    1 * time.Minute,
			InitialLookback: 5 * time.Minute,
		}

		mockClient := new(azureblob.MockBlobClient)
		receiver := &pollingReceiver{
			logger:          logger,
			cfg:             cfg,
			azureClient:     mockClient,
			checkpoint:      NewPollingCheckpoint(),
			pollInterval:    cfg.PollInterval,
			initialLookback: cfg.InitialLookback,
			mut:             &sync.Mutex{},
			wg:              &sync.WaitGroup{},
		}

		ctx := context.Background()

		// Only StreamBlobs should be called, NOT ListPrefixes
		mockClient.On("StreamBlobs", mock.Anything, "test-container", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				doneChan := args.Get(5).(chan struct{})
				close(doneChan)
			})

		receiver.runPoll(ctx)

		mockClient.AssertNotCalled(t, "ListPrefixes", mock.Anything, mock.Anything, mock.Anything)
		mockClient.AssertExpectations(t)
	})

	t.Run("Glob with zero matches scans nothing", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Container:       "test-container",
			RootFolder:      "nonexistent/*",
			PollInterval:    1 * time.Minute,
			InitialLookback: 5 * time.Minute,
		}

		mockClient := new(azureblob.MockBlobClient)
		receiver := &pollingReceiver{
			logger:          logger,
			cfg:             cfg,
			azureClient:     mockClient,
			checkpoint:      NewPollingCheckpoint(),
			pollInterval:    cfg.PollInterval,
			initialLookback: cfg.InitialLookback,
			mut:             &sync.Mutex{},
			wg:              &sync.WaitGroup{},
		}

		ctx := context.Background()

		// ListPrefixes returns empty — no directories match
		mockClient.On("ListPrefixes", mock.Anything, "test-container", "nonexistent/").
			Return([]string{}, nil)

		receiver.runPoll(ctx)

		// StreamBlobs should NOT be called — zero matches means scan nothing
		mockClient.AssertCalled(t, "ListPrefixes", mock.Anything, "test-container", "nonexistent/")
		mockClient.AssertNotCalled(t, "StreamBlobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		mockClient.AssertExpectations(t)
	})

	t.Run("ListPrefixes error falls back to literal root_folder", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &Config{
			Container:       "test-container",
			RootFolder:      "linux/*",
			PollInterval:    1 * time.Minute,
			InitialLookback: 5 * time.Minute,
		}

		mockClient := new(azureblob.MockBlobClient)
		receiver := &pollingReceiver{
			logger:          logger,
			cfg:             cfg,
			azureClient:     mockClient,
			checkpoint:      NewPollingCheckpoint(),
			pollInterval:    cfg.PollInterval,
			initialLookback: cfg.InitialLookback,
			mut:             &sync.Mutex{},
			wg:              &sync.WaitGroup{},
		}

		ctx := context.Background()

		// ListPrefixes returns error
		mockClient.On("ListPrefixes", mock.Anything, "test-container", "linux/").
			Return([]string(nil), errors.New("network error"))

		// StreamBlobs should be called with the literal root_folder as fallback
		mockClient.On("StreamBlobs", mock.Anything, "test-container", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				prefix := args.Get(2).(*string)
				require.NotNil(t, prefix)
				require.Equal(t, "linux/*", *prefix)
				doneChan := args.Get(5).(chan struct{})
				close(doneChan)
			})

		receiver.runPoll(ctx)

		mockClient.AssertExpectations(t)
	})
}

func TestPollingReceiver_InitialLookback(t *testing.T) {
	t.Run("Uses InitialLookback when configured", func(t *testing.T) {
		cfg := &Config{
			ConnectionString: "DefaultEndpointsProtocol=https;AccountName=test;AccountKey=dGVzdA==;EndpointSuffix=core.windows.net",
			Container:        "test-container",
			PollInterval:     1 * time.Minute,
			InitialLookback:  10 * time.Minute,
			BatchSize:        100,
			PageSize:         1000,
		}

		// Override the newAzureBlobClient function for testing
		originalNewAzureBlobClient := newAzureBlobClient
		defer func() { newAzureBlobClient = originalNewAzureBlobClient }()

		mockClient := new(azureblob.MockBlobClient)
		newAzureBlobClient = func(_ string, _ int, _ int, _ *zap.Logger) (azureblob.BlobClient, error) {
			return mockClient, nil
		}

		receiver, err := newLogsReceiver(
			component.MustNewID("azureblobpolling"),
			zap.NewNop(),
			cfg,
			consumertest.NewNop(),
		)
		require.NoError(t, err)
		require.NotNil(t, receiver)
		require.Equal(t, 10*time.Minute, receiver.initialLookback)
	})

	t.Run("Defaults to PollInterval when InitialLookback not set", func(t *testing.T) {
		cfg := &Config{
			ConnectionString: "DefaultEndpointsProtocol=https;AccountName=test;AccountKey=dGVzdA==;EndpointSuffix=core.windows.net",
			Container:        "test-container",
			PollInterval:     1 * time.Minute,
			InitialLookback:  0, // Not set
			BatchSize:        100,
			PageSize:         1000,
		}

		// Override the newAzureBlobClient function for testing
		originalNewAzureBlobClient := newAzureBlobClient
		defer func() { newAzureBlobClient = originalNewAzureBlobClient }()

		mockClient := new(azureblob.MockBlobClient)
		newAzureBlobClient = func(_ string, _ int, _ int, _ *zap.Logger) (azureblob.BlobClient, error) {
			return mockClient, nil
		}

		receiver, err := newLogsReceiver(
			component.MustNewID("azureblobpolling"),
			zap.NewNop(),
			cfg,
			consumertest.NewNop(),
		)
		require.NoError(t, err)
		require.NotNil(t, receiver)
		require.Equal(t, cfg.PollInterval, receiver.initialLookback)
	})
}

func TestNewLogsReceiver_BlobFormat(t *testing.T) {
	originalNewAzureBlobClient := newAzureBlobClient
	defer func() { newAzureBlobClient = originalNewAzureBlobClient }()

	mockClient := new(azureblob.MockBlobClient)
	newAzureBlobClient = func(_ string, _ int, _ int, _ *zap.Logger) (azureblob.BlobClient, error) {
		return mockClient, nil
	}

	baseCfg := func(format BlobFormat) *Config {
		return &Config{
			ConnectionString: "DefaultEndpointsProtocol=https;AccountName=test;AccountKey=dGVzdA==;EndpointSuffix=core.windows.net",
			Container:        "test-container",
			PollInterval:     1 * time.Minute,
			BatchSize:        100,
			PageSize:         1000,
			BlobFormat:       format,
		}
	}

	t.Run("Default format uses LogsConsumer", func(t *testing.T) {
		r, err := newLogsReceiver(component.MustNewID("azureblobpolling"), zap.NewNop(), baseCfg(""), consumertest.NewNop())
		require.NoError(t, err)
		require.IsType(t, &blobconsume.LogsConsumer{}, r.consumer)
	})

	t.Run("OTLP format uses LogsConsumer", func(t *testing.T) {
		r, err := newLogsReceiver(component.MustNewID("azureblobpolling"), zap.NewNop(), baseCfg(BlobFormatOTLP), consumertest.NewNop())
		require.NoError(t, err)
		require.IsType(t, &blobconsume.LogsConsumer{}, r.consumer)
	})

	t.Run("JSON format uses NDJSONLogsConsumer", func(t *testing.T) {
		r, err := newLogsReceiver(component.MustNewID("azureblobpolling"), zap.NewNop(), baseCfg(BlobFormatJSON), consumertest.NewNop())
		require.NoError(t, err)
		require.IsType(t, &blobconsume.NDJSONLogsConsumer{}, r.consumer)
	})

	t.Run("Text format uses RawTextLogsConsumer", func(t *testing.T) {
		r, err := newLogsReceiver(component.MustNewID("azureblobpolling"), zap.NewNop(), baseCfg(BlobFormatText), consumertest.NewNop())
		require.NoError(t, err)
		require.IsType(t, &blobconsume.RawTextLogsConsumer{}, r.consumer)
	})
}

func TestNewLogsReceiver_RejectsUnsupportedFormat(t *testing.T) {
	originalNewAzureBlobClient := newAzureBlobClient
	defer func() { newAzureBlobClient = originalNewAzureBlobClient }()

	mockClient := new(azureblob.MockBlobClient)
	newAzureBlobClient = func(_ string, _ int, _ int, _ *zap.Logger) (azureblob.BlobClient, error) {
		return mockClient, nil
	}

	cfg := &Config{
		ConnectionString: "DefaultEndpointsProtocol=https;AccountName=test;AccountKey=dGVzdA==;EndpointSuffix=core.windows.net",
		Container:        "test-container",
		PollInterval:     1 * time.Minute,
		BatchSize:        100,
		PageSize:         1000,
		BlobFormat:       "invalid_format",
	}

	_, err := newLogsReceiver(component.MustNewID("azureblobpolling"), zap.NewNop(), cfg, consumertest.NewNop())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported blob_format")
}

func TestNewMetricsReceiver_RejectsNonOTLP(t *testing.T) {
	originalNewAzureBlobClient := newAzureBlobClient
	defer func() { newAzureBlobClient = originalNewAzureBlobClient }()

	mockClient := new(azureblob.MockBlobClient)
	newAzureBlobClient = func(_ string, _ int, _ int, _ *zap.Logger) (azureblob.BlobClient, error) {
		return mockClient, nil
	}

	cfg := &Config{
		ConnectionString: "DefaultEndpointsProtocol=https;AccountName=test;AccountKey=dGVzdA==;EndpointSuffix=core.windows.net",
		Container:        "test-container",
		PollInterval:     1 * time.Minute,
		BatchSize:        100,
		PageSize:         1000,
		BlobFormat:       BlobFormatJSON,
	}

	_, err := newMetricsReceiver(component.MustNewID("azureblobpolling"), zap.NewNop(), cfg, consumertest.NewNop())
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported for metrics pipelines")
}

func TestNewTracesReceiver_RejectsNonOTLP(t *testing.T) {
	originalNewAzureBlobClient := newAzureBlobClient
	defer func() { newAzureBlobClient = originalNewAzureBlobClient }()

	mockClient := new(azureblob.MockBlobClient)
	newAzureBlobClient = func(_ string, _ int, _ int, _ *zap.Logger) (azureblob.BlobClient, error) {
		return mockClient, nil
	}

	cfg := &Config{
		ConnectionString: "DefaultEndpointsProtocol=https;AccountName=test;AccountKey=dGVzdA==;EndpointSuffix=core.windows.net",
		Container:        "test-container",
		PollInterval:     1 * time.Minute,
		BatchSize:        100,
		PageSize:         1000,
		BlobFormat:       BlobFormatText,
	}

	_, err := newTracesReceiver(component.MustNewID("azureblobpolling"), zap.NewNop(), cfg, consumertest.NewNop())
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported for traces pipelines")
}

func TestPollingReceiver_MultiBatchNoDataLoss(t *testing.T) {
	t.Run("All blobs across multiple batches are processed when batch 1 has latest timestamp", func(t *testing.T) {
		logger := zap.NewNop()
		now := time.Now().UTC()

		cfg := &Config{
			Container:       "test-container",
			PollInterval:    1 * time.Minute,
			InitialLookback: 5 * time.Minute,
			UseLastModified: true,
		}

		mockClient := new(azureblob.MockBlobClient)
		checkpoint := NewPollingCheckpoint()

		receiver := &pollingReceiver{
			logger:             logger,
			cfg:                cfg,
			azureClient:        mockClient,
			checkpoint:         checkpoint,
			checkpointStore:    storageclient.NewNopStorage(),
			pollInterval:       cfg.PollInterval,
			initialLookback:    cfg.InitialLookback,
			supportedTelemetry: pipeline.SignalLogs,
			consumer:           blobconsume.NewLogsConsumer(consumertest.NewNop()),
			mut:                &sync.Mutex{},
			wg:                 &sync.WaitGroup{},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Simulate 3 batches where batch 1 contains the blob with the latest timestamp.
		// Before the fix, the per-batch checkpoint would advance LastTs after batch 1,
		// causing blobs in batches 2 and 3 to be skipped.
		batch1 := []*azureblob.BlobInfo{
			{Name: "blob-latest.json", Size: 100, LastModified: now.Add(-1 * time.Second)}, // latest timestamp
			{Name: "blob-early1.json", Size: 100, LastModified: now.Add(-10 * time.Second)},
		}
		batch2 := []*azureblob.BlobInfo{
			{Name: "blob-early2.json", Size: 100, LastModified: now.Add(-15 * time.Second)},
			{Name: "blob-early3.json", Size: 100, LastModified: now.Add(-20 * time.Second)},
		}
		batch3 := []*azureblob.BlobInfo{
			{Name: "blob-early4.json", Size: 100, LastModified: now.Add(-25 * time.Second)},
			{Name: "blob-early5.json", Size: 100, LastModified: now.Add(-30 * time.Second)},
		}

		// Track which blobs were downloaded (i.e. processed)
		downloadedBlobs := make(map[string]bool)
		downloadMu := sync.Mutex{}

		// Valid minimal OTLP JSON logs payload
		validJSON := []byte(`{"resourceLogs":[]}`)

		mockClient.On("DownloadBlob", mock.Anything, "test-container", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				blobName := args.Get(2).(string)
				buf := args.Get(3).([]byte)
				copy(buf, validJSON)
				downloadMu.Lock()
				downloadedBlobs[blobName] = true
				downloadMu.Unlock()
			}).
			Return(int64(len(validJSON)), nil)

		// Mock StreamBlobs to send 3 batches sequentially
		mockClient.On("StreamBlobs", mock.Anything, "test-container", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				blobChan := args.Get(4).(chan []*azureblob.BlobInfo)
				doneChan := args.Get(5).(chan struct{})

				blobChan <- batch1
				blobChan <- batch2
				blobChan <- batch3
				close(doneChan)
			})

		receiver.runPoll(ctx)

		// All 6 blobs should have been downloaded
		downloadMu.Lock()
		defer downloadMu.Unlock()

		allBlobs := []string{
			"blob-latest.json", "blob-early1.json",
			"blob-early2.json", "blob-early3.json",
			"blob-early4.json", "blob-early5.json",
		}
		for _, name := range allBlobs {
			require.True(t, downloadedBlobs[name], "blob %s should have been processed but was skipped", name)
		}
		require.Equal(t, 6, len(downloadedBlobs), "all 6 blobs should be processed")

		// Checkpoint LastTs should be set to the latest timestamp (blob-latest.json)
		require.NotNil(t, receiver.lastBlobTime)
		require.Equal(t, now.Add(-1*time.Second), *receiver.lastBlobTime)
	})
}
