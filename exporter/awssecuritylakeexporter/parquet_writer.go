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

package awssecuritylakeexporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/zstd"
	"github.com/parquet-go/parquet-go/encoding/plain"
)

// WriteParquet serializes OCSF records to Parquet format with ZSTD compression.
// Records are sorted by the "time" field before writing.
func WriteParquet(schema *parquet.Schema, records []map[string]any) (*bytes.Buffer, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records to write")
	}

	sortByTime(records)

	buf := new(bytes.Buffer)
	writer := parquet.NewWriter(buf,
		schema,
		parquet.Compression(&zstd.Codec{}),
		parquet.DefaultEncoding(&plain.Encoding{}),
	)

	for _, record := range records {
		row, err := mapToRow(schema, record)
		if err != nil {
			return nil, fmt.Errorf("converting record to row: %w", err)
		}
		if _, err := writer.WriteRows([]parquet.Row{row}); err != nil {
			return nil, fmt.Errorf("writing row: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing parquet writer: %w", err)
	}

	return buf, nil
}

// sortByTime sorts records in place by the "time" field ascending.
func sortByTime(records []map[string]any) {
	sort.SliceStable(records, func(i, j int) bool {
		ti, _ := toInt64(records[i]["time"])
		tj, _ := toInt64(records[j]["time"])
		return ti < tj
	})
}

// mapToRow converts a map[string]any to a parquet.Row by walking the schema's leaf columns.
func mapToRow(schema *parquet.Schema, record map[string]any) (parquet.Row, error) {
	columns := schema.Columns()
	row := make(parquet.Row, 0, len(columns))

	for _, path := range columns {
		leaf, ok := schema.Lookup(path...)
		if !ok {
			return nil, fmt.Errorf("column not found in schema: %v", path)
		}

		val, defLevel := lookupNestedValue(record, path, leaf)
		values := makeValues(val, defLevel, leaf)
		row = append(row, values...)
	}

	return row, nil
}

// lookupNestedValue traverses the map following the path and returns the leaf
// value along with the effective definition level.
//
// For repeated (list) columns the full slice value is returned with maxDef so
// the caller can handle the first element.
// For scalar columns the definition level reflects how deep into the path a nil
// was first encountered: present = maxDef, null at depth d = d.
func lookupNestedValue(m map[string]any, path []string, leaf parquet.LeafColumn) (any, int) {
	maxDef := leaf.MaxDefinitionLevel

	// For repeated columns walk the full path; if the slice is present return it
	// with maxDef so makeValue can inspect it.
	if leaf.MaxRepetitionLevel > 0 {
		current := any(m)
		for depth, key := range path {
			switch v := current.(type) {
			case map[string]any:
				val, exists := v[key]
				if !exists || val == nil {
					return nil, depth
				}
				current = val
			default:
				// Hit a non-map value (e.g., slice for list data) before
				// exhausting the schema path — this is the list data.
				return current, maxDef
			}
		}
		return current, maxDef
	}

	current := any(m)
	for depth, key := range path {
		switch v := current.(type) {
		case map[string]any:
			val, exists := v[key]
			if !exists || val == nil {
				// def level = number of ancestors already resolved.
				def := depth
				if def >= maxDef {
					def = maxDef - 1
				}
				return nil, def
			}
			current = val
		default:
			def := depth
			if def >= maxDef {
				def = maxDef - 1
			}
			return nil, def
		}
	}
	return current, maxDef
}

// makeValues creates parquet.Value(s) with the correct definition/repetition levels.
// For scalar columns a single value is returned. For list (repeated) columns one
// value per element is returned with the appropriate repetition levels.
func makeValues(val any, defLevel int, leaf parquet.LeafColumn) []parquet.Value {
	maxDef := leaf.MaxDefinitionLevel
	maxRep := leaf.MaxRepetitionLevel
	colIdx := leaf.ColumnIndex

	// List (repeated) columns.
	if maxRep > 0 {
		if val == nil {
			return []parquet.Value{parquet.Value{}.Level(0, defLevel, colIdx)}
		}
		return makeListValues(val, leaf)
	}

	// Scalar null.
	if val == nil {
		return []parquet.Value{parquet.Value{}.Level(0, defLevel, colIdx)}
	}

	// Scalar present.
	pv := scalarValue(val, leaf)
	return []parquet.Value{pv.Level(0, maxDef, colIdx)}
}

