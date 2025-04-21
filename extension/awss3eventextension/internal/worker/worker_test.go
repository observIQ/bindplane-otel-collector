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

// Package worker provides a worker that processes S3 event notifications.
package worker_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"

	"github.com/observiq/bindplane-otel-collector/extension/awss3eventextension/internal/worker"
	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
	"github.com/observiq/bindplane-otel-collector/internal/aws/event"
	"github.com/observiq/bindplane-otel-collector/internal/aws/fake"
)

func TestProcessMessage(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	testCases := []struct {
		name string
		// Test cases are defined as object sets to represent subsequent uploads to s3.
		// The test will aggregate the object sets into a single structure and verify the
		// downloaded files correspond to the aggregated structure.
		objectSets []map[string]map[string]string
	}{
		{
			name:       "no object created events",
			objectSets: []map[string]map[string]string{},
		},
		{
			name: "single object",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "myvalue1",
					},
				},
			},
		},
		{
			name: "single bucket multiple objects",
			objectSets: []map[string]map[string]string{
				{
					"mybucket": {
						"mykey1": "myvalue1",
						"mykey2": "myvalue2",
					},
				},
			},
		},
		{
			name: "multiple buckets single object",
			objectSets: []map[string]map[string]string{
				{
					"mybucket1": {
						"mykey1": "myvalue1",
					},
					"mybucket2": {
						"mykey1": "myvalue1",
					},
				},
			},
		},
		{
			name: "multiple buckets multiple objects",
			objectSets: []map[string]map[string]string{
				{
					"mybucket1": {
						"mykey1": "myvalue1",
						"mykey2": "myvalue2",
					},
					"mybucket2": {
						"mykey1": "myvalue1",
						"mykey2": "myvalue2",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			fakeAWS := client.NewClient(aws.Config{}).(*fake.AWS)

			// overall structure of all object sets
			expectedDownloads := map[string]map[string]string{}
			for _, objectSet := range testCase.objectSets {
				for bucket, objects := range objectSet {
					expectedDownloads[bucket] = objects
				}
				fakeAWS.CreateObjects(t, objectSet)
			}

			set := componenttest.NewNopTelemetrySettings()
			unmarshaler := event.NewS3Unmarshaler(set)
			dir := t.TempDir()
			w := worker.New(set, aws.Config{}, unmarshaler, dir)

			numCallbacks := 0

			for {
				msg, err := fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
				if err != nil {
					require.ErrorIs(t, err, fake.ErrEmptyQueue)
					break
				}
				for _, msg := range msg.Messages {
					w.ProcessMessage(ctx, msg, "myqueue", func() {
						numCallbacks++
					})
				}
			}

			require.Equal(t, len(testCase.objectSets), numCallbacks)

			// Find directories in dir
			files, err := os.ReadDir(dir)
			require.NoError(t, err)
			require.Equal(t, len(expectedDownloads), len(files))

			// Verify each directory contains the expected objects
			for bucket, objects := range expectedDownloads {
				subdir := filepath.Join(dir, bucket)
				files, err := os.ReadDir(subdir)
				require.NoError(t, err)
				require.Equal(t, len(objects), len(files))

				for _, file := range files {
					require.Contains(t, objects, file.Name())
					content, err := os.ReadFile(filepath.Join(subdir, file.Name()))
					require.NoError(t, err)
					require.Equal(t, objects[file.Name()], string(content))
				}
			}

			_, err = fakeAWS.SQS().ReceiveMessage(ctx, new(sqs.ReceiveMessageInput))
			require.Equal(t, fake.ErrEmptyQueue, err)
		})
	}
}
