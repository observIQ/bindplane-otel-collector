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

package logsreceiver

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/observiq/observiq-otel-collector/receiver/logsreceiver/severity"
	"github.com/observiq/observiq-otel-collector/receiver/logsreceiver/timestamp"
	"go.opentelemetry.io/collector/model/pdata"
)

// Transform performs various transformations to the LogRecord to transform it into a friendlier representation
func Transform(le *pdata.LogRecord, idToPipelineConfig map[string]map[string]interface{}) {
	promoteTimestamp(le)
	promoteSeverity(le)
	addPluginInfo(le, idToPipelineConfig)
	convertClient(le)
	convertStringArrays(le)
	convertIntAndFloatFields(le)
}

var timestampFields = []string{
	"@timestamp",
	"timestamp",
	"time",
}

// promoteTimestamp promotes one of the fields from timestampFields to the top-level timestamp, if one is found
func promoteTimestamp(le *pdata.LogRecord) {
	if le.Body().Type() != pdata.AttributeValueTypeMap {
		return
	}

	bodyMap := le.Body().MapVal()

	for _, tsField := range timestampFields {
		if val, ok := bodyMap.Get(tsField); ok {
			if ts, ok := timestamp.CoerceValToTimestamp(val); ok {
				le.SetTimestamp(ts)
				bodyMap.Delete(tsField)
				return
			}
		}
	}
}

// promoteSeverity looks for an integral "severity" field, and converts/promotes it to the top level of the record
func promoteSeverity(le *pdata.LogRecord) {
	if le.Body().Type() != pdata.AttributeValueTypeMap {
		return
	}

	bodyMap := le.Body().MapVal()

	if val, ok := bodyMap.Get("severity"); ok {
		switch val.Type() {
		case pdata.AttributeValueTypeInt:
			v := val.IntVal()
			le.SetSeverityNumber(severity.ConvertSeverity(v))
			bodyMap.Delete("severity")
		case pdata.AttributeValueTypeString:
			v := val.StringVal()
			if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
				le.SetSeverityNumber(severity.ConvertSeverity(intVal))
				bodyMap.Delete("severity")
			}
		}
	}
}

// addPluginInfo adds extra information about the plugin that gathered the entry, if the 'plugin_id' is present on Attributes
func addPluginInfo(le *pdata.LogRecord, idToPipelineConfig map[string]map[string]interface{}) {
	if idToPipelineConfig == nil {
		return
	}

	if pluginID, ok := le.Attributes().Get("plugin_id"); ok {
		if pluginID.Type() == pdata.AttributeValueTypeString {
			pluginIDStr := pluginID.StringVal()
			pluginConf := idToPipelineConfig[pluginIDStr]

			if pluginType, ok := pluginConf["type"]; ok {
				if pluginTypeStr, ok := pluginType.(string); ok {
					le.Attributes().Insert("plugin_type", pdata.NewAttributeValueString(pluginTypeStr))
				}
			}

			if pluginName, ok := pluginConf["name"]; ok {
				if pluginNameStr, ok := pluginName.(string); ok {
					le.Attributes().Insert("plugin_name", pdata.NewAttributeValueString(pluginNameStr))
				}
			}

			if pluginVersion, ok := pluginConf["version"]; ok {
				if pluginVersionStr, ok := pluginVersion.(string); ok {
					le.Attributes().Insert("plugin_version", pdata.NewAttributeValueString(pluginVersionStr))
				}
			}
		}
	}
}

// convertClient transforms the 'client' field on the body into its parts (ip (or address) and port)
func convertClient(le *pdata.LogRecord) {
	if le.Body().Type() != pdata.AttributeValueTypeMap {
		return
	}

	bodyMap := le.Body().MapVal()

	if clientData, ok := bodyMap.Get("client"); ok {
		if clientData.Type() == pdata.AttributeValueTypeString {
			clientDataStr := clientData.StringVal()
			bodyMap.Update("client", parseIPPort(clientDataStr))
		}
	}
}

var arrayFields = []string{
	"http_x_forwarded_for",
	"remote",
	"remote_addr",
	"proxy_protocol_addr",
	"proxy_add_x_forwarded_for",
}

