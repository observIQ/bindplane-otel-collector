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

// Package runes provides utilities for identifying mandatory runes in regexes
// and prefiltering strings based on those runes using an efficient two-phase approach.
package runes

import (
	"regexp/syntax"
)

// patternMandatoryRunes holds the mandatory runes for a single pattern.
type patternMandatoryRunes struct {
	mandatory map[rune]struct{}
}

// FilterSet precomputes rune information for a set of regex patterns
// and provides efficient filtering capabilities.
type FilterSet struct {
	essentialRunes map[rune]struct{}       // UNION of all mandatory runes across all patterns
	patterns       []patternMandatoryRunes // Mandatory runes for each pattern (in order)
}

// NewFilterSet creates a new FilterSet by analyzing the provided regex syntax trees.
func NewFilterSet(trees []*syntax.Regexp) *FilterSet {
	fs := &FilterSet{
		essentialRunes: make(map[rune]struct{}),
		patterns:       make([]patternMandatoryRunes, len(trees)),
	}

	for i, tree := range trees {
		pMandatory := make(map[rune]struct{})
		// Use the existing (but now potentially internal/unexported) findAllMandatoryRunes
		findAllMandatoryRunes(tree, pMandatory, false)

		fs.patterns[i] = patternMandatoryRunes{mandatory: pMandatory}

		// Add to global essential set
		for r := range pMandatory {
			fs.essentialRunes[r] = struct{}{}
		}
	}

	// Optimization: if no runes are essential globally, nil the map
	if len(fs.essentialRunes) == 0 {
		fs.essentialRunes = nil
	}

	return fs
}

// PreFilterResult holds the result of the initial string scan (Phase 1).
type PreFilterResult struct {
	runesFoundInS map[rune]struct{}
	filterSet     *FilterSet // Reference back to the FilterSet for pattern data
}

// PreFilter performs the initial scan of the input string against essential runes.
// It returns a result object that can be used to check individual patterns.
func (fs *FilterSet) PreFilter(s string) PreFilterResult {
	result := PreFilterResult{filterSet: fs}
	if fs.essentialRunes == nil {
		// Optimization: No essential runes, no need to scan or allocate
		return result
	}

	// Phase 1: Scan string s once and find essential runes present
	runesFound := make(map[rune]struct{})
	for _, r := range s {
		if _, isEssential := fs.essentialRunes[r]; isEssential {
			runesFound[r] = struct{}{}
		}
	}
	if len(runesFound) > 0 {
		result.runesFoundInS = runesFound
	} else {
		// If no essential runes were actually found in the string, set to nil
		// to signify this in ShouldAttemptMatch.
		result.runesFoundInS = nil
	}

	return result
}

// ShouldAttemptMatch checks if a specific pattern (by index) passes the rune filter
// based on the pre-filtered runes found in the string.
func (pr PreFilterResult) ShouldAttemptMatch(patternIndex int) bool {
	patternMandatory := pr.filterSet.patterns[patternIndex].mandatory

	// If globally no essential runes existed OR none were found in the string...
	if pr.runesFoundInS == nil {
		// ...allow match attempt only if the specific pattern also requires no runes.
		return len(patternMandatory) == 0
	}

	// If essential runes were found in the string, check if this pattern's
	// mandatory runes are all present in the found set.
	if len(patternMandatory) == 0 {
		return true // Pattern requires no runes, always attempt match
	}

	for requiredRune := range patternMandatory {
		if _, found := pr.runesFoundInS[requiredRune]; !found {
			return false // A mandatory rune for this pattern is missing from the string
		}
	}
	return true // All mandatory runes for this pattern were found in the string
}

// findAllMandatoryRunes recursively finds runes that *must* be present for the regex to match.
// It skips adding runes found within optional parts of the regex (e.g., inside |, *, ?).
// This remains internal to the runes package.
func findAllMandatoryRunes(re *syntax.Regexp, foundRunes map[rune]struct{}, isOptional bool) {
	// If this part of the regex is optional, none of its literals are mandatory.
	if re == nil || isOptional {
		return
	}
	switch re.Op {
	case syntax.OpLiteral:
		// If we are in a non-optional part, add all literal runes.
		for _, r := range re.Rune {
			foundRunes[r] = struct{}{}
		}
	case syntax.OpConcat, syntax.OpCapture:
		// All sub-expressions must match, so they inherit the optional status.
		for _, sub := range re.Sub {
			findAllMandatoryRunes(sub, foundRunes, isOptional) // Pass down isOptional
		}
	case syntax.OpAlternate, syntax.OpStar, syntax.OpQuest:
		// Sub-expressions within these are optional by definition.
		for _, sub := range re.Sub {
			findAllMandatoryRunes(sub, foundRunes, true) // Mark sub-expressions as optional
		}
	case syntax.OpPlus:
		// The first repetition is mandatory (inherits isOptional), the rest are optional.
		if len(re.Sub) > 0 {
			findAllMandatoryRunes(re.Sub[0], foundRunes, isOptional) // First is mandatory if parent is
		}
	case syntax.OpRepeat:
		// If Min > 0, the first Min repetitions are mandatory (inherit isOptional).
		if re.Min > 0 && len(re.Sub) > 0 {
			findAllMandatoryRunes(re.Sub[0], foundRunes, isOptional) // Mandatory if parent is
		} else if len(re.Sub) > 0 {
			findAllMandatoryRunes(re.Sub[0], foundRunes, true) // Treat like OpStar if Min is 0
		}
	}
	// Other Ops like OpAnyCharNotNL, OpCharClass, etc., don't guarantee specific literal runes.
}
