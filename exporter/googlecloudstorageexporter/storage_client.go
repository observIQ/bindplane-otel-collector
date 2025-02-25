package googlecloudstorageexporter // import "github.com/observiq/bindplane-agent/exporter/googlecloudstorageexporter"

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// storageClient is a wrapper for a Google Cloud Storage client to allow mocking for testing.
//
//go:generate mockery --name storageClient --output ./internal/mocks --with-expecter --filename mock_storage_client.go --structname mockStorageClient
type storageClient interface {
	// CreateBucket(ctx context.Context, projectID string, bucketName string, storageClass string, location string) error
	UploadObject(ctx context.Context, projectID string, bucketName string, objectName string, storageClass string, location string, buffer []byte) error
	ListBuckets(ctx context.Context, projectID string) ([]string, error)
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

func (c *googleCloudStorageClient) ListBuckets(ctx context.Context, projectID string) ([]string, error) {
	var bucketNames []string
	it := c.storageClient.Buckets(ctx, projectID)
	for {
		bucketAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing buckets: %w", err)
		}
		bucketNames = append(bucketNames, bucketAttrs.Name)
	}
	return bucketNames, nil
}

func (c *googleCloudStorageClient) UploadObject(ctx context.Context, projectID string, bucketName string, objectName string, storageClass string, location string, buffer []byte) error {
	// First, list available buckets and log them
	buckets, err := c.ListBuckets(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to list buckets before upload: %w", err)
	}
	
	// Log available buckets
	zap.L().Info("Available buckets in project",
		zap.String("project_id", projectID),
		zap.Strings("buckets", buckets),
		zap.String("target_bucket", bucketName))
	
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

// func (c *googleCloudStorageClient) CreateBucket(ctx context.Context, projectID string, bucketName string, storageClass string, location string) error {
// 	bucket := c.storageClient.Bucket(bucketName)
	
// 	storageClassAndLocation := &storage.BucketAttrs{
// 		StorageClass: storageClass,
// 		Location:     location,
// 	}
	
// 	if err := bucket.Create(ctx, projectID, storageClassAndLocation); err != nil {
// 		// Check if the error is because the bucket already exists
// 		if e, ok := err.(*googleapi.Error); ok && e.Code == 409 {
// 			return nil
// 		}
// 		return fmt.Errorf("failed to create bucket %q: %w", bucketName, err)
// 	}
// 	return nil
// }