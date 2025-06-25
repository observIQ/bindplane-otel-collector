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

func TestParseTextLogs(t *testing.T) {

	tests := []struct {
		filePath   string
		expectLogs int
	}{
		{filePath: "testdata/text_logs.txt", expectLogs: 3},
	}

	for _, test := range tests {
		t.Run(test.filePath, func(t *testing.T) {
			file, err := os.Open(test.filePath)
			require.NoError(t, err, "open log file")
			defer file.Close()

			bufferedReader := bufio.NewReader(file)

			parser := worker.NewLineParser(bufferedReader)
			logs, err := parser.Parse(context.Background())
			require.NoError(t, err, "parse logs")

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
