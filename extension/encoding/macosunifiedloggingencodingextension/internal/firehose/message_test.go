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
	"go.opentelemetry.io/collector/component"
)

type mockHostForTest struct {
	extensions map[component.ID]component.Component
}

func (h *mockHostForTest) GetExtensions() map[component.ID]component.Component { return h.extensions }

func TestExtractSharedStrings_NonActivity(t *testing.T) {
	archiveName := "0000000000000002"
	cacheProvider, results := testSetup(t, archiveName)

	catalogChunk := results.CatalogData[0].CatalogData

	testOffset := uint64(1331408102)
	testFirstProcID := uint64(45)
	testSecondProcID := uint32(188)
	originalOffset := uint64(0)

	messageData, err := firehose.ExtractSharedStrings(
		cacheProvider,
		testOffset,
		testFirstProcID,
		testSecondProcID,
		&catalogChunk,
		originalOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "/System/Library/PrivateFrameworks/AppleLOM.framework/Versions/A/AppleLOM", messageData.Library)
	require.Equal(t, "D8E5AF1CAF4F3CEB8731E6F240E8EA7D", messageData.LibraryUUID)
	require.Equal(t, "/usr/libexec/lightsoutmanagementd", messageData.Process)
	require.Equal(t, "6C3ADF991F033C1C96C4ADFAA12D8CED", messageData.ProcessUUID)
	require.Equal(t, "%@ start", messageData.FormatString)
}

