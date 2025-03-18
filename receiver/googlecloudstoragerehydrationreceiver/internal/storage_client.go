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

package internal

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/observiq/bindplane-otel-collector/receiver/googlecloudstoragerehydrationreceiver"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// BlobInfo contains information about a Google Cloud Storage object
type BlobInfo struct {
	Name         string
	LastModified time.Time
	Size         int64
}

// StorageClient is an interface for interacting with Google Cloud Storage
type StorageClient interface {
	// ListBlobs lists all blobs in the bucket within the given time range
	ListBlobs(ctx context.Context, startTime, endTime time.Time) ([]*BlobInfo, error)
	// DownloadBlob downloads a blob by name
	DownloadBlob(ctx context.Context, name string) ([]byte, error)
	// DeleteBlob deletes a blob by name
	DeleteBlob(ctx context.Context, name string) error
}

// googleCloudStorageClient implements the StorageClient interface
type googleCloudStorageClient struct {
	client     *storage.Client
	bucket     *storage.BucketHandle
	config     *googlecloudstoragerehydrationreceiver.Config
}

// NewStorageClient creates a new Google Cloud Storage client
func NewStorageClient(cfg *googlecloudstoragerehydrationreceiver.Config) (StorageClient, error) {
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
		client: client,
		bucket: client.Bucket(cfg.BucketName),
		config: cfg,
	}, nil
}

// ListBlobs lists all blobs in the bucket within the given time range
func (c *googleCloudStorageClient) ListBlobs(ctx context.Context, startTime, endTime time.Time) ([]*BlobInfo, error) {
	var blobs []*BlobInfo
	it := c.bucket.Objects(ctx, &storage.Query{
		Prefix: c.config.RootFolder,
	})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterator.Next: %w", err)
		}

		// Skip objects outside the time range
		if attrs.Updated.Before(startTime) || attrs.Updated.After(endTime) {
			continue
		}

		blobs = append(blobs, &BlobInfo{
			Name:         attrs.Name,
			LastModified: attrs.Updated,
			Size:         attrs.Size,
		})
	}

	return blobs, nil
}

// DownloadBlob downloads a blob by name
func (c *googleCloudStorageClient) DownloadBlob(ctx context.Context, name string) ([]byte, error) {
	obj := c.bucket.Object(name)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("object.NewReader: %w", err)
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// DeleteBlob deletes a blob by name
func (c *googleCloudStorageClient) DeleteBlob(ctx context.Context, name string) error {
	obj := c.bucket.Object(name)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("object.Delete: %w", err)
	}
	return nil
} 