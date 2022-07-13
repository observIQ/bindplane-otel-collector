// Copyright  observIQ, Inc.
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

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/observiq/observiq-otel-collector/updater/internal/download"
	"github.com/observiq/observiq-otel-collector/updater/internal/install"
	"github.com/observiq/observiq-otel-collector/updater/internal/version"
	"github.com/spf13/pflag"
)

// Unimplemented
func main() {
	var showVersion = pflag.BoolP("version", "v", false, "Prints the version of the collector and exits, if specified.")
	var downloadURL = pflag.String("url", "", "URL to download the update archive from.")
	var tmpDir = pflag.String("tmpdir", "", "Temporary directory for artifacts. Parent of the 'rollback' directory.")
	var contentHash = pflag.String("content-hash", "", "Hex encoded hash of the content at the specified URL.")
	pflag.Parse()

	if *showVersion {
		fmt.Println("observiq-otel-collector updater version", version.Version())
		fmt.Println("commit:", version.GitHash())
		fmt.Println("built at:", version.Date())
		return
	}

	if *downloadURL == "" {
		log.Println("The --url flag must be specified!")
		pflag.PrintDefaults()
		os.Exit(1)
	}

	if *tmpDir == "" {
		log.Println("The --tmpdir flag must be specified!")
		pflag.PrintDefaults()
		os.Exit(1)
	}

	if *contentHash == "" {
		log.Println("The --content-hash flag must be specified!")
		pflag.PrintDefaults()
		os.Exit(1)
	}

	if err := download.FetchAndExtractArchive(*downloadURL, *tmpDir, *contentHash); err != nil {
		log.Fatalf("Failed to download and verify update: %s", err)
	}

	installer, err := install.NewInstaller(*tmpDir)
	if err != nil {
		log.Fatalf("Failed to create installer: %s", err)
	}

	if err := installer.Install(); err != nil {
		log.Fatalf("Failed to install: %s", err)
	}
}
