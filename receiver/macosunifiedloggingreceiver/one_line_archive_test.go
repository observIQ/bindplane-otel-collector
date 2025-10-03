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
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

func TestParseOneLineArchive(t *testing.T) {
	filePaths := getFilePathsForArchiveInTestData("one-line.logarchive")
	sink := new(consumertest.LogsSink)

	setupAndStartReceiver(t, filePaths, sink, 1)

	require.Equal(t, 1, sink.LogRecordCount())
	oneLogRecord := sink.AllLogs()[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	require.Equal(t, "ULOG_TEST_FACD0438-4576-4FDB-B698-B70396B2371D", oneLogRecord.Body().AsString())

	pid, _ := oneLogRecord.Attributes().Get("pid")
	require.Equal(t, "20390", pid.AsString())

	euid, _ := oneLogRecord.Attributes().Get("euid")
	require.Equal(t, "501", euid.AsString())

	library, _ := oneLogRecord.Attributes().Get("library")
	require.Equal(t, "/usr/bin/logger", library.AsString())

	process, _ := oneLogRecord.Attributes().Get("process")
	require.Equal(t, "/usr/bin/logger", process.AsString())

	processUUID, _ := oneLogRecord.Attributes().Get("process_image_uuid")
	require.Equal(t, "D0690BA80F6C374E83D96B1F64163F65", processUUID.AsString())

	libraryUUID, _ := oneLogRecord.Attributes().Get("library_uuid")
	require.Equal(t, "D0690BA80F6C374E83D96B1F64163F65", libraryUUID.AsString())

	bootUUID, _ := oneLogRecord.Attributes().Get("boot_uuid")
	require.Equal(t, "181E2449363C4CE38C3C000B9E475965", bootUUID.AsString())

	eventType, _ := oneLogRecord.Attributes().Get("event_type")
	require.Equal(t, "Log", eventType.AsString())

	timestamp, _ := oneLogRecord.Attributes().Get("timestamp")
	require.Equal(t, "2025-09-05T10:16:01.990209792Z", timestamp.AsString())

}
