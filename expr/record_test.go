// Copyright  observIQ, Inc.
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

package expr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestConvertToRecords(t *testing.T) {
	now := time.Now().UTC()
	testResource := map[string]any{
		"resource": "attributes",
	}
	testAttrs := map[string]any{
		"attributes": "attributes",
	}
	testBody := "body"

	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	resourceLogs.Resource().Attributes().FromRaw(testResource)

	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecords := scopeLogs.LogRecords()
	log1 := logRecords.AppendEmpty()
	log1.Body().SetStr(testBody)
	log1.SetSeverityText("info")
	log1.SetTimestamp(pcommon.NewTimestampFromTime(now))
	log1.Attributes().FromRaw(testAttrs)

	records := ConvertToRecords(logs)
	require.Len(t, records, 1)
	require.Equal(t, map[string]any{
		ResourceField:       testResource,
		AttributesField:     testAttrs,
		BodyField:           testBody,
		SeverityEnumField:   "Unspecified",
		SeverityNumberField: int32(0),
		TimestampField:      now,
	}, records[0])
}

func TestConvertToResourceGroups(t *testing.T) {
	now := time.Now().UTC()
	testResource1 := map[string]any{
		"resource": "attributes",
	}
	testResource2 := map[string]any{
		"resource": "attributes2",
	}
	testAttrs := map[string]any{
		"attributes": "attributes",
	}
	testBody := "body"

	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	resourceLogs.Resource().Attributes().FromRaw(testResource1)

	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecords := scopeLogs.LogRecords()
	log1 := logRecords.AppendEmpty()
	log1.Body().SetStr(testBody)
	log1.SetSeverityText("info")
	log1.SetTimestamp(pcommon.NewTimestampFromTime(now))
	log1.Attributes().FromRaw(testAttrs)

	resourceLogs = logs.ResourceLogs().AppendEmpty()
	resourceLogs.Resource().Attributes().FromRaw(testResource2)

	scopeLogs = resourceLogs.ScopeLogs().AppendEmpty()
	logRecords = scopeLogs.LogRecords()
	log2 := logRecords.AppendEmpty()
	log2.Body().SetStr(testBody)
	log2.SetSeverityText("info")
	log2.SetTimestamp(pcommon.NewTimestampFromTime(now))
	log2.Attributes().FromRaw(testAttrs)

	groups := ConvertToResourceGroups(logs)
	require.Len(t, groups, 2)
	require.Equal(t, map[string]any{
		ResourceField:       testResource1,
		AttributesField:     testAttrs,
		BodyField:           testBody,
		SeverityEnumField:   "Unspecified",
		SeverityNumberField: int32(0),
		TimestampField:      now,
	}, groups[0].Records[0])
	require.Equal(t, map[string]any{
		ResourceField:       testResource2,
		AttributesField:     testAttrs,
		BodyField:           testBody,
		SeverityEnumField:   "Unspecified",
		SeverityNumberField: int32(0),
		TimestampField:      now,
	}, groups[1].Records[0])
}
