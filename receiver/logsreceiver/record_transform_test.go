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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/model/pdata"
)

type mockLogRecord struct {
	Attributes   map[string]interface{}
	Body         interface{}
	Timestamp    pdata.Timestamp
	Severity     pdata.SeverityNumber
	SeverityText string
}

func (m mockLogRecord) LogRecord(t *testing.T) pdata.LogRecord {
	lr := pdata.NewLogRecord()

	attribMap := toAttributeMap(m.Attributes)
	attribMap.MapVal().CopyTo(lr.Attributes())

	if m.Body != nil {
		switch v := m.Body.(type) {
		case map[string]interface{}:
			toAttributeMap(v).CopyTo(lr.Body())
		default:
			require.Fail(t, "Body type not implemented", m)
		}
	} else {
		pdata.NewAttributeValueMap().CopyTo(lr.Body())
	}

	lr.SetTimestamp(m.Timestamp)
	lr.SetSeverityNumber(m.Severity)
	lr.SetSeverityText(m.SeverityText)

	return lr
}

func sortMapKeys(m pdata.AttributeMap) {
	m.Sort()
	m.Range(func(k string, v pdata.AttributeValue) bool {
		if v.Type() == pdata.AttributeValueTypeMap {
			sortMapKeys(v.MapVal())
		}
		return true
	})
}

