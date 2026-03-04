// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ocsfstandardizationprocessor

import (
	"strconv"
	"time"
)

// coerceType coerces a value to the expected OCSF type.
// Only numeric and boolean coercions are applied. String-typed fields are left
// as-is because converting numeric values to strings would break range validation.
func coerceType(value any, typeName string) any {
	switch typeName {
	case "integer":
		return coerceToInt(value)
	case "long":
		return coerceToInt64(value)
	case "float":
		return coerceToFloat64(value)
	case "boolean":
		return coerceToBool(value)
	case "datetime":
		return coerceToDatetime(value)
	case "timestamp":
		return coerceToTimestamp(value)
	default:
		return value
	}
}

func coerceToInt(value any) any {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
		return value
	case bool:
		if v {
			return 1
		}
		return 0
	default:
		return value
	}
}

func coerceToInt64(value any) any {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
		return value
	case bool:
		if v {
			return int64(1)
		}
		return int64(0)
	default:
		return value
	}
}

func coerceToFloat64(value any) any {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
		return value
	default:
		return value
	}
}

func coerceToBool(value any) any {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
		return value
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	default:
		return value
	}
}

// coerceToTimestamp converts values to epoch milliseconds (int64).
// Handles numeric types directly and parses RFC3339 datetime strings.
func coerceToTimestamp(value any) any {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UnixMilli()
		}
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t.UnixMilli()
		}
		return value
	default:
		return value
	}
}

// coerceToDatetime converts epoch millisecond timestamps to RFC3339 strings.
// String values are returned as-is (assumed to already be formatted).
func coerceToDatetime(value any) any {
	switch v := value.(type) {
	case string:
		return v
	case int64:
		return time.UnixMilli(v).UTC().Format(time.RFC3339)
	case int:
		return time.UnixMilli(int64(v)).UTC().Format(time.RFC3339)
	case float64:
		return time.UnixMilli(int64(v)).UTC().Format(time.RFC3339)
	default:
		return value
	}
}
