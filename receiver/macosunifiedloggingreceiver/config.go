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

//go:build darwin

package macosunifiedloggingreceiver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Valid field names for macOS unified logging predicates
var validPredicateFields = map[string]bool{
	"activityIdentifier":             true,
	"bootUUID":                       true,
	"category":                       true,
	"composedMessage":                true,
	"continuousNanosecondsSinceBoot": true,
	"creatorActivityIdentifier":      true,
	"creatorProcessUniqueIdentifier": true,
	"date":                           true,
	"formatString":                   true,
	"logType":                        true,
	"machContinuousTimestamp":        true,
	"parentActivityIdentifier":       true,
	"process":                        true,
	"processIdentifier":              true,
	"processImagePath":               true,
	"processImageUUID":               true,
	"sender":                         true,
	"senderImageOffset":              true,
	"senderImagePath":                true,
	"senderImageUUID":                true,
	"signpostIdentifier":             true,
	"signpostScope":                  true,
	"signpostType":                   true,
	"size":                           true,
	"subsystem":                      true,
	"threadIdentifier":               true,
	"timeToLive":                     true,
	"traceIdentifier":                true,
	"transitionActivityIdentifier":   true,
	"type":                           true,
}

// Valid event types
var validEventTypes = map[string]bool{
	"activityCreateEvent":     true,
	"activityTransitionEvent": true,
	"userActionEvent":         true,
	"traceEvent":              true,
	"logEvent":                true,
	"timesyncEvent":           true,
	"signpostEvent":           true,
	"lossEvent":               true,
	"stateEvent":              true,
}

// Valid log types
var validLogTypes = map[string]bool{
	"default": true,
	"release": true,
	"info":    true,
	"debug":   true,
	"error":   true,
	"fault":   true,
}

// Valid signpost scopes
var validSignpostScopes = map[string]bool{
	"thread":  true,
	"process": true,
	"system":  true,
}

// Valid signpost types
var validSignpostTypes = map[string]bool{
	"event": true,
	"begin": true,
	"end":   true,
}

// Valid comparison operators
var validOperators = []string{
	"AND", "&&", "&",
	"OR", "||",
	"NOT", "!",
	"!=", "<>", "==", "=",
	"<", ">", "<=", "=<", ">=", "=>",
	"BEGINSWITH",
	"CONTAINS",
	"ENDSWITH",
	"LIKE",
	"MATCHES",
}

