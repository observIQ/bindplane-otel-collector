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

// Package utility provides utility functions for the snowflakeexporter package to consolidate code
package utility // "github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/utility"

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// Array implements driver.Valuer interface allowing arrays to be sent to Snowflake
type Array []any

// Value marshals the underlying array and then casts to a string value so it can be viewed in Snowflake
func (a Array) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

// ConvertAttributesToString converts the pcommon.Map into a JSON string representation
// this is due to a bug/lacking feature with the snowflake driver that prevents maps from being inserted into VARIANT & OBJECT columns
// github issue: https://github.com/snowflakedb/gosnowflake/issues/217
func ConvertAttributesToString(m pcommon.Map, logger *zap.Logger) string {
	bytes, err := json.Marshal(m.AsRaw())
	if err != nil {
		logger.Warn("failed to marshal attribute map", zap.Error(err), zap.Any("map", m))
		return ""
	}
	return string(bytes)
}

// TraceIDToHexOrEmptyString returns a string representation of the TraceID in hex
// Same implementation as "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/traceutil"
func TraceIDToHexOrEmptyString(id pcommon.TraceID) string {
	if id.IsEmpty() {
		return ""
	}
	return hex.EncodeToString(id[:])
}

// SpanIDToHexOrEmptyString returns a string representation of the SpanID in hex
// Same implementation as "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/traceutil"
func SpanIDToHexOrEmptyString(id pcommon.SpanID) string {
	if id.IsEmpty() {
		return ""
	}
	return hex.EncodeToString(id[:])
}

// FlattenExemplars will flatten the given exemplars into slices of the individual fields
func FlattenExemplars(exemplars pmetric.ExemplarSlice) (Array, Array, Array, Array, Array) {
	attributes := Array{}
	timestamps := Array{}
	traceIDs := Array{}
	spanIDs := Array{}
	values := Array{}

	for i := 0; i < exemplars.Len(); i++ {
		e := exemplars.At(i)
		attributes = append(attributes, e.FilteredAttributes().AsRaw())
		timestamps = append(timestamps, e.Timestamp().AsTime())
		traceIDs = append(traceIDs, TraceIDToHexOrEmptyString(e.TraceID()))
		spanIDs = append(spanIDs, SpanIDToHexOrEmptyString(e.SpanID()))

		if e.ValueType() == pmetric.ExemplarValueTypeInt {
			values = append(values, e.IntValue())
		} else {
			values = append(values, e.DoubleValue())
		}
	}

	return attributes, timestamps, traceIDs, spanIDs, values
}
