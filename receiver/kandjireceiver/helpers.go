package kandjireceiver

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

func ValidateParams(ep KandjiEndpoint, params map[string]any) error {
	spec, ok := EndpointRegistry[ep]
	if !ok {
		return fmt.Errorf("unknown endpoint: %s", ep)
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
			if _, ok := v.(string); !ok {
				return fmt.Errorf("param %s must be string", p.Name)
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
			if _, ok := v.(string); !ok {
				return fmt.Errorf("param %s must be UUID string", p.Name)
			}
			// could add UUID format check here later
		case ParamTime:
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("param %s must be RFC3339 string", p.Name)
			}

			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return fmt.Errorf("param %s must be valid RFC3339 time: %v", p.Name, err)
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
			if len(p.AllowedVals) > 0 {
				valid := false
				for _, allowed := range p.AllowedVals {
					if sv == allowed {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("param %s must be one of %v", p.Name, p.AllowedVals)
				}
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
