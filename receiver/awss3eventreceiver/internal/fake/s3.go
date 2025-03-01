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

// Package fake provides fake implementations of AWS clients for testing
package fake

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bps3"
)

// NewS3Client creates a new fake S3 client
func NewS3Client(_ aws.Config) bps3.Client {
	return &S3Client{
		objects: make(map[string]map[string][]byte),
	}
}

// S3Client is a fake S3 client
type S3Client struct {
	objects  map[string]map[string][]byte
	getError error
	mu       sync.Mutex
}

// GetObject gets an object from the fake S3 client
func (f *S3Client) GetObject(_ context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.getError != nil {
		return nil, f.getError
	}

	if params.Bucket == nil || params.Key == nil {
		return nil, errors.New("bucket or key is nil")
	}

	bucket := *params.Bucket
	key := *params.Key

	if _, ok := f.objects[bucket]; !ok {
		return nil, errors.New("bucket not found")
	}

	if _, ok := f.objects[bucket][key]; !ok {
		return nil, errors.New("key not found")
	}

	data := f.objects[bucket][key]
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	body := io.NopCloser(strings.NewReader(string(dataCopy)))

	return &s3.GetObjectOutput{
		Body: body,
	}, nil
}
