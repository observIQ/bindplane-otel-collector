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

package matcher_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor/internal/benchtest"
	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor/internal/matcher"
	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor/internal/named"
)

func TestMatch(t *testing.T) {
	regexes := []named.Regex{
		{Name: "five_digits", Regex: regexp.MustCompile(`\d{5}`)},
		{Name: "four_letters", Regex: regexp.MustCompile(`[a-zA-Z]{4}`)},
	}
	matcher, err := matcher.New(regexes, "default")
	require.NoError(t, err)

	// Input "no match" contains "matc", which matches four_letters.
	assert.Equal(t, "four_letters", matcher.Match("no match"))

	assert.Equal(t, "five_digits", matcher.Match("12345"))

	assert.Equal(t, "four_letters", matcher.Match("asdf"))

	// Should match the first regex (five_digits) because it appears first
	assert.Equal(t, "five_digits", matcher.Match("12345asdf"))

	// Should match the first regex (five_digits) because it appears first, even though "asdf" also matches four_letters
	assert.Equal(t, "five_digits", matcher.Match("asdf12345"))
}

func TestDefaultValue(t *testing.T) {
	regexes := []named.Regex{
		{Name: "five_digits", Regex: regexp.MustCompile(`\d{5}`)},
		{Name: "four_letters", Regex: regexp.MustCompile(`[a-zA-Z]{4}`)},
	}

	// Test with default "custom_default"
	customDefaultMatcher, err := matcher.New(regexes, "custom_default")
	require.NoError(t, err)
	assert.Equal(t, "custom_default", customDefaultMatcher.Match("no-ma-str-123"))

	// Test with empty string default
	emptyDefaultMatcher, err := matcher.New(regexes, "")
	require.NoError(t, err)
	assert.Equal(t, "", emptyDefaultMatcher.Match("no-ma-str-123"))
}

func TestMatchComplex(t *testing.T) {
	regexes := make([]named.Regex, 0, len(benchtest.Regexes))
	for _, r := range benchtest.Regexes {
		regexes = append(regexes, named.Regex{
			Name:  r.Name,
			Regex: r.Regex,
		})
	}

	matcher, err := matcher.New(regexes, "default")
	require.NoError(t, err)

	for k, v := range benchtest.Examples {
		assert.Equal(t, v, matcher.Match(k))
	}
}

func TestNoRegexesError(t *testing.T) {
	regexes := []named.Regex{}

	matcher, err := matcher.New(regexes, "default")
	require.Error(t, err)
	require.Nil(t, matcher)
}

func TestEmptyNameError(t *testing.T) {
	regexes := []named.Regex{
		{Name: "", Regex: regexp.MustCompile(`\d{5}`)},
	}

	matcher, err := matcher.New(regexes, "default")
	require.Error(t, err)
	require.Nil(t, matcher)
}

func TestDuplicateNameError(t *testing.T) {
	regexes := []named.Regex{
		{Name: "test", Regex: regexp.MustCompile(`\d{5}`)},
		{Name: "test", Regex: regexp.MustCompile(`[a-zA-Z]{4}`)},
	}

	matcher, err := matcher.New(regexes, "default")
	require.Error(t, err)
	require.Nil(t, matcher)
}

func TestNilRegexError(t *testing.T) {
	regexes := []named.Regex{
		{Name: "test", Regex: nil},
	}

	matcher, err := matcher.New(regexes, "default")
	require.Error(t, err)
	require.Nil(t, matcher)
}
