package googlecloudstorageexporter // import "github.com/observiq/bindplane-agent/exporter/googlecloudstorageexporter"

import (
	"context"

	"cloud.google.com/go/storage"
)

// storageClient is a wrapper for an Google Cloud Storage client to allow mocking for testing.
//
//go:generate mockery --name storageClient --output ./internal/mocks --with-expecter --filename mock_storage_client.go --structname mockStorageClient
type storageClient interface {
	// UploadBuffer uploads a buffer in blocks to a block blob.
	UploadBuffer(context.Context, string, string, []byte) error
}

// googleCloudStorageClient is the google cloud storage implementation of the storageClient
type googleCloudStorageClient struct {
	storageClient *storage.Client
}

// newGoogleCloudStorageClient creates a new googleCloudStorageClient with the given connection string
func newGoogleCloudStorageClient(connectionString string) (*googleCloudStorageClient, error) {
	storageClient, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, err
	}

	return &googleCloudStorageClient{
		storageClient: storageClient,
	}, nil
}

func (a *googleCloudStorageClient) UploadBuffer(ctx context.Context, containerName, blobName string, buffer []byte) error {
	_, err := a.storageClient.UploadBuffer(ctx, containerName, blobName, buffer, nil)
	return err
}