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

func TestFDREventUnmarshal(t *testing.T) {
	testCases := []struct {
		name          string
		fileName      string
		expectObjects []event.S3Object
		expectError   bool
	}{
		{
			name:     "single",
			fileName: "single.json",
			expectObjects: []event.S3Object{
				{
					Bucket: "cs-prod-cannon-8-dcba9n8oxpakjtn86o8w11bbthgagusw2b-s3alias",
					Key:    "abcd11658703038a-xyz593c5/data/61c7b298-bb3a-444f-9806-ef99a96c52a5/part-00000.gz",
					Size:   13090,
				},
			},
		},
		{
			name:     "multiple",
			fileName: "multiple.json",
			expectObjects: []event.S3Object{
				{
					Bucket: "cs-prod-cannon-8-dcba9n8oxpakjtn86o8w11bbthgagusw2b-s3alias",
					Key:    "abcd11658703038a-xyz593c5/data/61c7b298-bb3a-444f-9806-ef99a96c52a5/part-00000.gz",
					Size:   13090,
				},
				{
					Bucket: "cs-prod-cannon-8-dcba9n8oxpakjtn86o8w11bbthgagusw2b-s3alias",
					Key:    "abcd11658703038a-xyz593c5/data/61c7b298-bb3a-444f-9806-ef99a96c52a5/part-00001.gz",
					Size:   13092,
				},
			},
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
			filePath := filepath.Join("testdata", "fdr", testCase.fileName)
			body, err := os.ReadFile(filePath)
			require.NoError(t, err)

			set := componenttest.NewNopTelemetrySettings()
			unmarshaler := event.NewFDRUnmarshaler(set)

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

func TestFDRMarshaler(t *testing.T) {
	set := componenttest.NewNopTelemetrySettings()
	marshaler := event.NewFDRMarshaler(set)

	objects := []event.S3Object{
		{
			EventType: "ObjectCreated:Put",
			Bucket:    "cs-prod-cannon-8-dcba9n8oxpakjtn86o8w11bbthgagusw2b-s3alias",
			Key:       "abcd11658703038a-xyz593c5/data/61c7b298-bb3a-444f-9806-ef99a96c52a5/part-00000.gz",
			Size:      13090,
		},
		{
			EventType: "ObjectCreated:Put",
			Bucket:    "cs-prod-cannon-8-dcba9n8oxpakjtn86o8w11bbthgagusw2b-s3alias",
			Key:       "abcd11658703038a-xyz593c5/data/61c7b298-bb3a-444f-9806-ef99a96c52a5/part-00001.gz",
			Size:      13092,
		},
	}

	body, err := marshaler.Marshal(objects)
	require.NoError(t, err)
	require.NotNil(t, body)
}
