// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pcapreceiver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

// TestGolden tests that packet parsing produces the expected log output
// Input files are tcpdump text format, expected files are plog YAML format
func TestGolden(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{name: "tcp_https"},
		{name: "udp_dns"},
		{name: "icmp_echo"},
		{name: "ipv6_tcp"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Interface:       "eth0",
				ParseAttributes: true,
			}
			sink := &consumertest.LogsSink{}
			receiver := newTestReceiver(t, cfg, nil, sink)

			// Read input packet data
			inputPath := filepath.Join("testdata", "golden", "input", tc.name+".txt")
			content, err := os.ReadFile(inputPath)
			require.NoError(t, err)

			// Parse and process the packet
			lines := strings.Split(string(content), "\n")
			ctx := context.Background()
			receiver.processPacket(ctx, lines)

			// Verify log was created
			require.Equal(t, 1, sink.LogRecordCount(), "Expected exactly 1 log record")

			// Read expected output
			expectedPath := filepath.Join("testdata", "golden", "expected", tc.name+".txt.yaml")
			expectedLogs, err := golden.ReadLogs(expectedPath)
			require.NoError(t, err)

			// Compare logs (ignoring timestamps which are dynamic)
			err = plogtest.CompareLogs(expectedLogs, sink.AllLogs()[0],
				plogtest.IgnoreTimestamp(),
				plogtest.IgnoreObservedTimestamp(),
			)
			require.NoError(t, err)
		})
	}
}

// TestGolden_NoAttributes tests packet parsing with ParseAttributes disabled
func TestGolden_NoAttributes(t *testing.T) {
	cfg := &Config{
		Interface:       "eth0",
		ParseAttributes: false,
	}
	sink := &consumertest.LogsSink{}
	receiver := newTestReceiver(t, cfg, nil, sink)

	// Read TCP packet input
	inputPath := filepath.Join("testdata", "golden", "input", "tcp_https.txt")
	content, err := os.ReadFile(inputPath)
	require.NoError(t, err)

	// Parse and process the packet
	lines := strings.Split(string(content), "\n")
	ctx := context.Background()
	receiver.processPacket(ctx, lines)

	// Verify log was created
	require.Equal(t, 1, sink.LogRecordCount(), "Expected exactly 1 log record")

	// Verify body is set but no attributes
	logs := sink.AllLogs()[0]
	logRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	// Body should start with 0x
	body := logRecord.Body().AsString()
	require.True(t, strings.HasPrefix(body, "0x"), "Body should start with 0x prefix")

	// No attributes should be set when ParseAttributes is false
	require.Equal(t, 0, logRecord.Attributes().Len(), "No attributes should be set when ParseAttributes is false")
}
