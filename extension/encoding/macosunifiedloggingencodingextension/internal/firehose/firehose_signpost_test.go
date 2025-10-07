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
	"testing"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"
	firehose "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/firehose"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestParseFirehoseSignpost(t *testing.T) {
	testData := []byte{225, 244, 2, 0, 1, 0, 238, 238, 178, 178, 181, 176, 238, 238, 176, 63, 27, 0, 0, 0}
	testFlags := uint16(33282)
	signpost, _, err := firehose.ParseFirehoseSignpost(testData, testFlags)
	require.NoError(t, err)
	require.Equal(t, uint32(193761), signpost.UnknownPCID)
	require.Equal(t, uint32(0), signpost.UnknownActivityID)
	require.Equal(t, uint32(0), signpost.UnknownSentinel)
	require.Equal(t, uint16(1), signpost.Subsystem)
	require.Equal(t, uint64(17216892719917625070), signpost.SignpostID)
	require.Equal(t, uint32(1785776), signpost.SignpostName)
	require.Equal(t, uint8(0), signpost.TTLValue)
	require.Equal(t, uint32(0), signpost.DataRefValue)

	require.True(t, signpost.FirehoseFormatters.MainExe)
	require.False(t, signpost.FirehoseFormatters.SharedCache)
	require.Equal(t, uint16(0), signpost.FirehoseFormatters.HasLargeOffset)
	require.Equal(t, uint16(0), signpost.FirehoseFormatters.LargeSharedCache)
	require.False(t, signpost.FirehoseFormatters.Absolute)
	require.Equal(t, "", signpost.FirehoseFormatters.UUIDRelative)
	require.Equal(t, uint16(0), signpost.FirehoseFormatters.MainExeAltIndex)
}

func TestGetFirehoseSignpostStrings(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)

	basePath := filepath.Join(testutil.ReceiverTestdataDir(), "system_logs_big_sur.logarchive")
	filePath := filepath.Join(basePath, "Signpost", "0000000000000001.tracev3")
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := macosunifiedloggingencodingextension.ParseUnifiedLog(data)
	require.NoError(t, err)

	cacheProvider := testutil.CreateAndPopulateUUIDAndDSCCaches(t, basePath)
	catalogChunk := results.CatalogData[0].CatalogData
	activityType := uint8(0x6)

	for _, catalogData := range results.CatalogData {
		for _, preamble := range catalogData.FirehoseData {
			for _, entry := range preamble.PublicData {
				if entry.ActivityType == activityType {
					messageData, err := firehose.GetFirehoseSignpostStrings(
						entry.FirehoseSignpost,
						cacheProvider,
						uint64(entry.FormatStringLocation),
						preamble.FirstProcID,
						preamble.SecondProcID,
						&catalogChunk,
					)

					require.NoError(t, err)
					require.Equal(t, messageData.FormatString, "")
					require.Equal(t, messageData.Library, "/usr/libexec/kernelmanagerd")
					require.Equal(t, messageData.Process, "/usr/libexec/kernelmanagerd")
					require.Equal(t, messageData.ProcessUUID, "CCCF30257483376883C824222233386D")
					require.Equal(t, messageData.LibraryUUID, "CCCF30257483376883C824222233386D")

					return
				}
			}
		}
	}
}
