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

// Package matcher provides a matcher for regexes that respects order.
package matcher

import (
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"

	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor/internal/matcher/runes"
)

// NamedRegex is the input structure for named regular expressions.
type NamedRegex struct {
	Name  string
	Regex *regexp.Regexp
}

// patternInfo holds the original regex and name.
type patternInfo struct {
	name  string
	regex *regexp.Regexp
}

// Matcher holds patterns and the encapsulated rune FilterSet.
type Matcher struct {
	patterns      []patternInfo
	runeFilterSet *runes.FilterSet
	defaultValue  string
}

// New creates a new Matcher.
func New(regexes []NamedRegex, defaultValue string) (*Matcher, error) {
	if len(regexes) == 0 {
		return nil, errors.New("no regexes provided")
	}

	names := map[string]bool{}
	simplifiedTrees := make([]*syntax.Regexp, len(regexes))
	patternInfos := make([]patternInfo, len(regexes))

	for i, r := range regexes {
		if r.Name == "" {
			return nil, errors.New("regex name cannot be empty")
		}
		if names[r.Name] {
			return nil, fmt.Errorf("regex name is duplicated: %s", r.Name)
		}
		if r.Regex == nil {
			return nil, fmt.Errorf("regex '%s' is nil", r.Name)
		}
		names[r.Name] = true

		// Parse and simplify for rune analysis
		parsedRegex, err := syntax.Parse(r.Regex.String(), syntax.Perl)
		if err != nil {
			// This shouldn't happen if input is *regexp.Regexp, but check anyway
			return nil, fmt.Errorf("internal error: failed to re-parse regex string for '%s': %w", r.Name, err)
		}
		simplifiedTrees[i] = parsedRegex.Simplify()

		// Store basic pattern info
		patternInfos[i] = patternInfo{name: r.Name, regex: r.Regex}
	}

	// Create the encapsulated rune filter set by passing the syntax trees
	filterSet := runes.NewFilterSet(simplifiedTrees)

	matcher := &Matcher{
		patterns:      patternInfos,
		runeFilterSet: filterSet,
		defaultValue:  defaultValue,
	}

	return matcher, nil
}

// Match finds the name of the first regex that matches the string s, using the FilterSet.
func (m *Matcher) Match(s string) string {
	// Perform Phase 1 filtering via the FilterSet
	preFilterResult := m.runeFilterSet.PreFilter(s)

	// Iterate through patterns and perform Phase 2 check + full match
	for i, p := range m.patterns {
		// Check if this pattern passes the rune filter (Phase 2)
		if !preFilterResult.ShouldAttemptMatch(i) {
			continue // Skip expensive regex match
		}

		// If rune filter passed, perform the full regex match
		if p.regex.MatchString(s) {
			return p.name
		}
	}
	return m.defaultValue
}