func TestTransform(t *testing.T) {
	testDate := time.Date(2021, 6, 16, 13, 32, 0, 0, time.UTC)

	testCases := []struct {
		name                string
		lrIn                mockLogRecord
		pluginIDToConfigMap map[string]map[string]interface{}
		lrOut               mockLogRecord
	}{
		{
			name: "Timestamp is promoted (@timestamp)",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"@timestamp":   testDate.Format(time.RFC3339),
					"anotherValue": "val",
				},
			},
			lrOut: mockLogRecord{
				Timestamp: pdata.NewTimestampFromTime(testDate),
				Body: map[string]interface{}{
					"anotherValue": "val",
				},
			},
		},
		{
			name: "Timestamp is promoted (timestamp)",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"timestamp":    testDate.Format(time.RFC3339),
					"anotherValue": "val",
				},
			},
			lrOut: mockLogRecord{
				Timestamp: pdata.NewTimestampFromTime(testDate),
				Body: map[string]interface{}{
					"anotherValue": "val",
				},
			},
		},
		{
			name: "Timestamp is promoted (time)",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"time":         testDate.Format(time.RFC3339),
					"anotherValue": "val",
				},
			},
			lrOut: mockLogRecord{
				Timestamp: pdata.NewTimestampFromTime(testDate),
				Body: map[string]interface{}{
					"anotherValue": "val",
				},
			},
		},
		{
			name: "Severity is promoted (integral)",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"severity":     10,
					"anotherValue": "val",
				},
			},
			lrOut: mockLogRecord{
				Severity: pdata.SeverityNumberTRACE,
				Body: map[string]interface{}{
					"anotherValue": "val",
				},
			},
		},
		{
			name: "Severity is promoted (string)",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"severity":     "10",
					"anotherValue": "val",
				},
			},
			lrOut: mockLogRecord{
				Severity: pdata.SeverityNumberTRACE,
				Body: map[string]interface{}{
					"anotherValue": "val",
				},
			},
		},
		{
			name: "Adds plugin info",
			lrIn: mockLogRecord{
				Attributes: map[string]interface{}{
					"plugin_id": "myid",
				},
			},
			pluginIDToConfigMap: map[string]map[string]interface{}{
				"myid": {
					"id":      "myid",
					"name":    "my_plugin_1",
					"version": "0.0.10",
					"type":    "my_plugin",
				},
			},
			lrOut: mockLogRecord{
				Attributes: map[string]interface{}{
					"plugin_id":      "myid",
					"plugin_name":    "my_plugin_1",
					"plugin_version": "0.0.10",
					"plugin_type":    "my_plugin",
				},
			},
		},
		{
			name: "Skips plugin info if it doesn't exist",
			lrIn: mockLogRecord{
				Attributes: map[string]interface{}{
					"plugin_id": "myid",
				},
			},
			pluginIDToConfigMap: map[string]map[string]interface{}{},
			lrOut: mockLogRecord{
				Attributes: map[string]interface{}{
					"plugin_id": "myid",
				},
			},
		},
		{
			name: "Converts client ip:port",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"client": "192.168.1.1:9001",
				},
			},
			lrOut: mockLogRecord{
				Body: map[string]interface{}{
					"client": map[string]interface{}{
						"ip":   "192.168.1.1",
						"port": 9001,
					},
				},
			},
		},
		{
			name: "Converts client ip",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"client": "192.168.1.1",
				},
			},
			lrOut: mockLogRecord{
				Body: map[string]interface{}{
					"client": map[string]interface{}{
						"ip": "192.168.1.1",
					},
				},
			},
		},
		{
			name: "Converts client port",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"client": "myhostname:9001",
				},
			},
			lrOut: mockLogRecord{
				Body: map[string]interface{}{
					"client": map[string]interface{}{
						"address": "myhostname",
						"port":    9001,
					},
				},
			},
		},
		{
			name: "Converts client just hostname",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"client": "myhostname",
				},
			},
			lrOut: mockLogRecord{
				Body: map[string]interface{}{
					"client": map[string]interface{}{
						"address": "myhostname",
					},
				},
			},
		},
		{
			name: "Converts array of IPs to actual array (1)",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"remote": "[1.1.1.1, 2.2.2.2]",
				},
			},
			lrOut: mockLogRecord{
				Body: map[string]interface{}{
					"remote": []interface{}{"1.1.1.1", "2.2.2.2"},
				},
			},
		},
		{
			name: "Converts array of IPs to actual array (2)",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"remote": " [1.1.1.1, 2.2.2.2]",
				},
			},
			lrOut: mockLogRecord{
				Body: map[string]interface{}{
					"remote": []interface{}{"1.1.1.1", "2.2.2.2"},
				},
			},
		},
		{
			name: "Converts array of IPs to actual array (3)",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"remote": "[1.1.1.1 2.2.2.2]",
				},
			},
			lrOut: mockLogRecord{
				Body: map[string]interface{}{
					"remote": []interface{}{"1.1.1.1 2.2.2.2"},
				},
			},
		},
		{
			name: "Converts array of IPs to actual array (4)",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"remote": "1.1.1.1, 2.2.2.2",
				},
			},
			lrOut: mockLogRecord{
				Body: map[string]interface{}{
					"remote": []interface{}{"1.1.1.1", "2.2.2.2"},
				},
			},
		},
		{
			name: "Doesn't panic if expected IP array field is empty",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"remote": "",
				},
			},
			lrOut: mockLogRecord{
				Body: map[string]interface{}{
					"remote": "",
				},
			},
		},
		{
			name: "Converts known int fields",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"bytes_sent":    "1024",
					"rows_examined": "455",
				},
			},
			lrOut: mockLogRecord{
				Body: map[string]interface{}{
					"bytes_sent":    1024,
					"rows_examined": 455,
				},
			},
		},
		{
			name: "Converts known float fields",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"query_time": "1.75",
					"lock_time":  "3.5",
				},
			},
			lrOut: mockLogRecord{
				Body: map[string]interface{}{
					"query_time": 1.75,
					"lock_time":  3.5,
				},
			},
		},
		{
			name: "Doesn't convert int fields if not integral",
			lrIn: mockLogRecord{
				Body: map[string]interface{}{
					"bytes_sent":    "sent",
					"rows_examined": "examined",
				},
			},
			lrOut: mockLogRecord{
				Body: map[string]interface{}{
					"bytes_sent":    "sent",
					"rows_examined": "examined",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			lrIn := testCase.lrIn.LogRecord(t)
			Transform(&lrIn, testCase.pluginIDToConfigMap)
			lrOut := testCase.lrOut.LogRecord(t)

			sortMapKeys(lrIn.Attributes())
			sortMapKeys(lrOut.Attributes())

			if lrIn.Body().Type() == pdata.AttributeValueTypeMap {
				sortMapKeys(lrIn.Body().MapVal())
			}

			if lrOut.Body().Type() == pdata.AttributeValueTypeMap {
				sortMapKeys(lrOut.Body().MapVal())
			}

			require.Equal(t, lrOut, lrIn)
		})
	}
}
