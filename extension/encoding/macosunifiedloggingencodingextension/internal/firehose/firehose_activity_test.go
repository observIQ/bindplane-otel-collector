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

package firehose_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"
	firehose "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/firehose"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/testutil"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/uuidtext"
	"github.com/stretchr/testify/require"
)

func TestParseFirehoseActivity(t *testing.T) {
	data := []byte{
		178, 251, 0, 0, 0, 0, 0, 128, 236, 0, 0, 0, 0, 0, 0, 0, 178, 251, 0, 0, 0, 0, 0, 128,
		179, 251, 0, 0, 0, 0, 0, 128, 64, 63, 24, 18, 1, 0, 2, 0,
	}

	flags := uint16(573)
	logType := uint8(0x1)
	results, _, err := firehose.ParseFirehoseActivity(data, flags, logType)
	require.NoError(t, err)

	require.Equal(t, results.ActivityID, uint32(64434))
	require.Equal(t, results.Sentinel, uint32(2147483648))
	require.Equal(t, results.PID, uint64(236))
	require.Equal(t, results.ActivityID2, uint32(64434))
	require.Equal(t, results.Sentinel2, uint32(2147483648))
	require.Equal(t, results.ActivityID3, uint32(64435))
	require.Equal(t, results.Sentinel3, uint32(2147483648))
	require.Equal(t, results.MessageStringRef, uint32(0))
	require.False(t, results.FirehoseFormatters.MainExe)
	require.False(t, results.FirehoseFormatters.Absolute)
	require.False(t, results.FirehoseFormatters.SharedCache)
	require.Equal(t, results.FirehoseFormatters.MainExeAltIndex, uint16(0))
	require.Equal(t, results.FirehoseFormatters.UUIDRelative, "")
	require.Equal(t, results.PCID, uint32(303578944))
	require.Equal(t, results.FirehoseFormatters.HasLargeOffset, uint16(1))
	require.Equal(t, results.FirehoseFormatters.LargeSharedCache, uint16(2))
}

func TestParseFirehoseActivity_BigSur(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)
	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "system_logs_big_sur.logarchive", "Persist", "0000000000000004.tracev3")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := macosunifiedloggingencodingextension.ParseUnifiedLog(data)
	require.NoError(t, err)

	cacheProvider := uuidtext.NewCacheProvider()
	catalogChunk := results.CatalogData[0].CatalogData
	activityType := uint8(0x2)

	for _, catalogData := range results.CatalogData {
		for _, preamble := range catalogData.FirehoseData {
			for _, entry := range preamble.PublicData {
				if entry.ActivityType == activityType {
					messageData, err := firehose.GetFirehoseActivityStrings(
						entry.FirehoseActivity,
						cacheProvider,
						uint64(entry.FormatStringLocation),
						preamble.FirstProcID,
						preamble.SecondProcID,
						&catalogChunk,
					)

					require.NoError(t, err)
					require.Equal(t, messageData.FormatString, "Internal: Check the state of a node")
					require.Equal(t, messageData.Library, "/usr/libexec/opendirectoryd")
					require.Equal(t, messageData.Process, "/usr/libexec/opendirectoryd")
					require.Equal(t, messageData.ProcessUUID, "B736DF1625F538248E9527A8CEC4991E")
					require.Equal(t, messageData.LibraryUUID, "B736DF1625F538248E9527A8CEC4991E")
					return
				}
			}
		}
	}
}