func TestExtractSharedStrings_NonActivity_BadOffset(t *testing.T) {
	archiveName := "0000000000000002"
	cacheProvider, results := testSetup(t, archiveName)
	catalogChunk := results.CatalogData[0].CatalogData

	badOffset := uint64(7)
	testFirstProcID := uint64(45)
	testSecondProcID := uint32(188)
	originalOffset := uint64(0)

	messageData, err := firehose.ExtractSharedStrings(
		cacheProvider,
		badOffset,
		testFirstProcID,
		testSecondProcID,
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
	archiveName := "0000000000000002"
	cacheProvider, results := testSetup(t, archiveName)
	catalogChunk := results.CatalogData[2].CatalogData

	testOffset := uint64(2420246585)
	testFirstProcID := uint64(32)
	testSecondProcID := uint32(424)

	messageData, err := firehose.ExtractSharedStrings(
		cacheProvider,
		testOffset,
		testFirstProcID,
		testSecondProcID,
		&catalogChunk,
		testOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "/usr/lib/system/libsystem_blocks.dylib", messageData.Library)
	require.Equal(t, "4DF6D8F5D9C23A968DE45E99D6B73DC8", messageData.LibraryUUID)

	require.Equal(t, "/Library/Apple/System/Library/CoreServices/MRT.app/Contents/MacOS/MRT", messageData.Process)
	require.Equal(t, "95A48BD740423BEFBA6E0818A2EED8BE", messageData.ProcessUUID)
	require.Equal(t, "%s", messageData.FormatString)
}

func TestExtractFormatStrings_NonActivity(t *testing.T) {
	archiveName := "0000000000000002"
	cacheProvider, results := testSetup(t, archiveName)
	catalogChunk := results.CatalogData[0].CatalogData

	testOffset := uint64(14960)
	testFirstProcID := uint64(45)
	testSecondProcID := uint32(188)

	messageData, err := firehose.ExtractFormatStrings(
		cacheProvider,
		testOffset,
		testFirstProcID,
		testSecondProcID,
		&catalogChunk,
		testOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "/usr/libexec/lightsoutmanagementd", messageData.Process)
	require.Equal(t, "6C3ADF991F033C1C96C4ADFAA12D8CED", messageData.ProcessUUID)

	require.Equal(t, "/usr/libexec/lightsoutmanagementd", messageData.Library)
	require.Equal(t, "6C3ADF991F033C1C96C4ADFAA12D8CED", messageData.LibraryUUID)

	require.Equal(t, "LOMD Start", messageData.FormatString)
}

func TestExtractFormatStrings_NonActivity_BadOffset(t *testing.T) {
	archiveName := "0000000000000002"
	cacheProvider, results := testSetup(t, archiveName)

	catalogChunk := results.CatalogData[0].CatalogData
	badOffset := uint64(1)
	testFirstProcID := uint64(45)
	testSecondProcID := uint32(188)
	originalOffset := uint64(0)

	messageData, err := firehose.ExtractFormatStrings(
		cacheProvider,
		badOffset,
		testFirstProcID,
		testSecondProcID,
		&catalogChunk,
		originalOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "/usr/libexec/lightsoutmanagementd", messageData.Process)
	require.Equal(t, "Error: Invalid offset 1 for UUID 6C3ADF991F033C1C96C4ADFAA12D8CED", messageData.FormatString)
}

func TestExtractFormatStrings_NonActivity_Dynamic(t *testing.T) {
	archiveName := "0000000000000002"
	cacheProvider, results := testSetup(t, archiveName)
	catalogChunk := results.CatalogData[4].CatalogData

	testOffset := uint64(2147519968)
	testFirstProcID := uint64(38)
	testSecondProcID := uint32(317)

	messageData, err := firehose.ExtractFormatStrings(
		cacheProvider,
		testOffset,
		testFirstProcID,
		testSecondProcID,
		&catalogChunk,
		testOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "/System/Library/PrivateFrameworks/SystemAdministration.framework/Versions/A/Resources/UpdateSettingsTool", messageData.Process)
	require.Equal(t, "6F2A273A77993A719F649607CADC090B", messageData.ProcessUUID)

	require.Equal(t, "/System/Library/PrivateFrameworks/SystemAdministration.framework/Versions/A/Resources/UpdateSettingsTool", messageData.Library)
	require.Equal(t, "6F2A273A77993A719F649607CADC090B", messageData.LibraryUUID)

	require.Equal(t, "%s", messageData.FormatString)
}

func TestExtractFormatStrings_NonActivity_DynamicBadOffset(t *testing.T) {
	archiveName := "0000000000000002"
	cacheProvider, results := testSetup(t, archiveName)

	catalogChunk := results.CatalogData[4].CatalogData
	badOffset := uint64(55)
	testFirstProcID := uint64(38)
	testSecondProcID := uint32(317)

	messageData, err := firehose.ExtractFormatStrings(
		cacheProvider,
		badOffset,
		testFirstProcID,
		testSecondProcID,
		&catalogChunk,
		badOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "/System/Library/PrivateFrameworks/SystemAdministration.framework/Versions/A/Resources/UpdateSettingsTool", messageData.Process)
	require.Equal(t, "Error: Invalid offset 55 for UUID 6F2A273A77993A719F649607CADC090B", messageData.FormatString)
}

func TestExtractAbsoluteStrings_NonActivity(t *testing.T) {
	archiveName := "0000000000000002"
	cacheProvider, results := testSetup(t, archiveName)

	catalogChunk := results.CatalogData[0].CatalogData
	testOffset := uint64(396912)
	testAbsoluteOffset := uint64(280925241119206)
	testFirstProcID := uint64(0)
	testSecondProcID := uint32(0)
	originalOffset := uint64(0)

	messageData, err := firehose.ExtractAbsoluteStrings(
		cacheProvider,
		testAbsoluteOffset,
		testOffset,
		testFirstProcID,
		testSecondProcID,
		&catalogChunk,
		originalOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "/System/Library/Extensions/AppleACPIPlatform.kext/Contents/MacOS/AppleACPIPlatform", messageData.Library)
	require.Equal(t, "%s", messageData.FormatString)
}

func TestExtractAbsoluteStrings_NonActivity_BadOffset(t *testing.T) {
	archiveName := "0000000000000002"
	cacheProvider, results := testSetup(t, archiveName)

	catalogChunk := results.CatalogData[0].CatalogData
	testOffset := uint64(396912)
	testBadAbsoluteOffset := uint64(12)
	testFirstProcID := uint64(0)
	testSecondProcID := uint32(0)
	originalOffset := uint64(0)

	messageData, err := firehose.ExtractAbsoluteStrings(
		cacheProvider,
		testBadAbsoluteOffset,
		testOffset,
		testFirstProcID,
		testSecondProcID,
		&catalogChunk,
		originalOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "", messageData.Library)
	require.Equal(t, "Failed to get string message from absolute UUIDText file: ", messageData.FormatString)
}

func TestExtractAbsoluteStrings_NonActivity_Dynamic(t *testing.T) {
	archiveName := "0000000000000002"
	cacheProvider, results := testSetup(t, archiveName)

	require.Equal(t, 56, len(results.CatalogData))

	catalogChunk := results.CatalogData[1].CatalogData
	testOffset := uint64(102)
	testAbsoluteOffset := uint64(102)
	testFirstProcID := uint64(0)
	testSecondProcID := uint32(0)

	messageData, err := firehose.ExtractAbsoluteStrings(
		cacheProvider,
		testAbsoluteOffset,
		testOffset,
		testFirstProcID,
		testSecondProcID,
		&catalogChunk,
		testOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "/System/Library/DriverExtensions/com.apple.AppleUserHIDDrivers.dext/", messageData.Library)
	require.Equal(t, "%s", messageData.FormatString)
}

func TestExtractAbsoluteStrings_NonActivity_DynamicBadOffset(t *testing.T) {
	archiveName := "0000000000000002"
	cacheProvider, results := testSetup(t, archiveName)

	require.Equal(t, 56, len(results.CatalogData))

	catalogChunk := results.CatalogData[1].CatalogData
	testBadOffset := uint64(111)
	testAbsoluteOffset := uint64(102)
	testFirstProcID := uint64(0)
	testSecondProcID := uint32(0)

	messageData, err := firehose.ExtractAbsoluteStrings(
		cacheProvider,
		testAbsoluteOffset,
		testBadOffset,
		testFirstProcID,
		testSecondProcID,
		&catalogChunk,
		testBadOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "/System/Library/DriverExtensions/com.apple.AppleUserHIDDrivers.dext/", messageData.Library)
	require.Equal(t, "Error: Invalid offset 111 for absolute UUID 0AB77111A2723F2697571948ECE9BDB5", messageData.FormatString)
}

func TestExtractAltUUIDStrings(t *testing.T) {
	archiveName := "0000000000000005"
	cacheProvider, results := testSetup(t, archiveName)

	catalogChunk := results.CatalogData[0].CatalogData

	testOffset := uint64(221408)
	testFirstProcID := uint64(105)
	testSecondProcID := uint32(240)
	testUUID := "C275D5EEBAD43A86B74F16F3E62BF57D"
	originalOffset := uint64(0)

	messageData, err := firehose.ExtractAltUUIDStrings(
		cacheProvider,
		testOffset,
		testUUID,
		testFirstProcID,
		testSecondProcID,
		&catalogChunk,
		originalOffset,
	)
	require.NoError(t, err)

	require.Equal(t, "/System/Library/OpenDirectory/Modules/SystemCache.bundle/Contents/MacOS/SystemCache", messageData.Library)
	require.Equal(t, "C275D5EEBAD43A86B74F16F3E62BF57D", messageData.LibraryUUID)

	require.Equal(t, "/usr/libexec/opendirectoryd", messageData.Process)
	require.Equal(t, "B736DF1625F538248E9527A8CEC4991E", messageData.ProcessUUID)
}

func TestGetCatalogDSC(t *testing.T) {
	archiveName := "0000000000000002"
	_, results := testSetup(t, archiveName)

	catalogChunk := results.CatalogData[0].CatalogData

	testFirstProcID := uint64(136)
	testSecondProcID := uint32(342)

	dscUUID, mainUUID := firehose.GetCatalogDSC(
		&catalogChunk,
		testFirstProcID,
		testSecondProcID,
	)

	require.Equal(t, "80896B329EB13A10A7C5449B15305DE2", dscUUID)
	require.Equal(t, "87721013944F3EA7A42C604B141CCDAA", mainUUID)
}

func TestGetUUIDImagePath(t *testing.T) {
	archiveName := "0000000000000002"
	cacheProvider, _ := testSetup(t, archiveName)

	testUUID := "B736DF1625F538248E9527A8CEC4991E"

	imagePath, err := firehose.GetUUIDImagePath(
		testUUID,
		cacheProvider,
	)
	require.NoError(t, err)

	require.Equal(t, "/usr/libexec/opendirectoryd", imagePath)
}

func testSetup(t *testing.T, archiveName string) (*uuidtext.CacheProvider, *macosunifiedloggingencodingextension.UnifiedLogData) {
	testutil.SkipIfNoReceiverTestdata(t)

	basePath := filepath.Join(testutil.ReceiverTestdataDir(), "system_logs_big_sur.logarchive")
	filePath := filepath.Join(basePath, "Persist", archiveName+".tracev3")
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	cacheProvider := testutil.CreateAndPopulateUUIDAndDSCCaches(t, basePath)

	results, err := macosunifiedloggingencodingextension.ParseUnifiedLog(data)
	require.NoError(t, err)

	return cacheProvider, results
}
