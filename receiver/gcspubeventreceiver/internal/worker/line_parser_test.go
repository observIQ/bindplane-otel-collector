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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/observiq/bindplane-otel-collector/receiver/gcspubeventreceiver/internal/worker"
)

// newLineParserFromString is a test helper that creates a NewBufferedReader from a string
// and wraps it in a NewLineParser.
func newLineParserFromString(s string, maxLogSize int) worker.LogParser {
	r := strings.NewReader(s)
	br := worker.NewBufferedReader(r, maxLogSize)
	return worker.NewLineParser(br)
}

func TestLineParser_Parse(t *testing.T) {
	t.Parallel()

	const maxLogSize = 4096

	testCases := []struct {
		name        string
		input       string
		startOffset int64
		wantCount   int
		wantErr     bool
	}{
		{
			name:        "single line",
			input:       "hello world",
			startOffset: 0,
			wantCount:   1,
		},
		{
			name:        "multiple lines",
			input:       "line1\nline2\nline3",
			startOffset: 0,
			wantCount:   3,
		},
		{
			name:        "empty lines are skipped",
			input:       "line1\n\nline3",
			startOffset: 0,
			wantCount:   2,
		},
		{
			name:        "trailing newline",
			input:       "line1\nline2\n",
			startOffset: 0,
			wantCount:   2,
		},
		{
			name:        "non-zero start offset skips bytes",
			input:       "ab\ncd\nef\n",
			startOffset: 3, // skip "ab\n", start at "cd"
			wantCount:   2,
		},
		{
			// When startOffset is past end of stream, io.CopyN returns EOF and
			// Parse propagates it as an error.
			name:        "start offset past end returns error",
			input:       "ab\ncd\n",
			startOffset: 1000,
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			parser := newLineParserFromString(tc.input, maxLogSize)

			logs, err := parser.Parse(context.Background(), tc.startOffset)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			count := 0
			for _, parseErr := range logs {
				require.NoError(t, parseErr)
				count++
			}
			require.Equal(t, tc.wantCount, count)
		})
	}
}

func TestLineParser_Offset(t *testing.T) {
	t.Parallel()

	const maxLogSize = 4096
	input := "ab\ncd\nef\n"

	parser := newLineParserFromString(input, maxLogSize)
	logs, err := parser.Parse(context.Background(), 0)
	require.NoError(t, err)

	expectedOffsets := []int64{3, 6, 9}
	i := 0
	for log, parseErr := range logs {
		require.NoError(t, parseErr)
		_ = log
		require.Equal(t, expectedOffsets[i], parser.Offset())
		i++
	}
	require.Equal(t, len(expectedOffsets), i, "expected exactly %d log records", len(expectedOffsets))
}

func TestLineParser_AppendLogBody_String(t *testing.T) {
	t.Parallel()

	const maxLogSize = 4096
	parser := newLineParserFromString("hello", maxLogSize)

	lr := plog.NewLogRecord()
	err := parser.AppendLogBody(context.Background(), lr, "hello")
	require.NoError(t, err)
	require.Equal(t, "hello", lr.Body().Str())
}

func TestLineParser_AppendLogBody_InvalidType(t *testing.T) {
	t.Parallel()

	const maxLogSize = 4096
	parser := newLineParserFromString("", maxLogSize)

	lr := plog.NewLogRecord()
	err := parser.AppendLogBody(context.Background(), lr, 42)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected string record")
}
