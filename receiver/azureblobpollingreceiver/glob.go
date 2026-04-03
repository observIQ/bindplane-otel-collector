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

package azureblobpollingreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/azureblobpollingreceiver"

import (
	"path"
	"strings"
)

// isGlobPattern returns true if the string contains glob metacharacters.
func isGlobPattern(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// splitGlobPrefix splits a glob pattern into the static prefix (everything before the
// first glob character, trimmed to the last '/') and the full glob pattern.
// For example, "linux/*/foo" returns ("linux/", "linux/*/foo").
func splitGlobPrefix(pattern string) (staticPrefix, globPattern string) {
	// Find the first glob metacharacter
	idx := strings.IndexAny(pattern, "*?[")
	if idx == -1 {
		return pattern, pattern
	}

	prefix := pattern[:idx]

	// Trim to the last '/' so we get a complete directory prefix
	if lastSlash := strings.LastIndex(prefix, "/"); lastSlash >= 0 {
		prefix = prefix[:lastSlash+1]
	} else {
		prefix = ""
	}

	return prefix, pattern
}

// matchGlob matches a name against a glob pattern, normalizing trailing slashes.
// Azure returns directory prefixes with trailing slashes (e.g., "linux/auditd/"),
// so both the pattern and name have trailing slashes stripped before matching.
func matchGlob(pattern, name string) bool {
	pattern = strings.TrimSuffix(pattern, "/")
	name = strings.TrimSuffix(name, "/")

	matched, err := path.Match(pattern, name)
	if err != nil {
		return false
	}
	return matched
}
