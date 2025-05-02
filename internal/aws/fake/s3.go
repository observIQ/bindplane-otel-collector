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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
)

var _ client.S3Client = &s3Client{}

var fakeS3 = struct {
	mu      sync.Mutex
	objects map[string]map[string]string
}{
	objects: make(map[string]map[string]string),
}

// NewS3Client creates a new fake S3 client
func NewS3Client(_ *testing.T) client.S3Client {
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

	fullData := []byte(object) // Use original full data for total size
	dataToReturn := make([]byte, len(fullData))
	copy(dataToReturn, fullData)

	if params.Range == nil {
		return &s3.GetObjectOutput{
			Body:          io.NopCloser(strings.NewReader(string(dataToReturn))),
			ContentLength: aws.Int64(int64(len(dataToReturn))),
		}, nil
	}

	start, end := parseRangeHeader(fullData, *params.Range)
	dataToReturn = fullData[start : end+1]
	contentRange := fmt.Sprintf("bytes %d-%d/%d", start, end, len(fullData))

	return &s3.GetObjectOutput{
		Body:          io.NopCloser(bytes.NewReader(dataToReturn)),
		ContentLength: aws.Int64(int64(len(dataToReturn))),
		ContentRange:  aws.String(contentRange),
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

func parseRangeHeader(data []byte, rangeStr string) (int, int) { // Renamed from trimRange
	// Range header format: "bytes=start-end"
	parts := strings.Split(strings.TrimPrefix(rangeStr, "bytes="), "-")
	if len(parts) != 2 {
		return 0, len(data) - 1
	}

	// Get default start and end values
	start := 0
	end := len(data) - 1

	// Parse start position
	if parts[0] != "" {
		if parsedStart, err := strconv.Atoi(parts[0]); err == nil && parsedStart >= 0 && parsedStart < len(data) {
			start = parsedStart
		}
	}

	// Parse end position (if provided)
	if parts[1] != "" {
		if parsedEnd, err := strconv.Atoi(parts[1]); err == nil && parsedEnd >= start && parsedEnd < len(data) {
			end = parsedEnd
		}
	}

	// Handle range validation
	if start > end || start >= len(data) {
		return 0, len(data) - 1
	}

	// The end index is inclusive in range headers (+1 for slice upper bound)
	return start, end
}
