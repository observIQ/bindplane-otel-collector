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

package googlecloudstorageexporter // import "github.com/observiq/bindplane-agent/exporter/googlecloudstorageexporter"

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// storageClient is a wrapper for a Google Cloud Storage client to allow mocking for testing.
//
//go:generate mockery --name storageClient --output ./internal/mocks --with-expecter --filename mock_storage_client.go --structname mockStorageClient
type storageClient interface {
	UploadObject(ctx context.Context, objectName string, buffer []byte) error
}

// googleCloudStorageClient is the google cloud storage implementation of the storageClient
type googleCloudStorageClient struct {
	storageClient *storage.Client
	config        *Config
}

// newGoogleCloudStorageClient creates a new googleCloudStorageClient with the given config
func newGoogleCloudStorageClient(cfg *Config) (*googleCloudStorageClient, error) {
	ctx := context.Background()
	var opts []option.ClientOption

	// Handle credentials if provided, otherwise use default credentials
	switch {
	case cfg.Credentials != "":
		opts = append(opts, option.WithCredentialsJSON([]byte(cfg.Credentials)))
	case cfg.CredentialsFile != "":
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	default:
		// Will use default credentials from the environment
	}

	storageClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}

	return &googleCloudStorageClient{
		storageClient: storageClient,
		config:        cfg,
	}, nil
}

func (c *googleCloudStorageClient) UploadObject(ctx context.Context, objectName string, buffer []byte) error {
	bucket := c.storageClient.Bucket(c.config.BucketName)
	obj := bucket.Object(objectName)
	writer := obj.NewWriter(ctx)
	
	// Try writing first
	if _, err := writer.Write(buffer); err != nil {
		// Check if error is due to missing bucket
		if isBucketNotFoundError(err) {
			// Try to create the bucket
			if err := c.createBucket(ctx); err != nil {
				// If creation failed because bucket exists in another project
				if isBucketExistsError(err) {
					return fmt.Errorf("bucket %q exists but is not accessible: %w", c.config.BucketName, err)
				}
				return fmt.Errorf("failed to create bucket %q: %w", c.config.BucketName, err)
			}
			
			// Bucket created, try writing again
			writer = obj.NewWriter(ctx)
			if _, err := writer.Write(buffer); err != nil {
				return fmt.Errorf("failed to write to bucket %q after creation: %w", c.config.BucketName, err)
			}
		} else {
			return fmt.Errorf("failed to write to bucket %q: %w", c.config.BucketName, err)
		}
	}
	
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer for bucket %q: %w", c.config.BucketName, err)
	}
	
	return nil
}

func (c *googleCloudStorageClient) createBucket(ctx context.Context) error {
	bucket := c.storageClient.Bucket(c.config.BucketName)
	
	storageClassAndLocation := &storage.BucketAttrs{
		StorageClass: c.config.BucketStorageClass,
		Location:     c.config.BucketLocation,
	}
	
	return bucket.Create(ctx, c.config.ProjectID, storageClassAndLocation)
}

// isBucketNotFoundError checks if the error indicates the bucket doesn't exist
func isBucketNotFoundError(err error) bool {
	if e, ok := err.(*googleapi.Error); ok {
		return e.Code == 404 && e.Message == "The specified bucket does not exist."
	}
	return false
}

// isBucketExistsError checks if the error indicates the bucket already exists
func isBucketExistsError(err error) bool {
	if e, ok := err.(*googleapi.Error); ok {
		return e.Code == 409
	}
	return false
}