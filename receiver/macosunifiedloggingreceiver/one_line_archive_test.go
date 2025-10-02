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

package macosunifiedloggingreceiver

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

func TestParseOneLineArchive(t *testing.T) {
	filePath := filepath.Join("testdata", "one-line.logarchive", "**", "*.tracev3")
	sink := new(consumertest.LogsSink)

	setupAndStartReceiver(t, filePath, sink, 1)

	// Verify the log content
	logCounts := countLogInformation(sink.AllLogs())

	require.Equal(t, 5, sink.LogRecordCount())
	require.Equal(t, 0, logCounts["byEventType"]["signpostEvent"])
}
