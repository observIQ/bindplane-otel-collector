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

// Package client provides a client for AWS services.
package client // import "github.com/observiq/bindplane-otel-collector/internal/aws/client"

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client is the interface for the S3 client
//
//go:generate mockery --name S3Client --output ./mocks --with-expecter --filename mock_s3_client.go --structname MockS3Client
type S3Client interface {
	// GetObject retrieves an object from S3
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}
