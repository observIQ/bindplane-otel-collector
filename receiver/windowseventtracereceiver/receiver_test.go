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

//go:build windows

package windowseventtracereceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw"
)

func newTestReceiver(t *testing.T) *logsReceiver {
	t.Helper()
	lr, err := newLogsReceiver(createTestConfig(), nil, zap.NewNop())
	require.NoError(t, err)
	return lr
}

func newFullEvent() *etw.Event {
	return &etw.Event{
		Session:      "TestSession",
		Flags:        "test-flags",
		Timestamp:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EventData:    map[string]any{"key1": "value1", "key2": 42},
		UserData:     map[string]any{"userKey": "userValue"},
		ExtendedData: []string{"ext1", "ext2"},
		Security:     etw.EventSecurity{SID: "S-1-5-18"},
		System: etw.EventSystem{
			ActivityID: "activity-id-123",
			Channel:    "Security",
			Computer:   "DESKTOP-ABC123",
			EventID:    "4624",
			Level:      4,
			Opcode:     "Info",
			Task:       "Logon",
			Keywords:   "0x8020000000000000",
			Version:    2,
			Provider: etw.EventProvider{
				Name: "Microsoft-Windows-Security-Auditing",
				GUID: "{54849625-5478-4994-A5BA-3E3B0328C30D}",
			},
			Correlation: etw.EventCorrelation{
				ActivityID:        "corr-activity-id",
				RelatedActivityID: "corr-related-id",
			},
			Execution: etw.EventExecution{
				ProcessID: 1234,
				ThreadID:  5678,
			},
		},
	}
}

func TestParseEventData_AllFieldsInBody(t *testing.T) {
	lr := newTestReceiver(t)
	event := newFullEvent()

	record := plog.NewLogRecord()
	lr.parseEventData(event, record)

	body := record.Body().Map()

	// Fields previously only in resource attributes - now also in body
	v, ok := body.Get("session")
	require.True(t, ok, "session should be in body")
	assert.Equal(t, "TestSession", v.Str())

	v, ok = body.Get("channel")
	require.True(t, ok, "channel should be in body")
	assert.Equal(t, "Security", v.Str())

	v, ok = body.Get("computer")
	require.True(t, ok, "computer should be in body")
	assert.Equal(t, "DESKTOP-ABC123", v.Str())

	// Fields not previously captured
	v, ok = body.Get("level")
	require.True(t, ok, "level should be in body")
	assert.Equal(t, int64(4), v.Int())

	v, ok = body.Get("version")
	require.True(t, ok, "version should be in body")
	assert.Equal(t, int64(2), v.Int())

	v, ok = body.Get("flags")
	require.True(t, ok, "flags should be in body")
	assert.Equal(t, "test-flags", v.Str())

	v, ok = body.Get("activity_id")
	require.True(t, ok, "activity_id should be in body")
	assert.Equal(t, "activity-id-123", v.Str())

	// Existing body fields
	v, ok = body.Get("opcode")
	require.True(t, ok, "opcode should be in body")
	assert.Equal(t, "Info", v.Str())

	v, ok = body.Get("task")
	require.True(t, ok, "task should be in body")
	assert.Equal(t, "Logon", v.Str())

	v, ok = body.Get("keywords")
	require.True(t, ok, "keywords should be in body")
	assert.Equal(t, "0x8020000000000000", v.Str())

	provider, ok := body.Get("provider")
	require.True(t, ok, "provider should be in body")
	name, ok := provider.Map().Get("name")
	require.True(t, ok)
	assert.Equal(t, "Microsoft-Windows-Security-Auditing", name.Str())
	guid, ok := provider.Map().Get("guid")
	require.True(t, ok)
	assert.Equal(t, "{54849625-5478-4994-A5BA-3E3B0328C30D}", guid.Str())

	eventID, ok := body.Get("event_id")
	require.True(t, ok, "event_id should be in body")
	id, ok := eventID.Map().Get("id")
	require.True(t, ok)
	assert.Equal(t, "4624", id.Str())

	correlation, ok := body.Get("correlation")
	require.True(t, ok, "correlation should be in body")
	actID, ok := correlation.Map().Get("activity_id")
	require.True(t, ok)
	assert.Equal(t, "corr-activity-id", actID.Str())
	relID, ok := correlation.Map().Get("related_activity_id")
	require.True(t, ok)
	assert.Equal(t, "corr-related-id", relID.Str())

	security, ok := body.Get("security")
	require.True(t, ok, "security should be in body")
	sid, ok := security.Map().Get("sid")
	require.True(t, ok)
	assert.Equal(t, "S-1-5-18", sid.Str())

	execution, ok := body.Get("execution")
	require.True(t, ok, "execution should be in body")
	pid, ok := execution.Map().Get("process_id")
	require.True(t, ok)
	assert.Equal(t, "1234", pid.Str())

	_, ok = body.Get("event_data")
	assert.True(t, ok, "event_data should be in body")

	_, ok = body.Get("user_data")
	assert.True(t, ok, "user_data should be in body")

	_, ok = body.Get("extended_data")
	assert.True(t, ok, "extended_data should be in body")
}

