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

	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor/internal/named"
)

// Matcher is a type that can match a string against a set of regexes.
type Matcher interface {
	Match(s string) string
}

type matcher struct {
	// Regexes are stored in a map for efficient lookup.
	regexMap map[string]*regexp.Regexp

	// Order of regexes is referred to as needed.
	regexOrder []string

	// If not empty, the value to return when no regex matches.
	defaultValue string
}

// New creates a new Matcher.
func New(regexes []named.Regex, defaultValue string) (Matcher, error) {
	if len(regexes) == 0 {
		return nil, errors.New("no regexes provided")
	}

	regexOrder := make([]string, 0, len(regexes))
	regexMap := make(map[string]*regexp.Regexp, len(regexes))
	for _, r := range regexes {
		if r.Name == "" {
			return nil, errors.New("regex name cannot be empty")
		}
		if _, ok := regexMap[r.Name]; ok {
			return nil, fmt.Errorf("regex name is duplicated: %s", r.Name)
		}
		if r.Regex == nil {
			return nil, fmt.Errorf("regex '%s' is nil", r.Name)
		}

		syntaxTree, err := syntax.Parse(r.Regex.String(), syntax.Perl)
		if err != nil {
			return nil, fmt.Errorf("error parsing regex '%s': %w", r.Name, err)
		}

		regexMap[r.Name] = regexp.MustCompile(syntaxTree.Simplify().String())

		regexOrder = append(regexOrder, r.Name)
	}

	return &matcher{
		regexOrder:   regexOrder,
		regexMap:     regexMap,
		defaultValue: defaultValue,
	}, nil
}

// Match finds the name of the first regex that matches the string s, using the FilterSet.
func (m *matcher) Match(s string) string {
	for _, p := range m.regexOrder {
		if m.regexMap[p].MatchString(s) {
			return p
		}
	}
	return m.defaultValue
}
