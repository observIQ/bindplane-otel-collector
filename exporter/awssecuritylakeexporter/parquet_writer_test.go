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
	"io"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listRecord struct {
	Tags   []*string `parquet:"tags,optional,list"`
	Values []*int64  `parquet:"values,optional,list"`
	Time   *int64    `parquet:"time,optional"`
}

func TestWrite_Columns(t *testing.T) {
	schema := parquet.SchemaOf((*listRecord)(nil))

	tests := []struct {
		name         string
		records      []map[string]any
		wantRows     int
		wantTagCount []int      // number of tag values per row
		wantTags     [][]string // expected tag strings per row
		wantIntCount []int      // number of int values per row
		wantInts     [][]int64  // expected int values per row
	}{
		{
			name: "single element list",
			records: []map[string]any{
				{"tags": []any{"alpha"}, "values": []any{int64(10)}, "time": int64(1000)},
			},
			wantRows:     1,
			wantTagCount: []int{1},
			wantTags:     [][]string{{"alpha"}},
			wantIntCount: []int{1},
			wantInts:     [][]int64{{10}},
		},
		{
			name: "multiple elements",
			records: []map[string]any{
				{"tags": []any{"alpha", "beta", "gamma"}, "values": []any{int64(10), int64(20), int64(30)}, "time": int64(1000)},
			},
			wantRows:     1,
			wantTagCount: []int{3},
			wantTags:     [][]string{{"alpha", "beta", "gamma"}},
			wantIntCount: []int{3},
			wantInts:     [][]int64{{10, 20, 30}},
		},
		{
			name: "empty list",
			records: []map[string]any{
				{"tags": []any{}, "values": []any{}, "time": int64(1000)},
			},
			wantRows:     1,
			wantTagCount: []int{1}, // single null value
			wantIntCount: []int{1},
		},
		{
			name: "missing list fields",
			records: []map[string]any{
				{"time": int64(1000)},
			},
			wantRows:     1,
			wantTagCount: []int{1}, // single null value
			wantIntCount: []int{1},
		},
		{
			name: "list with null element",
			records: []map[string]any{
				{"tags": []any{"alpha", nil, "gamma"}, "time": int64(1000)},
			},
			wantRows:     1,
			wantTagCount: []int{3},
			wantTags:     [][]string{{"alpha", "<null>", "gamma"}}, // null parquet value
		},
		{
			name: "multiple records with different list sizes",
			records: []map[string]any{
				{"tags": []any{"a", "b"}, "time": int64(1000)},
				{"tags": []any{"x", "y", "z"}, "time": int64(2000)},
			},
			wantRows:     2,
			wantTagCount: []int{2, 3},
			wantTags:     [][]string{{"a", "b"}, {"x", "y", "z"}},
		},
		{
			name: "typed string slice",
			records: []map[string]any{
				{"tags": []string{"one", "two", "three"}, "time": int64(1000)},
			},
			wantRows:     1,
			wantTagCount: []int{3},
			wantTags:     [][]string{{"one", "two", "three"}},
		},
		{
			name: "typed int64 slice",
			records: []map[string]any{
				{"values": []int64{100, 200, 300}, "time": int64(1000)},
			},
			wantRows:     1,
			wantIntCount: []int{3},
			wantInts:     [][]int64{{100, 200, 300}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := WriteParquet(schema, tt.records)
			require.NoError(t, err)

			rows := readBack(t, buf, schema)
			require.Len(t, rows, tt.wantRows)

			for i, row := range rows {
				if i < len(tt.wantTagCount) {
					tagVals := filterColumn(row, 0)
					assert.Len(t, tagVals, tt.wantTagCount[i], "row %d tag count", i)

					if i < len(tt.wantTags) {
						for j, want := range tt.wantTags[i] {
							assert.Equal(t, want, tagVals[j].String(), "row %d tag[%d]", i, j)
						}
					}
				}

				if i < len(tt.wantIntCount) {
					intVals := filterColumn(row, 1)
					assert.Len(t, intVals, tt.wantIntCount[i], "row %d int count", i)

					if i < len(tt.wantInts) {
						for j, want := range tt.wantInts[i] {
							assert.Equal(t, want, intVals[j].Int64(), "row %d value[%d]", i, j)
						}
					}
				}
			}
		})
	}
}

func TestWrite_RepetitionLevels(t *testing.T) {
	schema := parquet.SchemaOf((*listRecord)(nil))
	records := []map[string]any{
		{"tags": []any{"a", "b", "c"}, "time": int64(1000)},
	}

	buf, err := WriteParquet(schema, records)
	require.NoError(t, err)

	rows := readBack(t, buf, schema)
	require.Len(t, rows, 1)

	tagVals := filterColumn(rows[0], 0)
	require.Len(t, tagVals, 3)

	assert.Equal(t, 0, tagVals[0].RepetitionLevel(), "first element rep level")
	assert.Equal(t, 1, tagVals[1].RepetitionLevel(), "second element rep level")
	assert.Equal(t, 1, tagVals[2].RepetitionLevel(), "third element rep level")
}

func readBack(t *testing.T, buf *bytes.Buffer, schema *parquet.Schema) []parquet.Row {
	t.Helper()
	reader := parquet.NewReader(bytes.NewReader(buf.Bytes()), schema)
	defer func() {
		if err := reader.Close(); err != nil {
			t.Errorf("close reader: %v", err)
		}
	}()

	var rows []parquet.Row
	for {
		row := make(parquet.Row, len(schema.Columns())*4) // extra space for repeated columns
		n, err := reader.ReadRows([]parquet.Row{row})
		if n > 0 {
			rows = append(rows, row[:rowLen(row)])
		}
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}
	return rows
}

// rowLen returns the number of non-zero-initialized values in a row.
func rowLen(row parquet.Row) int {
	for i := len(row) - 1; i >= 0; i-- {
		if row[i].Column() >= 0 {
			return i + 1
		}
	}
	return 0
}

func filterColumn(row parquet.Row, colIdx int) []parquet.Value {
	var values []parquet.Value
	for _, v := range row {
		if v.Column() == colIdx {
			values = append(values, v)
		}
	}
	return values
}
