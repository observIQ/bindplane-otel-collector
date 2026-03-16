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

import "go.opentelemetry.io/collector/pdata/plog"

const (
	// BodyField is the name of the field containing the log body.
	BodyField = "body"

	// ResourceField is the name of the field containing the resource attributes.
	ResourceField = "resource"

	// AttributesField is the name of the field containing the log attributes.
	AttributesField = "attributes"

	// SeverityEnumField is the name of the field containing the log severity enum.
	SeverityEnumField = "severity_enum"

	// SeverityNumberField is the name of the field containing the log severity number.
	SeverityNumberField = "severity_number"

	// TimestampField is the name of the field containing the log timestamp.
	TimestampField = "timestamp"
)

// Record is the simplified representation of a log record.
type Record = map[string]any

// ConvertToRecords converts plog.Logs to a slice of records.
func ConvertToRecords(logs plog.Logs) []Record {
	records := make([]Record, 0, logs.ResourceLogs().Len())

	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		resourceLogs := logs.ResourceLogs().At(i)
		resource := resourceLogs.Resource().Attributes().AsRaw()
		for j := 0; j < resourceLogs.ScopeLogs().Len(); j++ {
			logs := resourceLogs.ScopeLogs().At(j).LogRecords()
			for k := 0; k < logs.Len(); k++ {
				log := logs.At(k)
				records = append(records, ConvertToRecord(log, resource))
			}
		}
	}

	return records
}

// ConvertToRecord converts a log record to a simplified representation.
func ConvertToRecord(log plog.LogRecord, resource map[string]any) Record {
	return Record{
		ResourceField:       resource,
		AttributesField:     log.Attributes().AsRaw(),
		BodyField:           log.Body().AsRaw(),
		SeverityEnumField:   log.SeverityNumber().String(),
		SeverityNumberField: int32(log.SeverityNumber()),
		TimestampField:      log.Timestamp().AsTime(),
	}
}

// ResourceGroup is a group of records with the same resource attributes.
type ResourceGroup struct {
	Resource map[string]any
	Records  []Record
}

// ConvertToResourceGroups converts plog.Logs to a slice of resource groups.
func ConvertToResourceGroups(logs plog.Logs) []ResourceGroup {
	groups := make([]ResourceGroup, 0, logs.ResourceLogs().Len())

	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		resourceLogs := logs.ResourceLogs().At(i)
		resource := resourceLogs.Resource().Attributes().AsRaw()
		group := ResourceGroup{
			Resource: resource,
			Records:  make([]Record, 0, resourceLogs.ScopeLogs().Len()),
		}
		for j := 0; j < resourceLogs.ScopeLogs().Len(); j++ {
			logs := resourceLogs.ScopeLogs().At(j).LogRecords()
			for k := 0; k < logs.Len(); k++ {
				log := logs.At(k)
				group.Records = append(group.Records, ConvertToRecord(log, resource))
			}
		}
		groups = append(groups, group)
	}

	return groups
}
