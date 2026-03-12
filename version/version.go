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

// Package version exposes data on the collector set at compile time
package version

// these will be replaced at link time by make.
var (
	version = "latest"  // Semantic version, or "latest" by default
	gitHash = "unknown" // Commit hash from which this build was generated
	date    = "unknown" // Date the build was generated
)

// Version returns the version of the collector.
func Version() string {
	return version
}

// GitHash returns the githash associated with the collector's version.
func GitHash() string {
	return gitHash
}

// Date returns the publish date associated with the collector's version.
func Date() string {
	return date
}
