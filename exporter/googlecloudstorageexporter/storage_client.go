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

package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter"

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// storageClient is a wrapper for a Google Cloud Storage client to allow mocking for testing.
//
//go:generate mockery --name storageClient --output ./internal/mocks --with-expecter --filename mock_storage_client.go --structname MockStorageClient
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
		creds, err := google.FindDefaultCredentials(ctx, storage.ScopeFullControl)
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

	storageClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}

	return &googleCloudStorageClient{
		storageClient: storageClient,
		config:        cfg,
	}, nil
}

// UploadObject will try to write to the bucket. It will create the bucket if it doesn't exist
func (c *googleCloudStorageClient) UploadObject(ctx context.Context, objectName string, buffer []byte) error {
	bucket := c.storageClient.Bucket(c.config.BucketName)
	obj := bucket.Object(objectName)

	// First attempt to write
	if err := c.writeToObject(ctx, obj, buffer); err != nil {
		// If bucket doesn't exist, try to create it and write again
		if isBucketNotFoundError(err) {
			if err := c.createBucket(ctx); err != nil {
				return fmt.Errorf("create bucket %q: %w", c.config.BucketName, err)
			}

			// Try writing again after bucket creation
			if err := c.writeToObject(ctx, obj, buffer); err != nil {
				return fmt.Errorf("write to bucket %q after creation: %w", c.config.BucketName, err)
			}
		} else {
			return fmt.Errorf("write to bucket %q: %w", c.config.BucketName, err)
		}
	}

	return nil
}

// writeToObject attempts to write data to a GCS object and waits for completion
func (c *googleCloudStorageClient) writeToObject(ctx context.Context, obj *storage.ObjectHandle, buffer []byte) error {
	writer := obj.NewWriter(ctx)

	if _, err := writer.Write(buffer); err != nil {
		// If Write returns an error, we should still try to close and check for close error
		if closeErr := writer.Close(); closeErr != nil {
			return fmt.Errorf("write: %v, close: %v", err, closeErr)
		}
		return fmt.Errorf("write to object %q: %w", obj.ObjectName(), err)
	}

	// Always check Close error as the source of truth for write success. Err is handled by UploadObject
	return writer.Close()
}

// createBucket will create a bucket in the project
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
