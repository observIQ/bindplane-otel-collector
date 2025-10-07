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

package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/sharedcache"
	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/uuidtext"
	"github.com/stretchr/testify/require"
)

// SkipIfNoTestdata skips the test if the local testdata directory doesn't exist.
func SkipIfNoTestdata(t *testing.T) {
	t.Helper()
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		t.Skip("Skipping test: testdata directory not found")
	}
}

// SkipIfNoReceiverTestdata skips the test if the receiver testdata directory doesn't exist.
func SkipIfNoReceiverTestdata(t *testing.T) {
	t.Helper()
	testdataPath := ReceiverTestdataDir()
	if _, err := os.Stat(testdataPath); os.IsNotExist(err) {
		t.Skip("Skipping test: testdata directory not found")
	}
}

// ModuleRoot returns the repository root (assumes running from a package under the repo).
// It walks up from the current working directory until it finds a go.mod containing the module path.
func ModuleRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// ReceiverTestdataDir returns the absolute path to the receiver's testdata directory.
func ReceiverTestdataDir() string {
	root, err := ModuleRoot()
	if err != nil {
		return "testdata"
	}
	return filepath.Clean(filepath.Join(root, "..", "..", "..", "receiver", "macosunifiedloggingreceiver", "testdata"))
}

// ExtensionTestdataDir returns the absolute path to this package's testdata directory.
func ExtensionTestdataDir() string {
	root, err := ModuleRoot()
	if err != nil {
		return "testdata"
	}
	return filepath.Join(root, "testdata")
}

func CreateAndPopulateUUIDAndDSCCaches(t *testing.T) *uuidtext.CacheProvider {
	cacheProvider := uuidtext.NewCacheProvider()

	uuidTextFilePath := filepath.Join(ReceiverTestdataDir(), "system_logs_big_sur.logarchive", "[0-9A-F][0-9A-F]", "*")
	uuidTextFilePaths, _ := filepath.Glob(uuidTextFilePath)

	for _, uuidPath := range uuidTextFilePaths {
		uuidTextData, _ := os.ReadFile(uuidPath)
		parsedUUIDText, err := uuidtext.ParseUUIDText(uuidTextData)
		require.NoError(t, err)
		uuid := helpers.ExtractUUIDFromPath(uuidPath)
		cacheProvider.UpdateUUID(uuid, "", parsedUUIDText)
	}

	dscFilePath := filepath.Join(ReceiverTestdataDir(), "system_logs_big_sur.logarchive", "dsc", "*")
	dscFilePaths, _ := filepath.Glob(dscFilePath)

	for _, dscPath := range dscFilePaths {
		dscData, _ := os.ReadFile(dscPath)
		parsedDSC, err := sharedcache.ParseDSC(dscData)
		require.NoError(t, err)
		cacheProvider.UpdateDSC(helpers.ExtractUUIDFromPath(dscPath), "", parsedDSC)
	}
	return cacheProvider
}
