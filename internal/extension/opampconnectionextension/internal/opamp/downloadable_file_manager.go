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

// Package opamp contains configurations and protocol implementations to handle OpAmp communication.
package opamp

import (
	"github.com/open-telemetry/opamp-go/protobufs"
)

// DownloadableFileManager handles DownloadableFile's from a PackagesAvailable message
//
//go:generate mockery --name DownloadableFileManager --filename mock_downloadable_file_manager.go --structname MockDownloadableFileManager
type DownloadableFileManager interface {
	// FetchAndExtractArchive fetches the archive at the specified URL.
	// It then checks to see if it matches the expected sha256 sum of the file.
	// If it matches, the archive is extracted.
	// If the archive cannot be extracted, downloaded, or verified, then an error is returned.
	FetchAndExtractArchive(*protobufs.DownloadableFile) error
	// CleanupArtifacts removes temporary artifacts from previous download/installs
	CleanupArtifacts()
}
