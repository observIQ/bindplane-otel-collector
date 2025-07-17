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

func TestStartsWithAvroOcfMagic(t *testing.T) {
	tests := []struct {
		filePath   string
		expectTrue bool
	}{
		{filePath: "testdata/logs_array_in_records.json", expectTrue: false},
		{filePath: "testdata/sample_logs.avro", expectTrue: true},
	}

	for _, test := range tests {
		t.Run(test.filePath, func(t *testing.T) {
			file, err := os.Open(test.filePath)
			require.NoError(t, err, "open log file")
			defer file.Close()

			bufferedReader := worker.NewBufferedReader(file, 1024)
			startsWithAvroOcfMagic, err := worker.StartsWithAvroOcfMagic(bufferedReader)
			require.NoError(t, err, "check if starts with avro ocf magic")
			require.Equal(t, test.expectTrue, startsWithAvroOcfMagic)
		})
	}
}

func TestParseAvroOcfLogs(t *testing.T) {
	tests := []struct {
		filePath      string
		startOffset   int64
		expectLogs    int
		expectError   error
		expectOffsets []int64
	}{
		{filePath: "testdata/sample_logs.avro", expectLogs: 10, expectOffsets: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{filePath: "testdata/sample_logs.avro", expectLogs: 9, expectOffsets: []int64{2, 3, 4, 5, 6, 7, 8, 9, 10}, startOffset: 1},
		{filePath: "testdata/sample_logs.avro", expectLogs: 3, expectOffsets: []int64{8, 9, 10}, startOffset: 7},
		{filePath: "testdata/sample_logs.avro", expectLogs: 0, expectOffsets: []int64{}, startOffset: 10},
		{filePath: "testdata/sample_logs.avro.gz", expectLogs: 1000},
		{filePath: "testdata/sample_logs.avro.gz", expectLogs: 900, startOffset: 100},
		{filePath: "testdata/sample_logs.avro.gz", expectLogs: 1, startOffset: 999},
		{filePath: "testdata/sample_logs.avro.gz", expectLogs: 0, startOffset: 1000},
	}

	for _, test := range tests {
		t.Run(test.filePath, func(t *testing.T) {
			file, err := os.Open(test.filePath)
			require.NoError(t, err, "open log file")
			defer file.Close()

			stream := worker.LogStream{
				Name:        test.filePath,
				ContentType: aws.String("application/avro"),
				MaxLogSize:  1024,
				Body:        file,
				Logger:      zap.NewNop(),
			}

			bufferedReader, err := stream.BufferedReader(context.Background())
			require.NoError(t, err, "get buffered reader")

			parser := worker.NewAvroOcfParser(bufferedReader, zap.NewNop())
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
