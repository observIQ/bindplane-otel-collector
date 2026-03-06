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

//go:build windows

package windowseventtracereceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestPutAnyValue_String(t *testing.T) {
	m := pcommon.NewMap()
	putAnyValue(m, "key", "value")
	v, ok := m.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "value", v.Str())
}

func TestPutAnyValue_NestedMap(t *testing.T) {
	m := pcommon.NewMap()
	putAnyValue(m, "nested", map[string]any{
		"a": "1",
		"b": "2",
	})
	v, ok := m.Get("nested")
	assert.True(t, ok)
	assert.Equal(t, pcommon.ValueTypeMap, v.Type())
	inner := v.Map()
	a, ok := inner.Get("a")
	assert.True(t, ok)
	assert.Equal(t, "1", a.Str())
	b, ok := inner.Get("b")
	assert.True(t, ok)
	assert.Equal(t, "2", b.Str())
}

func TestPutAnyValue_Slice(t *testing.T) {
	m := pcommon.NewMap()
	putAnyValue(m, "items", []any{"x", "y", "z"})
	v, ok := m.Get("items")
	assert.True(t, ok)
	assert.Equal(t, pcommon.ValueTypeSlice, v.Type())
	s := v.Slice()
	assert.Equal(t, 3, s.Len())
	assert.Equal(t, "x", s.At(0).Str())
	assert.Equal(t, "y", s.At(1).Str())
	assert.Equal(t, "z", s.At(2).Str())
}

func TestPutAnyValue_SliceOfMaps(t *testing.T) {
	m := pcommon.NewMap()
	putAnyValue(m, "records", []any{
		map[string]any{"id": "1", "name": "foo"},
		map[string]any{"id": "2", "name": "bar"},
	})
	v, ok := m.Get("records")
	assert.True(t, ok)
	assert.Equal(t, pcommon.ValueTypeSlice, v.Type())
	s := v.Slice()
	assert.Equal(t, 2, s.Len())

	first := s.At(0).Map()
	id, ok := first.Get("id")
	assert.True(t, ok)
	assert.Equal(t, "1", id.Str())

	name, ok := first.Get("name")
	assert.True(t, ok)
	assert.Equal(t, "foo", name.Str())
}

func TestPutAnyValue_DeeplyNested(t *testing.T) {
	m := pcommon.NewMap()
	putAnyValue(m, "outer", map[string]any{
		"inner": map[string]any{
			"leaf": "value",
		},
	})
	v, ok := m.Get("outer")
	assert.True(t, ok)
	inner, ok := v.Map().Get("inner")
	assert.True(t, ok)
	leaf, ok := inner.Map().Get("leaf")
	assert.True(t, ok)
	assert.Equal(t, "value", leaf.Str())
}

func TestPutAnyValue_UnknownTypeFallback(t *testing.T) {
	m := pcommon.NewMap()
	putAnyValue(m, "num", 42)
	v, ok := m.Get("num")
	assert.True(t, ok)
	assert.Equal(t, "42", v.Str())
}
