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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw"
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

func TestParseEventData_AllFieldsInBody(t *testing.T) {
	lr := &logsReceiver{
		cfg:      createDefaultConfig().(*Config),
		logger:   zap.NewNop(),
		consumer: consumertest.NewNop(),
		wg:       nil,
	}

	event := &etw.Event{
		Session:   "TestSession",
		Timestamp: time.Now(),
		System: etw.EventSystem{
			Channel:  "Security",
			Computer: "MYCOMPUTER",
			EventID:  "4624",
			Version:  2,
			Level:    4,
			Keywords: "9232379236109516800",
			Opcode:   "Info",
			Task:     "Logon",
			TimeCreated: etw.EventTimeCreated{
				SystemTime: time.Date(2024, 3, 13, 12, 34, 56, 789000000, time.UTC),
			},
			Provider: etw.EventProvider{
				Name: "Microsoft-Windows-Security-Auditing",
				GUID: "{54849625-5478-4994-A5BA-3E3B0328C30D}",
			},
			Execution: etw.EventExecution{
				ProcessID: 1234,
				ThreadID:  5678,
			},
			Correlation: etw.EventCorrelation{
				ActivityID:        "{AAA-BBB}",
				RelatedActivityID: "{CCC-DDD}",
			},
		},
		Security: etw.EventSecurity{SID: "S-1-5-18"},
		EventData: map[string]any{
			"SubjectUserName": "SYSTEM",
		},
	}

	logs, err := lr.parseEvent(event)
	require.NoError(t, err)

	body := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Map()

	assertBodyStr := func(key, want string) {
		t.Helper()
		v, ok := body.Get(key)
		require.True(t, ok, "expected body key %q", key)
		assert.Equal(t, want, v.Str())
	}

	assertNestedStr := func(mapKey, fieldKey, want string) {
		t.Helper()
		parent, ok := body.Get(mapKey)
		require.True(t, ok, "expected body map %q", mapKey)
		v, ok := parent.Map().Get(fieldKey)
		require.True(t, ok, "expected key %q inside %q", fieldKey, mapKey)
		assert.Equal(t, want, v.Str())
	}

	// Fields from Event Viewer XML system block
	assertBodyStr("channel", "Security")
	assertBodyStr("computer", "MYCOMPUTER")
	assertBodyStr("level", "4")
	assertBodyStr("keywords", "9232379236109516800")

	// event_id carries the id; version is a top-level field
	assertNestedStr("event_id", "id", "4624")
	assertBodyStr("version", "2")
	assertBodyStr("time_created", "2024-03-13T12:34:56.789Z")

	// execution carries process_id and thread_id
	assertNestedStr("execution", "process_id", "1234")
	assertNestedStr("execution", "thread_id", "5678")

	// Verify resource attributes still present (backwards compat)
	attrs := logs.ResourceLogs().At(0).Resource().Attributes()
	ch, ok := attrs.Get("channel")
	require.True(t, ok)
	assert.Equal(t, "Security", ch.Str())
	comp, ok := attrs.Get("computer")
	require.True(t, ok)
	assert.Equal(t, "MYCOMPUTER", comp.Str())
}
