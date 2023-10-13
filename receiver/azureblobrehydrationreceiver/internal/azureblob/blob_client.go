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

// Package azureblob contains client interfaces and implementations for accessing Blob storage
package azureblob //import "github.com/observiq/bindplane-agent/receiver/azureblobrehydrationreceiver/internal/azureblob"

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// BlobInfo contains the necessary info to process a blob
type BlobInfo struct {
	Name string
	Size int64
}

// BlobClient provides a client for Blob operations
//
//go:generate mockery --name BlobClient --output ./mocks --with-expecter --filename mock_blob_client.go --structname MockBlobClient
type BlobClient interface {
	// ListBlobs returns a list of blobInfo objects present in the container with the given prefix
	ListBlobs(ctx context.Context, container string, prefix, marker *string) ([]*BlobInfo, *string, error)

	// DownloadBlob downloads the contents of the blob into the supplied buffer.
	// It will return the count of bytes used in the buffer.
	DownloadBlob(ctx context.Context, container, blobPath string, buf []byte) (int64, error)

	// DeleteBlob deletes the blob in the specified container
	DeleteBlob(ctx context.Context, container, blobPath string) error
}

// AzureClient is an implementation of the BlobClient for Azure
type AzureClient struct {
	azClient *azblob.Client
}

// NewAzureBlobClient creates a new azureBlobClient with the given connection string
func NewAzureBlobClient(connectionString string) (BlobClient, error) {
	azClient, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, err
	}

	return &AzureClient{
		azClient: azClient,
	}, nil
}

// contentLengthKey key for the content length metadata
const contentLengthKey = "ContentLength"

// ListBlobs returns a list of blobInfo objects present in the container with the given prefix
func (a *AzureClient) ListBlobs(ctx context.Context, container string, prefix, marker *string) ([]*BlobInfo, *string, error) {
	listOptions := &azblob.ListBlobsFlatOptions{
		Marker: marker,
		Prefix: prefix,
	}

	pager := a.azClient.NewListBlobsFlatPager(container, listOptions)

	var nextMarker *string
	blobs := make([]*BlobInfo, 0)
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("listBlobs: %w", err)
		}

		for _, blob := range resp.Segment.BlobItems {
			// Skip deleted blobs
			if blob.Deleted != nil && *blob.Deleted {
				continue
			}
			// All blob fields are pointers so check all pointers we need before we try to process it
			if blob.Name == nil || blob.Properties == nil || blob.Properties.ContentLength == nil {
				continue
			}

			info := &BlobInfo{
				Name: *blob.Name,
				Size: *blob.Properties.ContentLength,
			}

			blobs = append(blobs, info)
		}
		nextMarker = resp.NextMarker
	}

	return blobs, nextMarker, nil
}

// DownloadBlob downloads the contents of the blob into the supplied buffer.
// It will return the count of bytes used in the buffer.
func (a *AzureClient) DownloadBlob(ctx context.Context, container, blobPath string, buf []byte) (int64, error) {
	bytesDownloaded, err := a.azClient.DownloadBuffer(ctx, container, blobPath, buf, nil)
	if err != nil {
		return 0, fmt.Errorf("download: %w", err)
	}

	return bytesDownloaded, nil
}

// DeleteBlob deletes the blob in the specified container
func (a *AzureClient) DeleteBlob(ctx context.Context, container, blobPath string) error {
	_, err := a.azClient.DeleteBlob(ctx, container, blobPath, nil)
	return err
}