func TestParseEventData_OmitsEmptyOptionalFields(t *testing.T) {
	lr := newTestReceiver(t)
	event := &etw.Event{
		Session:   "MinimalSession",
		Timestamp: time.Now(),
		System: etw.EventSystem{
			Level:  4,
			Opcode: "Info",
		},
	}

	record := plog.NewLogRecord()
	lr.parseEventData(event, record)

	body := record.Body().Map()

	// Always-present fields
	_, ok := body.Get("session")
	assert.True(t, ok)
	_, ok = body.Get("level")
	assert.True(t, ok)
	_, ok = body.Get("opcode")
	assert.True(t, ok)
	_, ok = body.Get("correlation") // correlation map always created
	assert.True(t, ok)

	// Optional fields absent when zero/empty
	_, ok = body.Get("channel")
	assert.False(t, ok, "empty channel should be omitted")
	_, ok = body.Get("computer")
	assert.False(t, ok, "empty computer should be omitted")
	_, ok = body.Get("version")
	assert.False(t, ok, "zero version should be omitted")
	_, ok = body.Get("flags")
	assert.False(t, ok, "empty flags should be omitted")
	_, ok = body.Get("activity_id")
	assert.False(t, ok, "empty activity_id should be omitted")
	_, ok = body.Get("task")
	assert.False(t, ok, "empty task should be omitted")
	_, ok = body.Get("provider")
	assert.False(t, ok, "empty provider should be omitted")
	_, ok = body.Get("event_data")
	assert.False(t, ok, "empty event_data should be omitted")
	_, ok = body.Get("keywords")
	assert.False(t, ok, "empty keywords should be omitted")
	_, ok = body.Get("event_id")
	assert.False(t, ok, "empty event_id should be omitted")
	_, ok = body.Get("security")
	assert.False(t, ok, "empty security should be omitted")
	_, ok = body.Get("execution")
	assert.False(t, ok, "zero execution should be omitted")
	_, ok = body.Get("extended_data")
	assert.False(t, ok, "empty extended_data should be omitted")
	_, ok = body.Get("user_data")
	assert.False(t, ok, "empty user_data should be omitted")
}

func TestParseEvent_ResourceAttributesPreserved(t *testing.T) {
	lr := newTestReceiver(t)
	event := newFullEvent()

	logs, err := lr.parseEvent(event)
	require.NoError(t, err)

	resourceLogs := logs.ResourceLogs()
	require.Equal(t, 1, resourceLogs.Len())

	attrs := resourceLogs.At(0).Resource().Attributes()

	v, ok := attrs.Get("session")
	require.True(t, ok, "session resource attribute should be present")
	assert.Equal(t, "TestSession", v.Str())

	v, ok = attrs.Get("provider")
	require.True(t, ok, "provider resource attribute should be present")
	assert.Equal(t, "Microsoft-Windows-Security-Auditing", v.Str())

	v, ok = attrs.Get("provider_guid")
	require.True(t, ok, "provider_guid resource attribute should be present")
	assert.Equal(t, "{54849625-5478-4994-A5BA-3E3B0328C30D}", v.Str())

	v, ok = attrs.Get("computer")
	require.True(t, ok, "computer resource attribute should be present")
	assert.Equal(t, "DESKTOP-ABC123", v.Str())

	v, ok = attrs.Get("channel")
	require.True(t, ok, "channel resource attribute should be present")
	assert.Equal(t, "Security", v.Str())
}

func TestParseEvent_AllFieldsAlsoInBody(t *testing.T) {
	lr := newTestReceiver(t)
	event := newFullEvent()

	logs, err := lr.parseEvent(event)
	require.NoError(t, err)

	record := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	body := record.Body().Map()

	// Fields that used to be resource-attrs-only are now also in the body
	_, ok := body.Get("session")
	assert.True(t, ok, "session should also appear in body")

	_, ok = body.Get("channel")
	assert.True(t, ok, "channel should also appear in body")

	_, ok = body.Get("computer")
	assert.True(t, ok, "computer should also appear in body")
}

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		level    uint8
		expected plog.SeverityNumber
	}{
		{0, plog.SeverityNumberInfo},
		{1, plog.SeverityNumberFatal},
		{2, plog.SeverityNumberError},
		{3, plog.SeverityNumberWarn},
		{4, plog.SeverityNumberInfo},
		{5, plog.SeverityNumberTrace},
		{99, plog.SeverityNumberInfo},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expected, parseSeverity(tc.level))
	}
}
