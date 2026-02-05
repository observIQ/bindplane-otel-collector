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

// parseTimeFromPattern extracts a timestamp from a blob path using a custom pattern.
// Supports two formats:
// 1. Named placeholders: {year}/{month}/{day}/{hour}/{minute}
// 2. Go time format: 2006/01/02/15/04
func parseTimeFromPattern(blobPath, pattern string) (*time.Time, error) {
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
		if pattern[i] != '{' {
			i++
			continue
		}

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

// generateTimePrefixes generates a list of prefixes based on the time range and pattern.
// It limits the resolution to the hour to avoid race conditions.
func generateTimePrefixes(startTime, endingTime time.Time, pattern, rootFolder string) ([]string, error) {
	// 1. Convert pattern to Go time layout
	layout := pattern
	if strings.Contains(pattern, "{") {
		layout = convertPlaceholdersToLayout(pattern)
	}

	// 2. Truncate layout to hour precision
	// We look for the "15" (hour) component in the layout.
	// If we find it, we keep everything up to it.
	// If not, we look for "02" (day), etc.
	truncatedLayout := layout
	idx := strings.Index(layout, "15")
	resolution := time.Hour

	if idx != -1 {
		// Keep "15" (length 2)
		truncatedLayout = layout[:idx+2]
	} else {
		// Fallback to Day if Hour not present
		idx = strings.Index(layout, "02")
		if idx != -1 {
			truncatedLayout = layout[:idx+2]
			resolution = 24 * time.Hour
		}
		// If neither hour nor day is present, we can't optimize much based on time.
		// Return the root folder or empty prefix effectively?
		// Actually, if we can't find time components, we should probably fall back to scanning everything.
		// But let's assume the user configured a valid time pattern.
		// If no time components found, let's just use the layout as is (might be static or have only year/month)
	}

	// 3. Generate prefixes
	prefixes := []string{}
	seen := make(map[string]bool)

	// Align start time to resolution
	// Use UTC to avoid DST issues (though caller should provide UTC)

	for current := startTime.UTC().Truncate(resolution); !current.After(endingTime.UTC()); current = current.Add(resolution) {
		prefix := current.Format(truncatedLayout)
		if rootFolder != "" {
			prefix = strings.TrimSuffix(rootFolder, "/") + "/" + strings.TrimPrefix(prefix, "/")
		}

		if !seen[prefix] {
			prefixes = append(prefixes, prefix)
			seen[prefix] = true
		}
	}

	return prefixes, nil
}

func convertPlaceholdersToLayout(pattern string) string {
	replacements := map[string]string{
		"{year}":   "2006",
		"{month}":  "01",
		"{day}":    "02",
		"{hour}":   "15",
		"{minute}": "04",
		"{second}": "05",
	}
	layout := pattern
	for k, v := range replacements {
		layout = strings.ReplaceAll(layout, k, v)
	}
	return layout
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
