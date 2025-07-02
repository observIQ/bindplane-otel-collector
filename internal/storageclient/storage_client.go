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

// Package storageclient contains interfaces and implementations used for storing data
package storageclient //import "github.com/observiq/bindplane-otel-collector/internal/storageclient"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/collector/pipeline"
)

// StorageClient handles storing of data for various use cases
type StorageClient interface {
	// SaveStorageData saves the supplied data
	SaveStorageData(ctx context.Context, key string, data StorageData) error

	// LoadStorageData loads data for the passed in key.
	// If no data is found return an empty one
	LoadStorageData(ctx context.Context, key string, data StorageData) error

	// Close closes the storage client
	Close(ctx context.Context) error
}

// StorageData is an interface that can be used to marshal and unmarshal data to and from a storage client
type StorageData interface {
	// Marshal marshals the data to a byte slice
	Marshal() ([]byte, error)

	// Unmarshal unmarshals the data from a byte slice
	Unmarshal([]byte) error
}

// NopStorage is a nop implementation of StorageClient
type NopStorage struct{}

// NewNopStorage creates a new NopStorage instance
func NewNopStorage() *NopStorage {
	return &NopStorage{}
}

// SaveStorageData returns nil
func (n *NopStorage) SaveStorageData(_ context.Context, _ string, _ StorageData) error {
	return nil
}

// LoadStorageData returns nil
func (n *NopStorage) LoadStorageData(_ context.Context, _ string, _ StorageData) error {
	return nil
}

// Close returns nil
func (n *NopStorage) Close(_ context.Context) error {
	return nil
}

// Storage is an implementation of StorageClient backed by a storage extension
type Storage struct {
	storageClient storage.Client
}

// NewStorageClient creates a new StorageClient based on the storage and component IDs
func NewStorageClient(ctx context.Context, host component.Host, storageID, componentID component.ID, pipelineSignal pipeline.Signal) (StorageClient, error) {
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

	return &Storage{
		storageClient: client,
	}, nil
}

// SaveStorageData saves the supplied data
func (c *Storage) SaveStorageData(ctx context.Context, key string, data StorageData) error {
	bytes, err := data.Marshal()
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	return c.storageClient.Set(ctx, key, bytes)
}

// LoadStorageData loads data for the passed in key.
func (c *Storage) LoadStorageData(ctx context.Context, key string, data StorageData) error {
	bytes, err := c.storageClient.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}

	if err := data.Unmarshal(bytes); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return nil
}

// Close closes the storage client
func (c *Storage) Close(ctx context.Context) error {
	return c.storageClient.Close(ctx)
}