// convertStringArrays converts known array fields that are encoded as strings into an array
func convertStringArrays(le *pdata.LogRecord) {
	if le.Body().Type() != pdata.AttributeValueTypeMap {
		return
	}

	bodyMap := le.Body().MapVal()

	for _, fieldName := range arrayFields {
		if val, ok := bodyMap.Get(fieldName); ok {
			if val.Type() != pdata.AttributeValueTypeString {
				// Skip non-string values
				continue
			}

			strVal := val.StringVal()
			strVal = strings.TrimSpace(strVal)

			if len(strVal) == 0 {
				continue
			}

			if strVal[0] == '[' && strVal[len(strVal)-1] == ']' {
				strVal = strVal[1 : len(strVal)-1]
			}

			strArr := strings.Split(strVal, ",")
			arrAttrib := pdata.NewAttributeValueArray()
			arrOut := arrAttrib.SliceVal()
			arrOut.EnsureCapacity(len(strArr))

			for _, val := range strArr {
				arrOut.AppendEmpty().SetStringVal(strings.TrimSpace(val))
			}

			bodyMap.Update(fieldName, arrAttrib)
		}
	}
}

var intFields = []string{
	"bytes_sent",
	"code",
	"dbid",
	"http_status",
	"level",
	"org_id",
	"pid",
	"process_id",
	"process_log_line",
	"rows_examined",
	"rows_sent",
	"sessionid",
	"size",
	"slow_query_timestamp",
	"status",
	"tid",
}

var floatFields = []string{
	"query_time",
	"lock_time",
}

// convertIntAndFloatFields converts known integer and float fields from strings, replacing the string field with their
//  actual type (int or float)
func convertIntAndFloatFields(le *pdata.LogRecord) {
	if le.Body().Type() != pdata.AttributeValueTypeMap {
		return
	}

	bodyMap := le.Body().MapVal()

	for _, fieldName := range intFields {
		if val, ok := bodyMap.Get(fieldName); ok {
			if val.Type() == pdata.AttributeValueTypeString {
				strVal := val.StringVal()
				if intVal, err := strconv.ParseInt(strVal, 10, 64); err == nil {
					bodyMap.Delete(fieldName)
					bodyMap.InsertInt(fieldName, intVal)
				}
			}
		}
	}

	for _, fieldName := range floatFields {
		if val, ok := bodyMap.Get(fieldName); ok {
			if val.Type() == pdata.AttributeValueTypeString {
				strVal := val.StringVal()
				if floatVal, err := strconv.ParseFloat(strVal, 64); err == nil {
					bodyMap.Delete(fieldName)
					bodyMap.InsertDouble(fieldName, floatVal)
				}
			}
		}
	}
}

var ipPortRegex = regexp.MustCompile(`^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}):(\d+)$`)
var ipRegex = regexp.MustCompile(`^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})$`)
var portRegex = regexp.MustCompile(`(.*):(\d+)$`)

const ipField = "ip"
const portField = "port"
const addressField = "address"

// parseIPPort parses the given string into its ipv4 and port components. If the ip or port cannot be parsed,
//  the field "address" is filled with the unparsed string.
func parseIPPort(s string) pdata.AttributeValue {
	ipPortAttribVal := pdata.NewAttributeValueMap()
	ipPortMap := ipPortAttribVal.MapVal()

	if match := ipPortRegex.FindStringSubmatch(s); match != nil {
		if port, err := strconv.ParseInt(match[2], 10, 64); err == nil {
			ipPortMap.Insert(ipField, pdata.NewAttributeValueString(match[1]))
			ipPortMap.Insert(portField, pdata.NewAttributeValueInt(port))
		} else {
			ipPortMap.Insert(addressField, pdata.NewAttributeValueString(s))
		}
		return ipPortAttribVal
	}

	if match := ipRegex.FindStringSubmatch(s); match != nil {
		ipPortMap.Insert(ipField, pdata.NewAttributeValueString(match[1]))
		return ipPortAttribVal
	}

	if match := portRegex.FindStringSubmatch(s); match != nil {
		if port, err := strconv.ParseInt(match[2], 10, 64); err == nil {
			if len(match[1]) > 0 {
				ipPortMap.Insert(addressField, pdata.NewAttributeValueString(match[1]))
			}
			ipPortMap.Insert(portField, pdata.NewAttributeValueInt(port))
		} else {
			ipPortMap.Insert(addressField, pdata.NewAttributeValueString(s))
		}
		return ipPortAttribVal
	}

	ipPortMap.Insert(addressField, pdata.NewAttributeValueString(s))

	return ipPortAttribVal
}