// Validate checks the Config is valid
func (cfg *Config) Validate() error {

	// Set default format if not specified
	if cfg.Format == "" {
		cfg.Format = "default"
	}

	// Validate format
	validFormats := map[string]bool{
		"default": true,
		"ndjson":  true,
		"json":    true,
		"syslog":  true,
		"compact": true,
	}
	if !validFormats[cfg.Format] {
		return fmt.Errorf("invalid format: %s (valid options: default, ndjson, json, syslog, compact)", cfg.Format)
	}

	// Validate predicate to prevent invalid characters
	if cfg.Predicate != "" {
		if err := validatePredicate(&cfg.Predicate); err != nil {
			return fmt.Errorf("invalid predicate: %w", err)
		}
	}

	// Validate archive path if specified
	if cfg.ArchivePath != "" {
		// sanitize the archive path
		cfg.ArchivePath = filepath.Clean(cfg.ArchivePath)

		info, err := os.Stat(cfg.ArchivePath)
		if err != nil {
			return fmt.Errorf("archive_path does not exist: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("archive_path must be a directory (.logarchive)")
		}
	}

	// Validate time format if specified
	if cfg.StartTime != "" {
		if _, err := time.Parse("2006-01-02 15:04:05", cfg.StartTime); err != nil {
			return fmt.Errorf("invalid start_time format (expected: 2006-01-02 15:04:05): %w", err)
		}
	}

	if cfg.EndTime != "" {
		if cfg.ArchivePath == "" {
			return fmt.Errorf("end_time can only be used with archive_path")
		}
		if _, err := time.Parse("2006-01-02 15:04:05", cfg.EndTime); err != nil {
			return fmt.Errorf("invalid end_time format (expected: 2006-01-02 15:04:05): %w", err)
		}
	}

	return nil
}

// validatePredicate performs basic validation on the predicate expression
func validatePredicate(predicate *string) error {
	var errs error

	// Check for balanced quotes
	if !hasBalancedQuotes(*predicate) {
		errs = errors.Join(errs, errors.New("unbalanced quotes in predicate expression"))
	}

	// Check for balanced parentheses
	if !hasBalancedParentheses(*predicate) {
		errs = errors.Join(errs, errors.New("unbalanced parentheses in predicate expression"))
	}

	// Validate that at least one valid field name appears in the predicate
	hasValidField := false
	for field := range validPredicateFields {
		if strings.Contains(*predicate, field) {
			hasValidField = true
			break
		}
	}
	if !hasValidField {
		errs = errors.Join(errs, errors.New("predicate must contain at least one valid field name"))
	}

	hasValidOperator := false
	for _, op := range validOperators {
		if strings.Contains(*predicate, op) {
			hasValidOperator = true
			break
		}
	}
	if !hasValidOperator {
		errs = errors.Join(errs, errors.New("predicate must contain at least one valid operator"))
	}

	predicateUsesEventType := strings.Contains(*predicate, "type")
	if predicateUsesEventType {
		if !hasValidEventType(*predicate) {
			errs = errors.Join(errs, errors.New("predicate must contain at least one valid event type"))
		}
	}

	predicateUsesLogType := strings.Contains(*predicate, "logType")
	if predicateUsesLogType {
		if !hasValidLogType(*predicate) {
			errs = errors.Join(errs, errors.New("predicate must contain at least one valid log type"))
		}
	}

	predicateUsesSignpostScope := strings.Contains(*predicate, "signpostScope")
	if predicateUsesSignpostScope {
		if !hasValidSignpostScope(*predicate) {
			errs = errors.Join(errs, errors.New("predicate must contain at least one valid signpost scope"))
		}
	}

	predicateUsesSignpostType := strings.Contains(*predicate, "signpostType")
	if predicateUsesSignpostType {
		if !hasValidSignpostType(*predicate) {
			errs = errors.Join(errs, errors.New("predicate must contain at least one valid signpost type"))
		}
	}

	// Normalize && to AND to prevent command chaining
	*predicate = strings.ReplaceAll(*predicate, "&&", "AND")
	// Normalize || to OR to prevent command chaining
	*predicate = strings.ReplaceAll(*predicate, "||", "OR")

	invalidChars := []string{
		";",  // Command separator (not valid in predicates)
		"|",  // Pipe (not valid in predicates - use AND/OR instead)
		"$",  // Variable expansion (not valid in predicates)
		"`",  // Backtick command substitution (not valid in predicates)
		"\n", // Newline (not valid in predicates)
		"\r", // Carriage return (not valid in predicates)
		">>", // Append redirect (not valid in predicates)
		"<<", // Here document (not valid in predicates)
	}
	for _, char := range invalidChars {
		if strings.Contains(*predicate, char) {
			return fmt.Errorf("predicate contains invalid character: %q", char)
		}
	}

	return errs
}

func hasValidEventType(predicate string) bool {
	for eventType := range validEventTypes {
		if strings.Contains(predicate, eventType) {
			return true
		}
	}
	return false
}

func hasValidLogType(predicate string) bool {
	for logType := range validLogTypes {
		if strings.Contains(predicate, logType) {
			return true
		}
	}
	return false
}

func hasValidSignpostScope(predicate string) bool {
	for signpostScope := range validSignpostScopes {
		if strings.Contains(predicate, signpostScope) {
			return true
		}
	}
	return false
}

func hasValidSignpostType(predicate string) bool {
	for signpostType := range validSignpostTypes {
		if strings.Contains(predicate, signpostType) {
			return true
		}
	}
	return false
}

// hasBalancedQuotes checks if the string has balanced double quotes
func hasBalancedQuotes(s string) bool {
	count := 0
	escaped := false
	for _, ch := range s {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			count++
		}
	}
	return count%2 == 0
}

// hasBalancedParentheses checks if the string has balanced parentheses
func hasBalancedParentheses(s string) bool {
	count := 0
	for _, ch := range s {
		switch ch {
		case '(':
			count++
		case ')':
			count--
		}
		if count < 0 {
			return false
		}
	}
	return count == 0
}