// makeListValues returns parquet.Values for all elements of a list column.
// The first element gets repetition level 0, subsequent elements get maxRep.
// Empty lists emit a single null value at def level 0.
func makeListValues(val any, leaf parquet.LeafColumn) []parquet.Value {
	maxDef := leaf.MaxDefinitionLevel
	maxRep := leaf.MaxRepetitionLevel
	colIdx := leaf.ColumnIndex

	switch s := val.(type) {
	case []any:
		if len(s) == 0 {
			return []parquet.Value{parquet.Value{}.Level(0, 0, colIdx)}
		}
		values := make([]parquet.Value, 0, len(s))
		for i, elem := range s {
			rep := maxRep
			if i == 0 {
				rep = 0
			}
			if elem == nil {
				values = append(values, parquet.Value{}.Level(rep, maxDef-1, colIdx))
			} else {
				values = append(values, scalarValue(elem, leaf).Level(rep, maxDef, colIdx))
			}
		}
		return values
	case []string:
		if len(s) == 0 {
			return []parquet.Value{parquet.Value{}.Level(0, 0, colIdx)}
		}
		values := make([]parquet.Value, 0, len(s))
		for i, elem := range s {
			rep := maxRep
			if i == 0 {
				rep = 0
			}
			values = append(values, parquet.ByteArrayValue([]byte(elem)).Level(rep, maxDef, colIdx))
		}
		return values
	case []int32:
		if len(s) == 0 {
			return []parquet.Value{parquet.Value{}.Level(0, 0, colIdx)}
		}
		values := make([]parquet.Value, 0, len(s))
		for i, elem := range s {
			rep := maxRep
			if i == 0 {
				rep = 0
			}
			values = append(values, parquet.Int32Value(elem).Level(rep, maxDef, colIdx))
		}
		return values
	case []int64:
		if len(s) == 0 {
			return []parquet.Value{parquet.Value{}.Level(0, 0, colIdx)}
		}
		values := make([]parquet.Value, 0, len(s))
		for i, elem := range s {
			rep := maxRep
			if i == 0 {
				rep = 0
			}
			values = append(values, parquet.Int64Value(elem).Level(rep, maxDef, colIdx))
		}
		return values
	default:
		return []parquet.Value{parquet.Value{}.Level(0, 0, colIdx)}
	}
}

// scalarValue creates a parquet.Value for a scalar without setting levels.
// Levels must be set by the caller via .Level(rep, def, col).
func scalarValue(val any, leaf parquet.LeafColumn) parquet.Value {
	switch leaf.Node.Type().Kind() {
	case parquet.Boolean:
		b, _ := val.(bool)
		return parquet.BooleanValue(b)
	case parquet.Int32:
		i, _ := toInt32(val)
		return parquet.Int32Value(i)
	case parquet.Int64:
		i, _ := toInt64(val)
		return parquet.Int64Value(i)
	case parquet.Float:
		f, _ := toFloat32(val)
		return parquet.FloatValue(f)
	case parquet.Double:
		f, _ := toFloat64(val)
		return parquet.DoubleValue(f)
	case parquet.ByteArray:
		s := toString(val)
		return parquet.ByteArrayValue([]byte(s))
	case parquet.FixedLenByteArray:
		s := toString(val)
		return parquet.FixedLenByteArrayValue([]byte(s))
	default:
		return parquet.ByteArrayValue(fmt.Appendf(nil, "%v", val))
	}
}

// toInt64 converts common numeric types to int64.
func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	case float32:
		return int64(n), true
	case int32:
		return int64(n), true
	case uint64:
		return int64(n), true //nolint:gosec
	case uint32:
		return int64(n), true
	default:
		return 0, false
	}
}

// toInt32 converts common numeric types to int32.
func toInt32(v any) (int32, bool) {
	switch n := v.(type) {
	case int32:
		return n, true
	case int:
		return int32(n), true //nolint:gosec
	case float64:
		return int32(n), true //nolint:gosec
	case int64:
		return int32(n), true //nolint:gosec
	default:
		return 0, false
	}
}

// toFloat32 converts common numeric types to float32.
func toFloat32(v any) (float32, bool) {
	switch n := v.(type) {
	case float32:
		return n, true
	case float64:
		return float32(n), true
	case int:
		return float32(n), true
	case int64:
		return float32(n), true
	default:
		return 0, false
	}
}

// toFloat64 converts common numeric types to float64.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// toString converts a value to its string representation.
// Maps and slices are JSON-encoded so that fields serialized as *string
// (e.g., circular OCSF refs, schemaless fields) produce valid JSON
// rather than Go's fmt.Sprint format like "map[key:value]".
func toString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	case map[string]any, []any:
		b, err := json.Marshal(s)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(b)
	default:
		return fmt.Sprint(v)
	}
}
