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
	"fmt"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// BlobInfo contains information about a Google Cloud Storage object
type BlobInfo struct {
	Name string
	Size int64
}

// StorageClient is an interface for interacting with Google Cloud Storage
//
//go:generate mockery --name StorageClient --output ./mocks --with-expecter --filename mock_storage_client.go --structname MockStorageClient
type StorageClient interface {
	// DownloadBlob downloads a blob by name
	DownloadBlob(ctx context.Context, name string) ([]byte, error)
	// DeleteBlob deletes a blob by name
	DeleteBlob(ctx context.Context, name string) error
	// StreamBlobs will stream BlobInfo to the blobChan and errors to the errChan, generally if an errChan gets an item
	// then the stream should be stopped
	StreamBlobs(ctx context.Context, startTime, endTime time.Time, errChan chan error, blobChan chan []*BlobInfo, doneChan chan struct{})
}

// googleCloudStorageClient implements the StorageClient interface
type googleCloudStorageClient struct {
	client     *storage.Client
	bucket     *storage.BucketHandle
	config     *Config
	batchSize  int
}

// NewStorageClient creates a new Google Cloud Storage client
func NewStorageClient(cfg *Config) (StorageClient, error) {
	ctx := context.Background()
	var opts []option.ClientOption

	// Handle credentials if provided, otherwise use default credentials
	switch {
	case cfg.Credentials != "":
		opts = append(opts, option.WithCredentialsJSON([]byte(cfg.Credentials)))
		if cfg.ProjectID == "" {
			creds, err := google.CredentialsFromJSON(ctx, []byte(cfg.Credentials))
			if err != nil {
				return nil, fmt.Errorf("credentials from json: %w", err)
			}
			cfg.ProjectID = creds.ProjectID
		}
	case cfg.CredentialsFile != "":
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
		if cfg.ProjectID == "" {
			credBytes, err := os.ReadFile(cfg.CredentialsFile)
			if err != nil {
				return nil, fmt.Errorf("read credentials file: %w", err)
			}
			creds, err := google.CredentialsFromJSON(ctx, credBytes)
			if err != nil {
				return nil, fmt.Errorf("credentials from json: %w", err)
			}
			cfg.ProjectID = creds.ProjectID
		}
	default:
		// Find application default credentials from the environment
		creds, err := google.FindDefaultCredentials(ctx, storage.ScopeReadOnly)
		if err != nil {
			return nil, fmt.Errorf("find default credentials: %w", err)
		}
		opts = append(opts, option.WithCredentials(creds))
		if cfg.ProjectID == "" {
			cfg.ProjectID = creds.ProjectID
		}
	}

	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("project_id not set in config and could not be read from credentials")
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}

	return &googleCloudStorageClient{
		client:    client,
		bucket:    client.Bucket(cfg.BucketName),
		config:    cfg,
		batchSize: cfg.BatchSize,
	}, nil
}

// StreamBlobs streams blobs from the bucket within the given time range
func (g *googleCloudStorageClient) StreamBlobs(ctx context.Context, startTime, endTime time.Time, errChan chan error, blobChan chan []*BlobInfo, doneChan chan struct{}) {
	it := g.bucket.Objects(ctx, &storage.Query{
		Prefix: g.config.FolderName,
	})

	batch := []*BlobInfo{}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		attrs, err := it.Next()
		if err == iterator.Done {
			// Send any remaining blobs in the batch
			if len(batch) > 0 {
				blobChan <- batch
			}
			close(doneChan)
			return
		}
		if err != nil {
			errChan <- fmt.Errorf("iterator.Next: %w", err)
			return
		}

		// Skip objects outside the time range
		if attrs.Updated.Before(startTime) || attrs.Updated.After(endTime) {
			continue
		}

		batch = append(batch, &BlobInfo{
			Name: attrs.Name,
			Size: attrs.Size,
		})

		// Send batch when it reaches the batch size
		if len(batch) == g.batchSize {
			blobChan <- batch
			batch = []*BlobInfo{}
		}
	}
}

// DownloadBlob downloads a blob by name
func (g *googleCloudStorageClient) DownloadBlob(ctx context.Context, name string) ([]byte, error) {
	obj := g.bucket.Object(name)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("object.NewReader: %w", err)
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// DeleteBlob deletes the specified object from the bucket
func (g *googleCloudStorageClient) DeleteBlob(ctx context.Context, name string) error {
	obj := g.bucket.Object(name)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("object.Delete: %w", err)
	}
	return nil
}
