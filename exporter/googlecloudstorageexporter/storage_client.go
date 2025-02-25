package googlecloudstorageexporter // import "github.com/observiq/bindplane-agent/exporter/googlecloudstorageexporter"

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

// storageClient is a wrapper for an Google Cloud Storage client to allow mocking for testing.
//
//go:generate mockery --name storageClient --output ./internal/mocks --with-expecter --filename mock_storage_client.go --structname mockStorageClient
type storageClient interface {
	// UploadBuffer uploads a buffer in blocks to a block blob.
	UploadBuffer(context.Context, string, string, []byte) error
	CreateBucket(context.Context, string, string) error
}

// googleCloudStorageClient is the google cloud storage implementation of the storageClient
type googleCloudStorageClient struct {
	storageClient *storage.Client
}

// newGoogleCloudStorageClient creates a new googleCloudStorageClient with the given connection string
func newGoogleCloudStorageClient(connectionString string) (*googleCloudStorageClient, error) {
	ctx := context.Background()

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}
	// defer close?

	return &googleCloudStorageClient{
		storageClient: storageClient,
	}, nil
}

func (a *googleCloudStorageClient) UploadBuffer(ctx context.Context, containerName, blobName string, buffer []byte) error {
	bucket := a.storageClient.Bucket(containerName)
	obj := bucket.Object(blobName)
	writer := obj.NewWriter(ctx)
	
	if _, err := writer.Write(buffer); err != nil {
		return fmt.Errorf("failed to write to bucket %q: %w", containerName, err)
	}
	
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer for bucket %q: %w", containerName, err)
	}
	
	return nil
}

func (a *googleCloudStorageClient) createBucket(ctx context.Context, projectID, bucketName string) error {
	bucket := a.storageClient.Bucket(bucketName)
	
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
		StorageClass: "COLDLINE",
		Location:     "asia",
	}
	if err := bucket.Create(ctx, projectID, storageClassAndLocation); err != nil {
		return fmt.Errorf("failed to create bucket %q: %w", bucketName, err)
	}
	return nil
}