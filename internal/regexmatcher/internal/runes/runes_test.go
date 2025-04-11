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

package runes_test

import (
	"fmt"
	"regexp/syntax"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	runes "github.com/observiq/bindplane-otel-collector/internal/regexmatcher/internal/runes"
)

// TestFilterSet tests the two-phase filtering approach.
func TestFilterSet(t *testing.T) {
	testCases := []struct {
		name               string
		regexes            []string          // Input regex patterns for the FilterSet
		samples            map[string][]bool // map[inputString]expectedShouldAttemptMatchForEachRegex
		expectNilEssential bool              // Whether FilterSet.essentialRunes should be nil
	}{
		{
			name:    "single_no_required",
			regexes: []string{`^[a-z]+$`},
			samples: map[string][]bool{
				"abc":    {true}, // No mandatory runes, should always attempt
				"123":    {true}, // Should attempt
				"ABC":    {true}, // Should attempt
				"123ABC": {true}, // Should attempt
			},
			expectNilEssential: true, // No pattern has mandatory runes
		},
		{
			name:    "single_one_required",
			regexes: []string{`^a[a-z]+$`}, // Requires 'a'
			samples: map[string][]bool{
				"abc": {true},  // Contains 'a', should attempt
				"123": {false}, // Does not contain 'a', should NOT attempt
			},
		},
		{
			name: "multiple_mixed",
			regexes: []string{
				`^start-end$`, // Requires 's', 't', 'a', 'r', 't', '-', 'e', 'n', 'd'
				`^a+$`,        // Requires 'a'
				`^(cat|dog)$`, // Requires nothing
				`^你好$`,        // Requires '你', '好'
			},
			samples: map[string][]bool{
				"start-end": {true, true, true, false},
				"a":         {false, true, true, false},
				"cat":       {false, true, true, false},
				"dog":       {false, false, true, false},
				"你好":        {false, false, true, true},
				"startend":  {false, true, true, false}, // Missing '-'
				"other":     {false, false, true, false},
				"你好start":   {false, true, true, true},
				"a-你好":      {false, true, true, true},
				"":          {false, false, true, false}, // Always attempt pattern 2 (cat|dog)
			},
		},
		{
			name:    "all_no_required",
			regexes: []string{`^[a-z]+$`, `^\d+$`, `^(cat|dog)$`},
			samples: map[string][]bool{
				"abc": {true, true, true},
				"123": {true, true, true},
				"cat": {true, true, true},
				"":    {true, true, true},
			},
			expectNilEssential: true, // No pattern has mandatory runes
		},
		{
			// Test case where essential runes exist globally, but a specific input
			// string doesn't contain *any* of them. In this case, only patterns
			// that required no runes should be attempted.
			name:    "global_essential_but_none_in_input",
			regexes: []string{`a`, `b`, `c|d`}, // Essential: 'a', 'b'. Pattern 2 needs none.
			samples: map[string][]bool{
				"a":   {true, false, true},
				"b":   {false, true, true},
				"c":   {false, false, true},
				"xyz": {false, false, true}, // Does not contain 'a' or 'b', should only attempt pattern 2
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			trees := make([]*syntax.Regexp, len(tc.regexes))
			for i, reStr := range tc.regexes {
				re, err := syntax.Parse(reStr, syntax.Perl)
				require.NoError(t, err, "Failed to parse regex: %s", reStr)
				trees[i] = re.Simplify()
			}

			filterSet := runes.NewFilterSet(trees)
			require.NotNil(t, filterSet)

			// Assert essentialRunes nil status - requires helper or reflection
			// if tc.expectNilEssential { ... }

			// 3. Test each sample string
			for sample, expectedResults := range tc.samples {
				t.Run(fmt.Sprintf("sample_%s", sample), func(t *testing.T) {
					require.Len(t, expectedResults, len(tc.regexes), "Mismatch between expected results and number of regexes")

					// 4. Perform PreFilter (Phase 1)
					preFilterResult := filterSet.PreFilter(sample)

					// 5. Check ShouldAttemptMatch for each pattern (Phase 2)
					for i, expected := range expectedResults {
						actual := preFilterResult.ShouldAttemptMatch(i)
						assert.Equal(t, expected, actual,
							"Sample '%s', Pattern %d ('%s'): Expected ShouldAttemptMatch=%v, got %v",
							sample, i, tc.regexes[i], expected, actual)
					}
				})
			}
		})
	}
}
