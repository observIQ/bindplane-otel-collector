// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package aws provides an S3 client for AWS
package aws //import "github.com/observiq/bindplane-otel-collector/receiver/awss3rehydrationreceiver/internal/aws"

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
)

// S3Client provides a client for S3 object operations
//
//go:generate mockery --name S3Client --output ./mocks --with-expecter --filename mock_s3_client.go --structname MockS3Client
type S3Client interface {
	// StreamObjects continuously pulls & batches ObjectInfo from S3 before sending down the objectChan.
	// Errors are sent down the errChan and stop the process. When no more objects are retrieved from S3, the doneChan is closed.
	StreamObjects(ctx context.Context, bucket, prefix string, objectChan chan *ObjectResults, errChan chan error, doneChan chan struct{})
	// DownloadObject downloads the contents of the object into the buffer.
	DownloadObject(ctx context.Context, bucket, key string, buf []byte) (int64, error)
	// DeleteObject deletes the object with the given key in the specified bucket
	DeleteObject(ctx context.Context, bucket string, key string) error
}

// ObjectInfo contains necessary info to process S3 objects
type ObjectInfo struct {
	Name string
	Size int64
}

// ObjectResults contain a batch of ObjectInfo
type ObjectResults struct {
	Objects []*ObjectInfo
}

// DefaultClient is an implementation of the S3Client for AWS
type DefaultClient struct {
	logger *zap.Logger
	client s3.Client

	pollSize  int
	batchSize int
}

// NewAWSClient creates a new AWS Client
func NewAWSClient(logger *zap.Logger, region string, pollSize, batchSize int) (S3Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("aws load default config: %w", err)
	}

	return &DefaultClient{
		logger:    logger,
		client:    *s3.NewFromConfig(cfg),
		pollSize:  pollSize,
		batchSize: batchSize,
	}, nil
}

func (a *DefaultClient) StreamObjects(ctx context.Context, bucket, prefix string, objectChan chan *ObjectResults, errChan chan error, doneChan chan struct{}) {
	objectPaginator := s3.NewListObjectsV2Paginator(&a.client, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(int32(a.pollSize)),
	})

	for objectPaginator.HasMorePages() {
		// check context
		select {
		case <-ctx.Done():
			a.logger.Info("Context finished, stopping stream")
			return
		default:
		}

		output, err := objectPaginator.NextPage(ctx)
		if err != nil {
			errChan <- fmt.Errorf("next page: %w", err)
			return
		}

		batch := []*ObjectInfo{}
		for _, object := range output.Contents {
			// verify necessary fields for processing are present
			if object.Key == nil || object.Size == nil {
				a.logger.Debug("Not adding object to stream - missing fields needed for processing")
				continue
			}
			// add object; check & send batch if at batch limit
			batch = append(batch, &ObjectInfo{
				Name: *object.Key,
				Size: *object.Size,
			})
			if len(batch) == a.batchSize {
				objectChan <- &ObjectResults{
					Objects: batch,
				}
				batch = []*ObjectInfo{}
			}
		}

		// send remaining batch if present
		if len(batch) != 0 {
			objectChan <- &ObjectResults{
				Objects: batch,
			}
		}
	}
	a.logger.Info("No more pages to pull from S3, stopping stream")
	close(doneChan)
}

// DownloadObject downloads the contents of the object.
func (a *DefaultClient) DownloadObject(ctx context.Context, bucket, key string, buf []byte) (int64, error) {
	downloader := manager.NewDownloader(&a.client)

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	buffer := manager.NewWriteAtBuffer(buf)
	n, err := downloader.Download(ctx, buffer, input)
	if err != nil {
		return 0, fmt.Errorf("download: %w", err)
	}

	return n, err
}

// DeleteObject deletes the object with the given key in the specified bucket
func (a *DefaultClient) DeleteObject(ctx context.Context, bucket string, key string) error {
	_, err := a.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3 delete object: %w", err)
	}
	return nil
}
