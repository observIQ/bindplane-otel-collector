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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsGlobPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"linux/auditd", false},
		{"linux/*", true},
		{"linux/audit?", true},
		{"linux/[abc]", true},
		{"", false},
		{"plain/path/here", false},
		{"*/foo", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			require.Equal(t, tc.expected, isGlobPattern(tc.input))
		})
	}
}

func TestSplitGlobPrefix(t *testing.T) {
	tests := []struct {
		pattern        string
		expectedPrefix string
		expectedGlob   string
	}{
		{"linux/*", "linux/", "linux/*"},
		{"linux/*/subdir", "linux/", "linux/*/subdir"},
		{"*", "", "*"},
		{"a/b/c*", "a/b/", "a/b/c*"},
		{"plain/path", "plain/path", "plain/path"},
		{"[abc]/foo", "", "[abc]/foo"},
	}

	for _, tc := range tests {
		t.Run(tc.pattern, func(t *testing.T) {
			prefix, glob := splitGlobPrefix(tc.pattern)
			require.Equal(t, tc.expectedPrefix, prefix)
			require.Equal(t, tc.expectedGlob, glob)
		})
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern  string
		name     string
		expected bool
	}{
		// Basic wildcard matching
		{"linux/*", "linux/auditd", true},
		{"linux/*", "linux/auditd/", true}, // trailing slash normalized
		{"linux/*", "linux/logb", true},

		// * doesn't match /
		{"linux/*", "linux/auditd/sub", false},

		// Question mark
		{"linux/log?", "linux/logb", true},
		{"linux/log?", "linux/logbc", false},

		// Character class
		{"linux/[al]*", "linux/auditd", true},
		{"linux/[al]*", "linux/logb", true},
		{"linux/[al]*", "linux/zeta", false},

		// No glob chars
		{"linux/auditd", "linux/auditd", true},
		{"linux/auditd", "linux/other", false},
	}

	for _, tc := range tests {
		t.Run(tc.pattern+"_"+tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, matchGlob(tc.pattern, tc.name))
		})
	}
}
