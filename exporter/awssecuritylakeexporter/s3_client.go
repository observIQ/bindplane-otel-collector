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

package awssecuritylakeexporter

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"go.uber.org/zap"
)

// S3Client uploads objects to Amazon S3.
type S3Client interface {
	Upload(ctx context.Context, bucket, key string, body io.Reader) error
}

type s3ClientImpl struct {
	client *transfermanager.Client
	logger *zap.Logger
}

// NewS3Client creates a new S3Client.
//
// If roleARN is non-empty, STS AssumeRole credentials are used.
// If endpoint is non-empty, a custom endpoint resolver is set on the S3 client.
func NewS3Client(ctx context.Context, region, roleARN, endpoint string, logger *zap.Logger) (S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	if roleARN != "" {
		stsClient := sts.NewFromConfig(cfg)
		cfg.Credentials = aws.NewCredentialsCache(
			stscreds.NewAssumeRoleProvider(stsClient, roleARN),
		)
	}

	var s3Opts []func(*s3.Options)
	if endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	s3Svc := s3.NewFromConfig(cfg, s3Opts...)
	tm := transfermanager.New(s3Svc)

	return &s3ClientImpl{
		client: tm,
		logger: logger,
	}, nil
}

// Upload uploads body to the given bucket/key. It logs success or failure.
func (c *s3ClientImpl) Upload(ctx context.Context, bucket, key string, body io.Reader) error {
	_, err := c.client.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	})
	if err != nil {
		c.logger.Error("failed to upload object to S3", zap.String("key", key), zap.Error(err))
		return fmt.Errorf("s3 upload %q: %w", key, err)
	}
	return nil
}

// BuildS3Key constructs the S3 object key for a Security Lake parquet file.
// The key format is:
//
//	ext/{sourceName}/region={region}/accountId={accountID}/eventDay={YYYYMMDD}/{sourceName}_{timestamp}_{fileID}.parquet
//
// eventTime is converted to UTC to derive the eventDay partition value.
// The timestamp embedded in the filename is Unix epoch seconds.
func BuildS3Key(sourceName, region, accountID string, eventTime time.Time, fileID string) string {
	utc := eventTime.UTC()
	eventDay := utc.Format("20060102")
	timestamp := utc.Unix()
	return fmt.Sprintf(
		"ext/%s/region=%s/accountId=%s/eventDay=%s/%s_%d_%s.parquet",
		sourceName,
		region,
		accountID,
		eventDay,
		sourceName,
		timestamp,
		fileID,
	)
}
