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
package aws //import "github.com/observiq/bindplane-agent/receiver/awss3rehydrationreceiver/internal/aws"

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ObjectInfo contains necessary info to process S3 objects
type ObjectInfo struct {
	Name string
	Size int64
}

// S3Client provides a client for S3 object operations
//
//go:generate mockery --name S3Client --output ./mocks --with-expecter --filename mock_s3_client.go --structname MockS3Client
type S3Client interface {
	// ListObjects returns a list of ObjectInfo objects present in the bucket with the given prefix
	ListObjects(ctx context.Context, bucket string, prefix, continuationToken *string) ([]*ObjectInfo, *string, error)

	// DownloadObject downloads the contents of the object into the buffer.
	DownloadObject(ctx context.Context, bucket, key string, buf []byte) (int64, error)

	// DeleteObjects deletes the keys in the specified bucket
	DeleteObjects(ctx context.Context, bucket string, keys []string) error
}

// DefaultClient is an implementation of the S3Client for AWS
type DefaultClient struct {
	sessionCfg aws.Config
	roleArn    string
}

// NewAWSClient creates a new AWS Client
func NewAWSClient(region, roleArn string) (S3Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	return &DefaultClient{
		sessionCfg: cfg,
		roleArn:    roleArn,
	}, nil
}

// ListObjects list all the objects in the bucket with the given prefix. Uses the continuationToken from the last request for paging.
func (a *DefaultClient) ListObjects(ctx context.Context, bucket string, prefix, continuationToken *string) ([]*ObjectInfo, *string, error) {
	svc := s3.NewFromConfig(a.sessionCfg)

	input := &s3.ListObjectsV2Input{
		Bucket:            aws.String(bucket),
		ContinuationToken: continuationToken,
		Prefix:            prefix,
	}

	output, err := svc.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, nil, fmt.Errorf("list objects: %w", err)
	}

	nextToken := output.ContinuationToken

	objects := make([]*ObjectInfo, len(output.Contents))

	for i, object := range output.Contents {
		// All fields are pointers so check all pointers needed to process are present
		if object.Key == nil || object.Size == nil {
			continue
		}

		objects[i] = &ObjectInfo{
			Name: *object.Key,
			Size: *object.Size,
		}
	}

	return objects, nextToken, nil
}

// DownloadObject downloads the contents of the object.
func (a *DefaultClient) DownloadObject(ctx context.Context, bucket, key string, buf []byte) (int64, error) {
	client := s3.NewFromConfig(a.sessionCfg)

	downloader := manager.NewDownloader(client)

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

// DeleteObjects deletes the keys in the specified bucket
func (a *DefaultClient) DeleteObjects(ctx context.Context, bucket string, keys []string) error {
	client := s3.NewFromConfig(a.sessionCfg)

	objects := make([]types.ObjectIdentifier, len(keys))
	for i, key := range keys {
		objects[i] = types.ObjectIdentifier{
			Key: aws.String(key),
		}
	}

	params := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &types.Delete{
			Objects: objects,
		},
	}

	_, err := client.DeleteObjects(ctx, params)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}
