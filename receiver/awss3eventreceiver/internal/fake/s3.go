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
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpaws"
)

var _ bpaws.S3Client = &s3Client{}

var fakeS3 = struct {
	mu      sync.Mutex
	objects map[string]map[string]string
}{
	objects: make(map[string]map[string]string),
}

// NewS3Client creates a new fake S3 client
func NewS3Client(_ *testing.T) bpaws.S3Client {
	return &s3Client{}
}

// s3Client is a fake S3 client
type s3Client struct{}

// GetObject gets an object from the fake S3 client
func (f *s3Client) GetObject(_ context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if params.Bucket == nil || params.Key == nil {
		return nil, errors.New("bucket or key is nil")
	}

	fakeS3.mu.Lock()
	defer fakeS3.mu.Unlock()

	bucket, ok := fakeS3.objects[*params.Bucket]
	if !ok {
		return nil, errors.New("bucket not found")
	}

	object, ok := bucket[*params.Key]
	if !ok {
		return nil, errors.New("key not found")
	}

	dataCopy := make([]byte, len(object))
	copy(dataCopy, object)

	return &s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader(string(dataCopy))),
	}, nil
}

func (f *s3Client) putObject(bucket string, key string, body string) {
	fakeS3.mu.Lock()
	defer fakeS3.mu.Unlock()
	if _, ok := fakeS3.objects[bucket]; !ok {
		fakeS3.objects[bucket] = make(map[string]string)
	}
	fakeS3.objects[bucket][key] = body
}
