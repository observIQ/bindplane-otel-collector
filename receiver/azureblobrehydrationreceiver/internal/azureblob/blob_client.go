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
package azureblob //import "github.com/observiq/bindplane-otel-collector/receiver/azureblobrehydrationreceiver/internal/azureblob"

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
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
	// DownloadBlob downloads the contents of the blob into the supplied buffer.
	// It will return the count of bytes used in the buffer.
	DownloadBlob(ctx context.Context, container, blobPath string, buf []byte) (int64, error)

	// DeleteBlob deletes the blob in the specified container
	DeleteBlob(ctx context.Context, container, blobPath string) error

	// StreamBlobs will stream BlobInfo to the blobChan and errors to the errChan, generally if an errChan gets an item
	// then the stream should be stopped
	StreamBlobs(ctx context.Context, container string, prefix *string, errChan chan error, blobChan chan *BlobResults, doneChan chan struct{})
}

type blobClient interface {
	NewListBlobsFlatPager(containerName string, options *azblob.ListBlobsFlatOptions) *runtime.Pager[azblob.ListBlobsFlatResponse]
	DownloadBuffer(ctx context.Context, containerName string, blobPath string, buffer []byte, options *azblob.DownloadBufferOptions) (int64, error)
	DeleteBlob(ctx context.Context, containerName string, blobPath string, options *azblob.DeleteBlobOptions) (azblob.DeleteBlobResponse, error)
}

var _ blobClient = &azblob.Client{}

// AzureClient is an implementation of the BlobClient for Azure
type AzureClient struct {
	azClient  blobClient
	batchSize int
	pageSize  int32
}

// NewAzureBlobClient creates a new azureBlobClient with the given connection string
func NewAzureBlobClient(connectionString string, batchSize, pageSize int) (BlobClient, error) {
	azClient, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, err
	}

	return &AzureClient{
		azClient:  azClient,
		batchSize: batchSize,
		pageSize:  int32(pageSize),
	}, nil
}

const emptyPollLimit = 3

// BlobResults contains the blobs for the receiver to process and the last marker
type BlobResults struct {
	Blobs      []*BlobInfo
	LastMarker *string
}

// StreamBlobs will stream blobs to the blobChan and errors to the errChan, generally if an errChan gets an item
// then the stream should be stopped
func (a *AzureClient) StreamBlobs(ctx context.Context, container string, prefix *string, errChan chan error, blobChan chan *BlobResults, doneChan chan struct{}) {
	var marker *string
	pager := a.azClient.NewListBlobsFlatPager(container, &azblob.ListBlobsFlatOptions{
		Marker:     marker,
		Prefix:     prefix,
		MaxResults: &a.pageSize,
	})

	emptyPollCount := 0
	for pager.More() {
		select {
		case <-ctx.Done():
			return
		default:
			// If we had empty polls for the last 3 times, then we can assume that there are no more blobs to process
			// and we can close the stream to avoid charging for the requests
			if emptyPollCount == emptyPollLimit {
				close(doneChan)
				return
			}

			resp, err := pager.NextPage(ctx)
			if err != nil {
				errChan <- fmt.Errorf("error streaming blobs: %w", err)
				return
			}

			batch := []*BlobInfo{}
			for _, blob := range resp.Segment.BlobItems {
				if blob.Deleted != nil && *blob.Deleted {
					continue
				}
				if blob.Name == nil || blob.Properties == nil || blob.Properties.ContentLength == nil {
					continue
				}

				info := &BlobInfo{
					Name: *blob.Name,
					Size: *blob.Properties.ContentLength,
				}
				batch = append(batch, info)
				if len(batch) == int(a.batchSize) {
					blobChan <- &BlobResults{
						Blobs:      batch,
						LastMarker: marker,
					}
					batch = []*BlobInfo{}
				}
			}

			if len(batch) == 0 {
				emptyPollCount++
				continue
			}

			emptyPollCount = 0
			blobChan <- &BlobResults{
				Blobs:      batch,
				LastMarker: marker,
			}
			marker = resp.NextMarker
		}
	}

	close(doneChan)
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
