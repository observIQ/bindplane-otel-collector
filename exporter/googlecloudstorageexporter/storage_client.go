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
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// storageClient is a wrapper for a Google Cloud Storage client to allow mocking for testing.
//
//go:generate mockery --name storageClient --output ./internal/mocks --with-expecter --filename mock_storage_client.go --structname mockStorageClient
type storageClient interface {
	CreateBucket(ctx context.Context, projectID string, bucketName string, storageClass string, location string) error
	UploadObject(ctx context.Context, projectID string, bucketName string, objectName string, storageClass string, location string, buffer []byte) error
	BucketExists(ctx context.Context, projectID string, bucketName string) (bool, error)
}

// googleCloudStorageClient is the google cloud storage implementation of the storageClient
type googleCloudStorageClient struct {
	storageClient *storage.Client
}

// newGoogleCloudStorageClient creates a new googleCloudStorageClient with the given connection string
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
	}, nil
}

func (c *googleCloudStorageClient) BucketExists(ctx context.Context, projectID string, bucketName string) (bool, error) {
	it := c.storageClient.Buckets(ctx, projectID)
	for {
		bucketAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return false, fmt.Errorf("error listing buckets: %w", err)
		}
		if bucketAttrs.Name == bucketName {
			return true, nil
		}
	}
	return false, nil
}

func (c *googleCloudStorageClient) CreateBucket(ctx context.Context, projectID string, bucketName string, storageClass string, location string) error {
	bucket := c.storageClient.Bucket(bucketName)
	
	storageClassAndLocation := &storage.BucketAttrs{
		StorageClass: storageClass,
		Location:     location,
	}
	
	if err := bucket.Create(ctx, projectID, storageClassAndLocation); err != nil {
		return fmt.Errorf("failed to create bucket %q: %w", bucketName, err)
	}
	return nil
}

func (c *googleCloudStorageClient) UploadObject(ctx context.Context, projectID string, bucketName string, objectName string, storageClass string, location string, buffer []byte) error {
	// Check if bucket exists
	exists, err := c.BucketExists(ctx, projectID, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}
	
	// Create bucket if it doesn't exist
	if !exists {			
		if err := c.CreateBucket(ctx, projectID, bucketName, storageClass, location); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	
	bucket := c.storageClient.Bucket(bucketName)
	obj := bucket.Object(objectName)
	writer := obj.NewWriter(ctx)
	
	if _, err := writer.Write(buffer); err != nil {
		return fmt.Errorf("failed to write to bucket %q: %w", bucketName, err)
	}
	
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer for bucket %q: %w", bucketName, err)
	}
	
	return nil
}