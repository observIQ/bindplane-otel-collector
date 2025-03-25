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

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// ObjectInfo contains information about a Google Cloud Storage object
type ObjectInfo struct {
	Name string
	Size int64
}

// StorageClient is an interface for interacting with Google Cloud Storage
//
//go:generate mockery --name StorageClient --output ./mocks --with-expecter --filename mock_storage_client.go --structname MockStorageClient
type StorageClient interface {
	// DownloadObject downloads the contents of the object into the supplied buffer.
	// It will return the count of bytes used in the buffer.
	DownloadObject(ctx context.Context, name string, buf []byte) (int64, error)
	// DeleteObject deletes an object by name
	DeleteObject(ctx context.Context, name string) error
	// StreamObjects will stream ObjectInfo to the objectChan and errors to the errChan, generally if an errChan gets an item
	// then the stream should be stopped
	StreamObjects(ctx context.Context, errChan chan error, objectChan chan []*ObjectInfo, doneChan chan struct{})
}

// GoogleCloudStorageClient implements the StorageClient interface
type GoogleCloudStorageClient struct {
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

	return &GoogleCloudStorageClient{
		client:    client,
		bucket:    client.Bucket(cfg.BucketName),
		config:    cfg,
		batchSize: cfg.BatchSize,
	}, nil
}

// StreamObjects streams objects from the bucket within the given time range
func (g *GoogleCloudStorageClient) StreamObjects(ctx context.Context, errChan chan error, objectChan chan []*ObjectInfo, doneChan chan struct{}) {
	it := g.bucket.Objects(ctx, &storage.Query{
		Prefix: g.config.FolderName,
	})

	batch := []*ObjectInfo{}
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
				objectChan <- batch
			}
			close(doneChan)
			return
		}
		if err != nil {
			errChan <- fmt.Errorf("iterator.Next: %w", err)
			return
		}

		batch = append(batch, &ObjectInfo{
			Name: attrs.Name,
			Size: attrs.Size,
		})

		// Send batch when it reaches the batch size
		if len(batch) == g.batchSize {
			objectChan <- batch
			batch = []*ObjectInfo{}
		}
	}
}

// DownloadObject downloads the contents of the object into the supplied buffer.
// It will return the count of bytes used in the buffer.
func (g *GoogleCloudStorageClient) DownloadObject(ctx context.Context, name string, buf []byte) (int64, error) {
	obj := g.bucket.Object(name)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return 0, fmt.Errorf("object.NewReader: %w", err)
	}
	defer reader.Close()

	n, err := io.ReadFull(reader, buf)
	if err != nil {
		return 0, fmt.Errorf("read object: %w", err)
	}

	return int64(n), nil
}

// DeleteObject deletes the specified object from the bucket
func (g *GoogleCloudStorageClient) DeleteObject(ctx context.Context, name string) error {
	obj := g.bucket.Object(name)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("object.Delete: %w", err)
	}
	return nil
}
