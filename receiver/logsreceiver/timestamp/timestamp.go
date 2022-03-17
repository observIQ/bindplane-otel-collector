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

package timestamp

import (
	"strconv"
	"time"

	"go.opentelemetry.io/collector/model/pdata"
)

const msToNs = int64(time.Millisecond / time.Nanosecond)
const sToNs = int64(time.Second / time.Nanosecond)

// UnixTimestampToOtelTimestamp converts an integer representing either seconds or milliseconds since the unix epoch to
//  an pdata.Timestamp (time in nano-seconds since epoch).
func UnixTimestampToOtelTimestamp(v int64) pdata.Timestamp {
	// This is an assumption that if v <= 100 billion, that it's a timestamp in seconds.
	if v <= 100_000_000_000 {
		return pdata.Timestamp(v * sToNs)
	}
	return pdata.Timestamp(v * msToNs)
}

// See https://stackoverflow.com/questions/38596079/how-do-i-parse-an-iso-8601-timestamp-in-golang
// Basically, RFC3339 can fail for some ISO8601 valid date strings.
const iso8601TimestampLayout = "2006-01-02T15:04:05Z0700"

var possibleTimestampFormats = []struct {
	Layout     string
	UsesAbbrev bool
}{
	{iso8601TimestampLayout, false},
	{"2006-01-02T15:04:05.000Z07:00", false}, // RFC3339, but with micro-second precision
	{time.RFC3339, false},
	{time.RFC3339Nano, false},
	{"2006-01-02 15:04:05.999 MST", true}, // Not sure what this format is called, but it's a possible input format.
	{time.RFC822Z, false},
	{time.RFC822, true},
}

// CoerceValToTimestamp attempts to coerce val into a opentelemetry timestamp.
//  returns the coerced timestamp, and a bool indicating if it could be properly coerced or not.
func CoerceValToTimestamp(val pdata.AttributeValue) (pdata.Timestamp, bool) {
	switch val.Type() {
	case pdata.AttributeValueTypeString:
		// Check the timestamp against a few formats. If we don't get a match, we just drop it as
		//  an invalid string.
		v := val.StringVal()

		// First check if it's a stringified unix timestamp
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			// Call coerce timestamp for integer value
			return UnixTimestampToOtelTimestamp(val), true
		}

		for _, format := range possibleTimestampFormats {
			if date, err := time.Parse(format.Layout, v); err == nil {
				if format.UsesAbbrev {
					// Check that the abbreviation was actually found in time.Local
					abbrev, offset := date.Zone()
					if abbrev != "UTC" && abbrev != "GMT" && offset == 0 {
						// Invalid timestamp (not UTC/GMT, but 0 offset.)
						continue
					}
				}
				return pdata.NewTimestampFromTime(date), true
			}
		}
	case pdata.AttributeValueTypeInt:
		return UnixTimestampToOtelTimestamp(val.IntVal()), true
	}

	return 0, false
}
