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

package severity

import "go.opentelemetry.io/collector/model/pdata"

// mappings based on carbon severity parsing
// NOTE: If adding a value to this slice, make sure to put them such that it's in descending order according to the MinIntVal!
var severityInfos = []struct {
	MinIntVal int64
	Name      pdata.SeverityNumber
}{
	{100, pdata.SeverityNumberFATAL4}, // indicates that it is already too late (originally mapped to "catastrophe")
	{90, pdata.SeverityNumberFATAL2},  // indicates that the application is unusable (originally mapped to "emergency")
	{80, pdata.SeverityNumberERROR4},  // indicates that action must be taken immediately (originally mapped to "alert")
	{70, pdata.SeverityNumberERROR3},  // indicates that a problem requires attention immediately (originally mapped to "critical")
	{60, pdata.SeverityNumberERROR},   // indicates that something undesirable has actually happened (originally mapped to "error")
	{50, pdata.SeverityNumberWARN4},   // indicates that someone should look into an issue (originally mapped to "warning")
	{40, pdata.SeverityNumberWARN2},   // indicates that the log should be noticed (originally mapped to "notice")
	{30, pdata.SeverityNumberINFO},    // indicates that the log may be useful for understanding high level details about an application (originally mapped to "info")
	{20, pdata.SeverityNumberDEBUG},   // indicates that the log may be useful for debugging purposes (originally mapped to "debug")
	{10, pdata.SeverityNumberTRACE},   // indicates that the log may be useful for detailed debugging (originally mapped to "trace")
}

// ConvertSeverity converts an integral severity from a stanza log (0 - 100) to otel severity
func ConvertSeverity(severity int64) pdata.SeverityNumber {
	for _, severityInfo := range severityInfos {
		if severity >= severityInfo.MinIntVal {
			return severityInfo.Name
		}
	}
	return pdata.SeverityNumberUNDEFINED
}
