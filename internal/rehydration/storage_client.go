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

// Package rehydration contains interfaces and implementations used across all rehydration receivers
package rehydration //import "github.com/observiq/bindplane-otel-collector/internal/rehydration"

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/collector/pipeline"
)

// CheckpointStorer handles storing of checkpoints for rehydration receivers
type CheckpointStorer interface {
	// SaveCheckpoint saves the supplied checkpoint
	SaveCheckpoint(ctx context.Context, key string, checkpoint *CheckPoint) error

	// LoadCheckPoint loads a checkpoint for the passed in key.
	// If no checkpoint is found return an empty one
	LoadCheckPoint(ctx context.Context, key string) (*CheckPoint, error)

	// Close closes the storage client
	Close(ctx context.Context) error
}

// NopStorage a nop implementation of CheckpointStorer
type NopStorage struct{}

// NewNopStorage creates a new NopStorage instance
func NewNopStorage() *NopStorage {
	return &NopStorage{}
}

// SaveCheckpoint returns nil
func (n *NopStorage) SaveCheckpoint(_ context.Context, _ string, _ *CheckPoint) error {
	return nil
}

// LoadCheckPoint returns and empty checkpoint
func (n *NopStorage) LoadCheckPoint(_ context.Context, _ string) (*CheckPoint, error) {
	return &CheckPoint{}, nil
}

// Close returns nil
func (n *NopStorage) Close(_ context.Context) error {
	return nil
}

// CheckpointStorage is checkpoint storer backed by a storage extension
type CheckpointStorage struct {
	storageClient storage.Client
}

// NewCheckpointStorage creates a new CheckpointStorage based on the storage and component IDs
func NewCheckpointStorage(ctx context.Context, host component.Host, storageID, componentID component.ID, pipelineSignal pipeline.Signal) (*CheckpointStorage, error) {
	extension, ok := host.GetExtensions()[storageID]
	if !ok {
		return nil, fmt.Errorf("storage extension '%s' not found", storageID)
	}

	storageExtension, ok := extension.(storage.Extension)
	if !ok {
		return nil, fmt.Errorf("non-storage extension '%s' found", storageID)
	}

	client, err := storageExtension.GetClient(ctx, component.KindReceiver, componentID, pipelineSignal.String())
	if err != nil {
		return nil, fmt.Errorf("get client: %w", err)
	}

	return &CheckpointStorage{
		storageClient: client,
	}, nil
}

// SaveCheckpoint saves the supplied checkpoint
func (c *CheckpointStorage) SaveCheckpoint(ctx context.Context, key string, checkpoint *CheckPoint) error {
	data, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}

	return c.storageClient.Set(ctx, key, data)
}

// LoadCheckPoint loads a checkpoint for the passed in key.
// If no checkpoint is found return an empty one
func (c *CheckpointStorage) LoadCheckPoint(ctx context.Context, key string) (*CheckPoint, error) {
	checkpoint := NewCheckpoint()

	data, err := c.storageClient.Get(ctx, key)
	if err != nil {
		return checkpoint, fmt.Errorf("get: %w", err)
	}

	if data == nil {
		return checkpoint, nil
	}

	if err := json.Unmarshal(data, checkpoint); err != nil {
		return checkpoint, fmt.Errorf("unmarshal: %w", err)
	}

	return checkpoint, nil
}

// Close closes the checkpoint storage
func (c *CheckpointStorage) Close(ctx context.Context) error {
	return c.storageClient.Close(ctx)
}
