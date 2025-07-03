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

package worker_test

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/worker"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStartsWithJSONObjectOrArray(t *testing.T) {
	tests := []struct {
		filePath   string
		expectTrue bool
	}{
		{filePath: "testdata/logs_array_in_records.json", expectTrue: true},
		{filePath: "testdata/logs_array.json", expectTrue: true},
		{filePath: "testdata/cloudtrail.json", expectTrue: true},
		{filePath: "testdata/logs_array_fragment.txt", expectTrue: true},
		{filePath: "testdata/text_logs.txt", expectTrue: false},
		{filePath: "testdata/json_lines.txt", expectTrue: true},
		{filePath: "testdata/logs_numbers.txt", expectTrue: false},
	}

	for _, test := range tests {
		t.Run(test.filePath, func(t *testing.T) {
			file, err := os.Open(test.filePath)
			require.NoError(t, err, "open log file")
			defer file.Close()

			bufferedReader := worker.NewBufferedReader(file, 1024)
			startsWithJSONObjectOrArray, err := worker.StartsWithJSONObjectOrArray(bufferedReader)
			require.NoError(t, err, "check if starts with json object or array")
			require.Equal(t, test.expectTrue, startsWithJSONObjectOrArray)
		})
	}
}

func TestParseJSONLogs(t *testing.T) {
	tests := []struct {
		filePath      string
		startOffset   int64
		expectLogs    int
		expectError   error
		expectOffsets []int64
	}{
		{filePath: "testdata/logs_array_in_records.json", expectLogs: 4, expectOffsets: []int64{2506, 4497, 6554, 8256}, startOffset: 0},
		{filePath: "testdata/logs_array_in_records.json", expectLogs: 3, expectOffsets: []int64{4497, 6554, 8256}, startOffset: 2506},
		{filePath: "testdata/logs_array_in_records.json", expectLogs: 2, expectOffsets: []int64{6554, 8256}, startOffset: 4497},
		{filePath: "testdata/logs_array_in_records.json", expectLogs: 1, expectOffsets: []int64{8256}, startOffset: 6554},
		{filePath: "testdata/logs_array_in_records.json", expectLogs: 1, expectOffsets: []int64{8256}, startOffset: 6000}, // in between records
		{filePath: "testdata/logs_array_in_records.json", expectLogs: 0, expectOffsets: []int64{}, startOffset: 8256},
		{filePath: "testdata/logs_array_in_records.json", expectLogs: 0, expectOffsets: []int64{}, startOffset: 8257}, // after the end of the array
		// offsets for gzipped files are the same as the uncompressed files
		{filePath: "testdata/logs_array_in_records.json.gz", expectLogs: 4, expectOffsets: []int64{2506, 4497, 6554, 8256}, startOffset: 0},
		{filePath: "testdata/logs_array_in_records.json.gz", expectLogs: 3, expectOffsets: []int64{4497, 6554, 8256}, startOffset: 2506},
		{filePath: "testdata/logs_array_in_records.json.gz", expectLogs: 2, expectOffsets: []int64{6554, 8256}, startOffset: 4497},
		{filePath: "testdata/logs_array_in_records.json.gz", expectLogs: 1, expectOffsets: []int64{8256}, startOffset: 6554},
		{filePath: "testdata/logs_array_in_records.json.gz", expectLogs: 1, expectOffsets: []int64{8256}, startOffset: 6000}, // in between records
		{filePath: "testdata/logs_array_in_records.json.gz", expectLogs: 0, expectOffsets: []int64{}, startOffset: 8256},
		{filePath: "testdata/logs_array_in_records.json.gz", expectLogs: 0, expectOffsets: []int64{}, startOffset: 8257}, // after the end of the array
		{filePath: "testdata/logs_array_in_records_after_limit.json", expectError: worker.ErrNotArrayOrKnownObject},
		{filePath: "testdata/logs_array.json", expectLogs: 4, expectOffsets: []int64{2018, 3899, 5842, 7452}, startOffset: 0},
		{filePath: "testdata/logs_array.json", expectLogs: 1, expectOffsets: []int64{7452}, startOffset: 5842},
		{filePath: "testdata/logs_array.json", expectLogs: 1, expectOffsets: []int64{7452}, startOffset: 5842},
		{filePath: "testdata/cloudtrail.json", expectLogs: 4},
		{filePath: "testdata/logs_array_fragment.txt", expectLogs: 1},
		{filePath: "testdata/json_lines.txt", expectError: worker.ErrNotArrayOrKnownObject},
		{filePath: "testdata/logs_numbers.txt", expectError: worker.ErrNotArrayOrKnownObject},
	}

	for _, test := range tests {
		t.Run(test.filePath, func(t *testing.T) {
			file, err := os.Open(test.filePath)
			require.NoError(t, err, "open log file")
			defer file.Close()

			stream := worker.LogStream{
				Name:        test.filePath,
				ContentType: aws.String("application/json"),
				MaxLogSize:  1024,
				Body:        file,
				Logger:      zap.NewNop(),
			}

			bufferedReader, err := stream.BufferedReader(context.Background())
			require.NoError(t, err, "get buffered reader")

			parser := worker.NewJSONParser(bufferedReader)
			logs, err := parser.Parse(context.Background(), test.startOffset)
			if test.expectError != nil {
				require.ErrorIs(t, err, test.expectError)
				return
			}
			require.NoError(t, err, "parse logs")

			offsets := []int64{}
			count := 0
			for log, err := range logs {
				if err == nil {
					t.Logf("Log: %v", log)
					count++
					offsets = append(offsets, parser.Offset())
				}
			}

			require.Equal(t, test.expectLogs, count)
			if len(test.expectOffsets) > 0 {
				require.Equal(t, test.expectOffsets, offsets[:len(test.expectOffsets)])
			}
		})
	}
}
