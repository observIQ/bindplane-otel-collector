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

package azure

import (
	"testing"
	"time"

	azhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/stretchr/testify/require"
)

func TestParseEvent(t *testing.T) {
	var (
		testPartitionID  int16 = 10
		testPartitionKey       = "1"
		testSequenceNum  int64 = 600
		testTime               = time.Now()
		testOffset       int64 = 2000
		testString             = "a test string"
	)

	cases := []struct {
		name           string
		inputRecord    azhub.Event
		expectedRecord *entry.Entry
	}{
		{
			"timestamp-promotion",
			azhub.Event{
				Data: []byte("event hub entry"),
				ID:   "000-555-666",
				SystemProperties: &azhub.SystemProperties{
					SequenceNumber: &testSequenceNum,
					EnqueuedTime:   &testTime,
					Offset:         &testOffset,
				},
			},
			&entry.Entry{
				Timestamp: testTime,
				Body: map[string]interface{}{
					"message": "event hub entry",
					"system_properties": map[string]interface{}{
						"x-opt-sequence-number": &testSequenceNum,
						"x-opt-enqueued-time":   &testTime,
						"x-opt-offset":          &testOffset,
					},
				},
				Resource: map[string]string{
					"event_id": "000-555-666",
				},
			},
		},
		{
			"full",
			azhub.Event{
				Data:         []byte("hello world"),
				ID:           "1111",
				PartitionKey: &testPartitionKey,
				Properties: map[string]interface{}{
					"user": "stanza",
					"id":   1,
					"root": false,
				},
				SystemProperties: &azhub.SystemProperties{
					EnqueuedTime:               &testTime,
					SequenceNumber:             &testSequenceNum,
					Offset:                     &testOffset,
					PartitionID:                &testPartitionID,
					PartitionKey:               &testPartitionKey,
					IoTHubDeviceConnectionID:   &testString,
					IoTHubAuthGenerationID:     &testString,
					IoTHubConnectionAuthMethod: &testString,
					IoTHubConnectionModuleID:   &testString,
					IoTHubEnqueuedTime:         &testTime,
				},
			},
			&entry.Entry{
				Timestamp: testTime,
				Body: map[string]interface{}{
					"message":       "hello world",
					"partition_key": &testPartitionKey,
					"properties": map[string]interface{}{
						"user": "stanza",
						"id":   1,
						"root": false,
					},
					"system_properties": map[string]interface{}{
						"x-opt-sequence-number":                &testSequenceNum,
						"x-opt-enqueued-time":                  &testTime,
						"x-opt-offset":                         &testOffset,
						"x-opt-partition-id":                   &testPartitionID,
						"x-opt-partition-key":                  &testPartitionKey,
						"iothub-connection-device-id":          &testString,
						"iothub-connection-auth-generation-id": &testString,
						"iothub-connection-auth-method":        &testString,
						"iothub-connection-module-id":          &testString,
						"iothub-enqueuedtime":                  &testTime,
					},
				},
				Resource: map[string]string{
					"event_id": "1111",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := entry.New()
			err := ParseEvent(tc.inputRecord, e)
			require.NoError(t, err)
			require.Equal(t, tc.expectedRecord, e)
		})
	}
}

func TestPromoteTime(t *testing.T) {
	enqueuedTime := time.Now()
	ioTHubEnqueuedTime := time.Now().Add(time.Hour * 10)

	cases := []struct {
		name           string
		inputRecord    azhub.Event
		expectedRecord *entry.Entry
	}{
		{
			"enqueuedTime",
			azhub.Event{
				Data: []byte("event hub entry"),
				ID:   "000-555-666",
				SystemProperties: &azhub.SystemProperties{
					EnqueuedTime: &enqueuedTime,
				},
			},
			&entry.Entry{
				Timestamp: enqueuedTime,
				Body: map[string]interface{}{
					"message": "event hub entry",
					"system_properties": map[string]interface{}{
						"x-opt-enqueued-time": &enqueuedTime,
					},
				},
				Resource: map[string]string{
					"event_id": "000-555-666",
				},
			},
		},
		{
			"ioTHubEnqueuedTime",
			azhub.Event{
				Data: []byte("event hub entry"),
				ID:   "000-555-666",
				SystemProperties: &azhub.SystemProperties{
					IoTHubEnqueuedTime: &ioTHubEnqueuedTime,
				},
			},
			&entry.Entry{
				Timestamp: ioTHubEnqueuedTime,
				Body: map[string]interface{}{
					"message": "event hub entry",
					"system_properties": map[string]interface{}{
						"iothub-enqueuedtime": &ioTHubEnqueuedTime,
					},
				},
				Resource: map[string]string{
					"event_id": "000-555-666",
				},
			},
		},
		{
			"both-prefer-enqueuedTime",
			azhub.Event{
				Data: []byte("event hub entry"),
				ID:   "000-555-666",
				SystemProperties: &azhub.SystemProperties{
					EnqueuedTime:       &enqueuedTime,
					IoTHubEnqueuedTime: &ioTHubEnqueuedTime,
				},
			},
			&entry.Entry{
				Timestamp: enqueuedTime,
				Body: map[string]interface{}{
					"message": "event hub entry",
					"system_properties": map[string]interface{}{
						"x-opt-enqueued-time": &enqueuedTime,
						"iothub-enqueuedtime": &ioTHubEnqueuedTime,
					},
				},
				Resource: map[string]string{
					"event_id": "000-555-666",
				},
			},
		},
		{
			"both-prefer-ioTHubEnqueuedTime",
			azhub.Event{
				Data: []byte("event hub entry"),
				ID:   "000-555-666",
				SystemProperties: &azhub.SystemProperties{
					EnqueuedTime:       &time.Time{},
					IoTHubEnqueuedTime: &ioTHubEnqueuedTime,
				},
			},
			&entry.Entry{
				Timestamp: ioTHubEnqueuedTime,
				Body: map[string]interface{}{
					"message": "event hub entry",
					"system_properties": map[string]interface{}{
						"x-opt-enqueued-time": &time.Time{},
						"iothub-enqueuedtime": &ioTHubEnqueuedTime,
					},
				},
				Resource: map[string]string{
					"event_id": "000-555-666",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := entry.New()
			err := ParseEvent(tc.inputRecord, e)
			require.NoError(t, err)
			require.Equal(t, tc.expectedRecord, e)
		})
	}
}
