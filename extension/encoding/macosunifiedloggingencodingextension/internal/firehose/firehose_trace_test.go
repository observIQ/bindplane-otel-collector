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
package firehose_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"
	firehose "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/firehose"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestParseFirehoseTrace(t *testing.T) {
	testData := []byte{106, 139, 3, 0, 0}
	firehoseTrace, _, err := firehose.ParseFirehoseTrace(testData)
	require.NoError(t, err)
	require.Equal(t, uint32(232298), firehoseTrace.UnknownPCID)

	testData = []byte{248, 145, 3, 0, 200, 0, 0, 0, 0, 0, 0, 0, 8, 1}
	firehoseTrace, _, err = firehose.ParseFirehoseTrace(testData)
	require.NoError(t, err)
	require.Equal(t, uint32(233976), firehoseTrace.UnknownPCID)
	require.Equal(t, 1, len(firehoseTrace.MessageData.ItemInfo))
}

func TestGetMessage(t *testing.T) {
	testData := []byte{200, 0, 0, 0, 0, 0, 0, 0, 8, 1}
	slices.Reverse(testData)
	_, message, err := firehose.GetMessage(testData)
	require.NoError(t, err)
	require.Equal(t, "200", message.ItemInfo[0].MessageStrings)
}

func TestGetMessageMultiple(t *testing.T) {
	testData := []byte{2, 8, 8, 0, 0, 0, 0, 0, 0, 0, 200, 0, 0, 127, 251, 75, 225, 96, 176}
	_, message, err := firehose.GetMessage(testData)
	require.NoError(t, err)
	require.Equal(t, "140717286580400", message.ItemInfo[0].MessageStrings)
	require.Equal(t, "200", message.ItemInfo[1].MessageStrings)
}

func TestGetFirehoseTraceStrings(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)

	basePath := filepath.Join(testutil.ReceiverTestdataDir(), "system_logs_high_sierra.logarchive")
	filePath := filepath.Join(basePath, "logdata.LiveData.tracev3")
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := macosunifiedloggingencodingextension.ParseUnifiedLog(data)
	require.NoError(t, err)

	cacheProvider := testutil.CreateAndPopulateUUIDAndDSCCaches(t, basePath)
	activityType := uint8(0x3)

	for _, catalogData := range results.CatalogData {
		for _, preamble := range catalogData.FirehoseData {
			for _, entry := range preamble.PublicData {
				if entry.ActivityType == activityType {
					messageData, err := firehose.GetFirehoseTraceStrings(
						cacheProvider,
						uint64(entry.FormatStringLocation),
						preamble.FirstProcID,
						preamble.SecondProcID,
						&catalogData.CatalogData,
					)

					require.NoError(t, err)

					require.Equal(t, messageData.FormatString, "starting metadata download")
					require.Equal(t, messageData.Library, "/usr/libexec/mobileassetd")
					require.Equal(t, messageData.Process, "/usr/libexec/mobileassetd")
					require.Equal(t, messageData.ProcessUUID, "CC6C867B44D63D0ABAA7598659629484")
					require.Equal(t, messageData.LibraryUUID, "CC6C867B44D63D0ABAA7598659629484")

					return
				}
			}
		}
	}
}
