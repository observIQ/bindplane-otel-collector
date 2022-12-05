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

package logcountprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestConvertToRecords(t *testing.T) {
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
	log1.Attributes().FromRaw(testAttrs)

	records := convertToRecords(logs)
	require.Len(t, records, 1)
	require.Equal(t, map[string]any{
		resourceField:       testResource,
		attributesField:     testAttrs,
		bodyField:           testBody,
		severityEnumField:   "Unspecified",
		severityNumberField: int32(0),
	}, records[0])
}
