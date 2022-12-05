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

import "go.opentelemetry.io/collector/pdata/plog"

const (
	// bodyField is the name of the field containing the log body.
	bodyField = "body"

	// resourceField is the name of the field containing the resource attributes.
	resourceField = "resource"

	// attributesField is the name of the field containing the log attributes.
	attributesField = "attributes"

	// severityEnumField is the name of the field containing the log severity enum.
	severityEnumField = "severity_enum"

	// severityNumberField is the name of the field containing the log severity number.
	severityNumberField = "severity_number"
)

// Record is the simplified representation of a log record.
type Record = map[string]any

// convertToRecords converts plog.Logs to a slice of records.
func convertToRecords(logs plog.Logs) []Record {
	records := make([]Record, 0, logs.ResourceLogs().Len())

	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		resourceLogs := logs.ResourceLogs().At(i)
		resource := resourceLogs.Resource().Attributes().AsRaw()
		for j := 0; j < resourceLogs.ScopeLogs().Len(); j++ {
			logs := resourceLogs.ScopeLogs().At(j).LogRecords()
			for k := 0; k < logs.Len(); k++ {
				log := logs.At(k)
				records = append(records, convertToRecord(log, resource))
			}
		}
	}

	return records
}

// convertToRecord converts a log record to a simplified representation.
func convertToRecord(log plog.LogRecord, resource map[string]any) Record {
	return Record{
		resourceField:       resource,
		attributesField:     log.Attributes().AsRaw(),
		bodyField:           log.Body().AsRaw(),
		severityEnumField:   log.SeverityNumber().String(),
		severityNumberField: int32(log.SeverityNumber()),
	}
}
