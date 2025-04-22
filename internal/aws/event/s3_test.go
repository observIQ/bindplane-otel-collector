// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package event_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/observiq/bindplane-otel-collector/internal/aws/event"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestS3EventUnmarshal(t *testing.T) {
	testCases := []struct {
		name          string
		fileName      string
		expectObjects []event.S3Object
		expectError   bool
	}{
		{
			name:     "single",
			fileName: "created_single.json",
			expectObjects: []event.S3Object{
				{
					EventType: "ObjectCreated:Put",
					Bucket:    "s3eventreceiver-dev",
					Key:       "test2.txt",
					Size:      25,
				},
			},
		},
		{
			name:     "multiple",
			fileName: "created_multiple.json",
			expectObjects: []event.S3Object{
				{
					EventType: "ObjectCreated:Put",
					Bucket:    "s3eventreceiver-dev2",
					Key:       "test4.txt",
					Size:      28,
				},
				{
					EventType: "ObjectCreated:Put",
					Bucket:    "s3eventreceiver-dev2",
					Key:       "test3.txt",
					Size:      27,
				},
			},
		},
		{
			name:     "mixed",
			fileName: "mixed.json",
			expectObjects: []event.S3Object{
				{
					EventType: "ObjectCreated:Put",
					Bucket:    "s3eventreceiver-dev",
					Key:       "test3.txt",
					Size:      27,
				},
			},
		},
		{
			name:          "deleted",
			fileName:      "deleted.json",
			expectObjects: []event.S3Object{},
			expectError:   false,
		},
		{
			name:          "replication",
			fileName:      "replication.json",
			expectObjects: []event.S3Object{},
			expectError:   false,
		},
		{
			name:          "empty",
			fileName:      "empty.json",
			expectObjects: []event.S3Object{},
			expectError:   true,
		},
		{
			name:          "malformed",
			fileName:      "notjson.xml",
			expectObjects: []event.S3Object{},
			expectError:   true,
		},
		{
			name:          "wrong_schema",
			fileName:      "wrong_schema.json",
			expectObjects: []event.S3Object{},
			expectError:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			filePath := filepath.Join("testdata", "s3", testCase.fileName)
			body, err := os.ReadFile(filePath)
			require.NoError(t, err)

			set := componenttest.NewNopTelemetrySettings()
			unmarshaler := event.NewS3Unmarshaler(set)

			objects, err := unmarshaler.Unmarshal(body)

			if testCase.expectError {
				require.Error(t, err)
				require.Len(t, objects, 0)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, testCase.expectObjects, objects)
			}
		})
	}
}

func TestS3Marshaler(t *testing.T) {
	set := componenttest.NewNopTelemetrySettings()
	marshaler := event.NewS3Marshaler(set)

	objects := []event.S3Object{
		{
			EventType: "ObjectCreated:Put",
			Bucket:    "s3eventreceiver-dev",
			Key:       "test2.txt",
			Size:      25,
		},
		{
			EventType: "ObjectCreated:Put",
			Bucket:    "s3eventreceiver-dev",
			Key:       "test3.txt",
			Size:      27,
		},
	}

	body, err := marshaler.Marshal(objects)
	require.NoError(t, err)
	require.NotNil(t, body)
}
