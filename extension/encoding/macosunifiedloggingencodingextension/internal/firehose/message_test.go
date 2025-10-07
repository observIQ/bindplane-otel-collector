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
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
)

type mockHostForTest struct {
	extensions map[component.ID]component.Component
}

func (h *mockHostForTest) GetExtensions() map[component.ID]component.Component { return h.extensions }

func TestExtractSharedStrings_NonActivity(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)

	cacheProvider := testutil.CreateAndPopulateUUIDAndDSCCaches(t)

	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "system_logs_big_sur.logarchive", "Persist", "0000000000000002.tracev3")
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := macosunifiedloggingencodingextension.ParseUnifiedLog(data)
	require.NoError(t, err)

	catalogChunk := results.CatalogData[0].CatalogData

	testOffset := uint64(1331408102)
	testFirstProcId := uint64(45)
	testSecondProcId := uint32(188)
	originalOffset := uint64(0)

	messageData, err := firehose.ExtractSharedStrings(
		cacheProvider,
		testOffset,
		testFirstProcId,
		testSecondProcId,
		&catalogChunk,
		originalOffset,
	)
	require.NoError(t, err)

	//6C3ADF991F033C1C96C4ADFAA12D8CED

	require.Equal(t, "/System/Library/PrivateFrameworks/AppleLOM.framework/Versions/A/AppleLOM", messageData.Library)
	require.Equal(t, "D8E5AF1CAF4F3CEB8731E6F240E8EA7D", messageData.LibraryUUID)
	require.Equal(t, "/usr/libexec/lightsoutmanagementd", messageData.Process)
	require.Equal(t, "6C3ADF991F033C1C96C4ADFAA12D8CED", messageData.ProcessUUID)
	require.Equal(t, "%@ start", messageData.FormatString)
}

func TestExtractSharedStrings_NonActivity_BadOffset(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)

	cacheProvider := testutil.CreateAndPopulateUUIDAndDSCCaches(t)

	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "system_logs_big_sur.logarchive", "Persist", "0000000000000002.tracev3")
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := macosunifiedloggingencodingextension.ParseUnifiedLog(data)
	require.NoError(t, err)

	catalogChunk := results.CatalogData[0].CatalogData

	badOffset := uint64(7)
	testFirstProcId := uint64(45)
	testSecondProcId := uint32(188)
	originalOffset := uint64(0)

	messageData, err := firehose.ExtractSharedStrings(
		cacheProvider,
		badOffset,
		testFirstProcId,
		testSecondProcId,
		&catalogChunk,
		originalOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "/usr/lib/system/libsystem_blocks.dylib", messageData.Library)
	require.Equal(t, "4DF6D8F5D9C23A968DE45E99D6B73DC8", messageData.LibraryUUID)
	require.Equal(t, "/usr/libexec/lightsoutmanagementd", messageData.Process)
	require.Equal(t, "6C3ADF991F033C1C96C4ADFAA12D8CED", messageData.ProcessUUID)
	require.Equal(t, "Error: Invalid shared string offset", messageData.FormatString)
}

func TestExtractSharedStrings_NonActivity_Dynamic(t *testing.T) {
	testutil.SkipIfNoReceiverTestdata(t)

	cacheProvider := testutil.CreateAndPopulateUUIDAndDSCCaches(t)

	filePath := filepath.Join(testutil.ReceiverTestdataDir(), "system_logs_big_sur.logarchive", "Persist", "0000000000000002.tracev3")
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	results, err := macosunifiedloggingencodingextension.ParseUnifiedLog(data)
	require.NoError(t, err)

	catalogChunk := results.CatalogData[0].CatalogData

	badOffset := uint64(7)
	testFirstProcId := uint64(45)
	testSecondProcId := uint32(188)
	originalOffset := uint64(0)

	messageData, err := firehose.ExtractSharedStrings(
		cacheProvider,
		badOffset,
		testFirstProcId,
		testSecondProcId,
		&catalogChunk,
		originalOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "/usr/lib/system/libsystem_blocks.dylib", messageData.Library)
	require.Equal(t, "4DF6D8F5D9C23A968DE45E99D6B73DC8", messageData.LibraryUUID)
	require.Equal(t, "/usr/libexec/lightsoutmanagementd", messageData.Process)
	require.Equal(t, "6C3ADF991F033C1C96C4ADFAA12D8CED", messageData.ProcessUUID)
	require.Equal(t, "Error: Invalid shared string offset", messageData.FormatString)
}
