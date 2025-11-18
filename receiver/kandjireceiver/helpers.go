package kandjireceiver

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// validateParamName ensures parameter names are safe (alphanumeric + underscore/hyphen)
func validateParamName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched
}

// sanitizeStringParam trims whitespace and removes control characters from string params
func sanitizeStringParam(value string, constraints *ParamConstraints) string {
	// Trim whitespace
	sanitized := strings.TrimSpace(value)

	// Remove control characters (keep printable chars, newline, tab, carriage return)
	var b strings.Builder
	for _, r := range sanitized {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' || r == '\r' {
			b.WriteRune(r)
		}
	}
	sanitized = b.String()

	// Apply length constraints if specified
	if constraints != nil {
		if constraints.MaxLen != nil && len(sanitized) > *constraints.MaxLen {
			sanitized = sanitized[:*constraints.MaxLen]
		}
		if constraints.MinLen != nil && len(sanitized) < *constraints.MinLen {
			// Return empty if below min length (will be caught by validation)
			return ""
		}
	}

	return sanitized
}

// validateUUIDFormat checks if a string is a valid UUID format
func validateUUIDFormat(value string) bool {
	matched, _ := regexp.MatchString(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`, value)
	return matched
}

func ValidateParams(ep KandjiEndpoint, params map[string]any) error {
	spec, ok := EndpointRegistry[ep]
	if !ok {
		return fmt.Errorf("unknown endpoint: %s", ep)
	}

	// Validate all parameter names in the map
	for paramName := range params {
		if !validateParamName(paramName) {
			return fmt.Errorf("invalid parameter name: %q (must be alphanumeric with underscores/hyphens)", paramName)
		}
	}

	for _, p := range spec.Params {
		v, exists := params[p.Name]

		if p.Required && !exists {
			return fmt.Errorf("missing required param: %s", p.Name)
		}

		if !exists {
			continue
		}

		switch p.Type {
		case ParamString:
			sv, ok := v.(string)
			if !ok {
				return fmt.Errorf("param %s must be string", p.Name)
			}

			// Sanitize string parameter
			sanitized := sanitizeStringParam(sv, p.Constraints)
			if sanitized != sv {
				// Update the param map with sanitized value
				params[p.Name] = sanitized
			}

			// Apply length constraints
			if p.Constraints != nil {
				if p.Constraints.MaxLen != nil && len(sanitized) > *p.Constraints.MaxLen {
					return fmt.Errorf("param %s exceeds max length of %d", p.Name, *p.Constraints.MaxLen)
				}
				if p.Constraints.MinLen != nil && len(sanitized) < *p.Constraints.MinLen {
					return fmt.Errorf("param %s must be at least %d characters", p.Name, *p.Constraints.MinLen)
				}
			}
		case ParamInt:
			intval, ok := v.(int)
			if !ok {
				return fmt.Errorf("param %s must be int", p.Name)
			}

			if p.Constraints != nil {
				if p.Constraints.MinInt != nil && intval < *p.Constraints.MinInt {
					return fmt.Errorf("param %s must be >= %d", p.Name, *p.Constraints.MinInt)
				}
				if p.Constraints.MaxInt != nil && intval > *p.Constraints.MaxInt {
					return fmt.Errorf("param %s must be <= %d", p.Name, *p.Constraints.MaxInt)
				}
			}

		case ParamBool:
			if _, ok := v.(bool); !ok {
				return fmt.Errorf("param %s must be bool", p.Name)
			}
		case ParamUUID:
			sv, ok := v.(string)
			if !ok {
				return fmt.Errorf("param %s must be UUID string", p.Name)
			}
			// Validate UUID format (8-4-4-4-12 hex digits)
			sv = strings.TrimSpace(sv)
			if !validateUUIDFormat(sv) {
				return fmt.Errorf("param %s must be a valid UUID format (e.g., 123e4567-e89b-12d3-a456-426614174000)", p.Name)
			}
			// Update with trimmed value
			if sv != v.(string) {
				params[p.Name] = sv
			}
		case ParamTime:
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("param %s must be RFC3339 string", p.Name)
			}
			// Trim whitespace for time strings
			s = strings.TrimSpace(s)

			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return fmt.Errorf("param %s must be valid RFC3339 time: %v", p.Name, err)
			}
			// Update with trimmed value
			if s != v.(string) {
				params[p.Name] = s
			}

			if p.Constraints != nil {
				// Future disallowed
				if p.Constraints.NotNewerThanNow && t.After(time.Now()) {
					return fmt.Errorf("param %s cannot be in the future", p.Name)
				}

				// NotBefore
				if p.Constraints.NotBefore != nil && t.Before(*p.Constraints.NotBefore) {
					return fmt.Errorf("param %s must be >= %s", p.Name, p.Constraints.NotBefore.Format(time.RFC3339))
				}

				// NotAfter
				if p.Constraints.NotAfter != nil && t.After(*p.Constraints.NotAfter) {
					return fmt.Errorf("param %s must be <= %s", p.Name, p.Constraints.NotAfter.Format(time.RFC3339))
				}

				// MaxAge:
				if p.Constraints.MaxAge != nil {
					cutoff := time.Now().Add(-*p.Constraints.MaxAge)
					if t.Before(cutoff) {
						return fmt.Errorf("param %s exceeds max age of %s", p.Name, p.Constraints.MaxAge.String())
					}
				}
			}
			// optionally parse with time.Parse
		case ParamEnum:
			sv, ok := v.(string)
			if !ok {
				return fmt.Errorf("param %s must be string enum", p.Name)
			}
			// Trim whitespace for enum values
			sv = strings.TrimSpace(sv)
			if len(p.AllowedVals) > 0 {
				valid := false
				for _, allowed := range p.AllowedVals {
					// Case-sensitive exact match
					if sv == allowed {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("param %s must be one of %v", p.Name, p.AllowedVals)
				}
			}
			// Update with trimmed value
			if sv != v.(string) {
				params[p.Name] = sv
			}
		case ParamFloat:
			if _, ok := v.(float64); !ok {
				return fmt.Errorf("param %s must be float64", p.Name)
			}
		}
	}

	return nil
}

func normalizeCursor(raw *string) *string {
	if raw == nil {
		return nil
	}

	normalized := sanitizeCursor(*raw)
	if normalized == "" {
		return nil
	}

	return &normalized
}

func sanitizeCursor(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	if parsed, err := url.Parse(trimmed); err == nil {
		if cursor := parsed.Query().Get("cursor"); cursor != "" {
			return cursor
		}
	}

	if strings.Contains(trimmed, "=") {
		if vals, err := url.ParseQuery(strings.TrimPrefix(trimmed, "?")); err == nil {
			if cursor := vals.Get("cursor"); cursor != "" {
				return cursor
			}
		}
	}

	return trimmed
}
