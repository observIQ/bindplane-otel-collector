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
	"sync"
	"testing"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/azureblob"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
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
		newAzureBlobClient = func(_ string, _ int, _ int) (azureblob.BlobClient, error) {
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
		newAzureBlobClient = func(_ string, _ int, _ int) (azureblob.BlobClient, error) {
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
