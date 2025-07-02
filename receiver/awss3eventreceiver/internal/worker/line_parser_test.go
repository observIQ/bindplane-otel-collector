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

	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/worker"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestParseTextLogs(t *testing.T) {

	tests := []struct {
		filePath      string
		expectLogs    int
		startOffset   int64
		expectOffsets []int64
	}{
		{filePath: "testdata/text_logs.txt", expectLogs: 3, startOffset: 0, expectOffsets: []int64{6, 12, 41}},
		{filePath: "testdata/logs_numbers.txt", expectLogs: 9, startOffset: 0, expectOffsets: []int64{2, 4, 6, 8, 10, 12, 14, 16, 18}},
		{filePath: "testdata/logs_numbers.txt", expectLogs: 8, startOffset: 2, expectOffsets: []int64{4, 6, 8, 10, 12, 14, 16, 18}},
		{filePath: "testdata/logs_numbers.txt", expectLogs: 2, startOffset: 14, expectOffsets: []int64{16, 18}},
		{filePath: "testdata/logs_numbers.txt", expectLogs: 1, startOffset: 16, expectOffsets: []int64{18}},
		{filePath: "testdata/logs_numbers.txt", expectLogs: 0, startOffset: 18, expectOffsets: []int64{}},
		// offsets for gzipped files are the same as the uncompressed files
		{filePath: "testdata/logs_numbers.txt.gz", expectLogs: 9, startOffset: 0, expectOffsets: []int64{2, 4, 6, 8, 10, 12, 14, 16, 18}},
		{filePath: "testdata/logs_numbers.txt.gz", expectLogs: 8, startOffset: 2, expectOffsets: []int64{4, 6, 8, 10, 12, 14, 16, 18}},
		{filePath: "testdata/logs_numbers.txt.gz", expectLogs: 2, startOffset: 14, expectOffsets: []int64{16, 18}},
		{filePath: "testdata/logs_numbers.txt.gz", expectLogs: 1, startOffset: 16, expectOffsets: []int64{18}},
		{filePath: "testdata/logs_numbers.txt.gz", expectLogs: 0, startOffset: 18, expectOffsets: []int64{}},
	}

	for _, test := range tests {
		t.Run(test.filePath, func(t *testing.T) {
			file, err := os.Open(test.filePath)
			require.NoError(t, err, "open log file")
			defer file.Close()

			stream := worker.LogStream{
				Name:       test.filePath,
				MaxLogSize: 1024,
				Body:       file,
				Logger:     zap.NewNop(),
			}

			bufferedReader, err := stream.BufferedReader(context.Background())
			require.NoError(t, err, "get buffered reader")

			parser := worker.NewLineParser(bufferedReader)
			logs, err := parser.Parse(context.Background(), test.startOffset)
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
			require.Equal(t, test.expectOffsets, offsets[:len(test.expectOffsets)])
		})
	}
}
