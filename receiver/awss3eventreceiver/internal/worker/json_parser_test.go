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
	"bufio"
	"context"
	"os"
	"testing"

	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/worker"
	"github.com/stretchr/testify/require"
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
	}

	for _, test := range tests {
		t.Run(test.filePath, func(t *testing.T) {
			file, err := os.Open(test.filePath)
			require.NoError(t, err, "open log file")
			defer file.Close()

			bufferedReader := bufio.NewReader(file)
			startsWithJSONObjectOrArray, err := worker.StartsWithJSONObjectOrArray(bufferedReader)
			require.NoError(t, err, "check if starts with json object or array")
			require.Equal(t, test.expectTrue, startsWithJSONObjectOrArray)
		})
	}
}

func TestParseJSONLogs(t *testing.T) {
	tests := []struct {
		filePath    string
		expectLogs  int
		expectError error
	}{
		{filePath: "testdata/logs_array_in_records.json", expectLogs: 4},
		{filePath: "testdata/logs_array_in_records_after_limit.json", expectError: worker.ErrNotArrayOrKnownObject},
		{filePath: "testdata/logs_array.json", expectLogs: 4},
		{filePath: "testdata/cloudtrail.json", expectLogs: 4},
		{filePath: "testdata/logs_array_fragment.txt", expectLogs: 1},
		{filePath: "testdata/json_lines.txt", expectError: worker.ErrNotArrayOrKnownObject},
	}

	for _, test := range tests {
		t.Run(test.filePath, func(t *testing.T) {
			file, err := os.Open(test.filePath)
			require.NoError(t, err, "open log file")
			defer file.Close()

			bufferedReader := bufio.NewReader(file)

			parser := worker.NewJSONParser(bufferedReader)
			logs, err := parser.Parse(context.Background())
			if test.expectError != nil {
				require.ErrorIs(t, err, test.expectError)
				return
			} else {
				require.NoError(t, err, "parse logs")
			}

			count := 0
			for log, err := range logs {
				if err == nil {
					t.Logf("Log: %v", log)
					count++
				}
			}

			require.Equal(t, test.expectLogs, count)
		})
	}
}
