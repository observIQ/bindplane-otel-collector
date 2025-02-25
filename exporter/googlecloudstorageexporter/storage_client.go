package googlecloudstorageexporter // import "github.com/observiq/bindplane-agent/exporter/googlecloudstorageexporter"

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// storageClient is a wrapper for a Google Cloud Storage client to allow mocking for testing.
//
//go:generate mockery --name storageClient --output ./internal/mocks --with-expecter --filename mock_storage_client.go --structname mockStorageClient
type storageClient interface {
	CreateBucket(ctx context.Context, projectID string, bucketName string, storageClass string, location string) error
	UploadObject(ctx context.Context, bucketName string, objectName string, buffer []byte) error
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

func (c *googleCloudStorageClient) UploadObject(ctx context.Context, bucketName string, objectName string, buffer []byte) error {
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

func (c *googleCloudStorageClient) CreateBucket(ctx context.Context, projectID string, bucketName string, storageClass string, location string) error {
	bucket := c.storageClient.Bucket(bucketName)
	
	// Check if bucket already exists
	_, err := bucket.Attrs(ctx)
	if err == nil {
		// Bucket already exists
		return nil
	}
	if err != storage.ErrBucketNotExist {
		// Return error if it's not a "bucket not exist" error
		return fmt.Errorf("failed to check bucket %q: %w", bucketName, err)
	}

	// Bucket doesn't exist, create it
	storageClassAndLocation := &storage.BucketAttrs{
		StorageClass: storageClass,
		Location:     location,
	}
	if err := bucket.Create(ctx, projectID, storageClassAndLocation); err != nil {
		return fmt.Errorf("failed to create bucket %q: %w", bucketName, err)
	}
	return nil
}