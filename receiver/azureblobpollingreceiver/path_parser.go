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
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseTimeFromPattern extracts a timestamp from a blob path using a custom pattern.
// Supports two formats:
// 1. Named placeholders: {year}/{month}/{day}/{hour}/{minute}
// 2. Go time format: 2006/01/02/15/04
func ParseTimeFromPattern(blobPath, pattern string) (*time.Time, error) {
	// Try named placeholders first
	if strings.Contains(pattern, "{") {
		return parseWithPlaceholders(blobPath, pattern)
	}

	// Try Go time format
	return parseWithGoTimeFormat(blobPath, pattern)
}

// parseWithPlaceholders parses using named placeholders like {year}, {month}, etc.
func parseWithPlaceholders(blobPath, pattern string) (*time.Time, error) {
	// Map placeholders to regex patterns
	placeholderMap := map[string]string{
		"{year}":   `(\d{4})`,
		"{month}":  `(\d{2})`,
		"{day}":    `(\d{2})`,
		"{hour}":   `(\d{2})`,
		"{minute}": `(\d{2})`,
		"{second}": `(\d{2})`,
	}

	// Track which placeholders are in the pattern, in order
	placeholders := []string{}
	regexPattern := regexp.QuoteMeta(pattern)

	// Find placeholders in order they appear in the pattern
	for i := 0; i < len(pattern); {
		if pattern[i] == '{' {
			// Find the end of the placeholder
			end := strings.Index(pattern[i:], "}")
			if end == -1 {
				break
			}
			placeholder := pattern[i : i+end+1]
			if regex, ok := placeholderMap[placeholder]; ok {
				placeholders = append(placeholders, strings.Trim(placeholder, "{}"))
				regexPattern = strings.Replace(regexPattern, regexp.QuoteMeta(placeholder), regex, 1)
			}
			i += end + 1
		} else {
			i++
		}
	}

	// Compile and match the regex
	re, err := regexp.Compile("^" + regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	matches := re.FindStringSubmatch(blobPath)
	if matches == nil {
		return nil, fmt.Errorf("path does not match pattern")
	}

	// Extract time components
	components := make(map[string]int)
	for i, placeholder := range placeholders {
		val, err := strconv.Atoi(matches[i+1])
		if err != nil {
			return nil, fmt.Errorf("invalid %s value: %s", placeholder, matches[i+1])
		}
		components[placeholder] = val
	}

	// Build time from components (default to 0 if not present)
	year := components["year"]
	month := components["month"]
	day := components["day"]
	hour := components["hour"]
	minute := components["minute"]
	second := components["second"]

	// Validate required components
	if year == 0 {
		return nil, fmt.Errorf("year is required in pattern")
	}
	if month == 0 {
		month = 1
	}
	if day == 0 {
		day = 1
	}

	parsedTime := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
	return &parsedTime, nil
}

// parseWithGoTimeFormat parses using Go's time format (e.g., "2006/01/02/15/04")
func parseWithGoTimeFormat(blobPath, pattern string) (*time.Time, error) {
	// Extract the portion of the blob path that matches the pattern length
	// This handles cases where the blob has additional path components or filename

	// Count the number of path separators in the pattern
	patternParts := strings.Split(pattern, "/")
	blobParts := strings.Split(blobPath, "/")

	// Take the same number of parts from the blob path as in the pattern
	if len(blobParts) < len(patternParts) {
		return nil, fmt.Errorf("blob path has fewer components than pattern")
	}

	// Extract the relevant portion of the blob path
	relevantPath := strings.Join(blobParts[:len(patternParts)], "/")

	// Parse using Go time format
	parsedTime, err := time.Parse(pattern, relevantPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time: %w", err)
	}

	return &parsedTime, nil
}
