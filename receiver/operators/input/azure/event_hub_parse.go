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
	"reflect"
	"time"

	azhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/mitchellh/mapstructure"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
)

// ParseEvent parses an Azure Event Hub event as an Entry.
func ParseEvent(event azhub.Event, e *entry.Entry) error {
	promoteTime(event, e)
	return parse(event, e)
}

// parse parses an Azure Event Hub event into a map
func parse(event azhub.Event, e *entry.Entry) error {
	m := make(map[string]interface{})

	if len(event.Data) != 0 {
		m["message"] = string(event.Data)
	}

	if event.PartitionKey != nil {
		m["partition_key"] = event.PartitionKey
	}

	if event.Properties != nil {
		m["properties"] = event.Properties
	}

	// promote event.ID to resource.event_id
	if event.ID != "" {
		e.AddResourceKey("event_id", event.ID)
	}

	sysProp := make(map[string]interface{})
	if event.SystemProperties != nil {
		if err := mapstructure.Decode(event.SystemProperties, &sysProp); err != nil {
			return err
		}
		for key := range sysProp {
			if sysProp[key] == nil || reflect.ValueOf(sysProp[key]).IsNil() {
				delete(sysProp, key)
			}
		}
		m["system_properties"] = sysProp
	}

	e.Body = m
	return nil
}

// promoteTime promotes an Azure Event Hub event's timestamp
// EnqueuedTime takes precedence over IoTHubEnqueuedTime
func promoteTime(event azhub.Event, e *entry.Entry) {
	timestamps := []*time.Time{
		event.SystemProperties.EnqueuedTime,
		event.SystemProperties.IoTHubEnqueuedTime,
	}

	for _, t := range timestamps {
		if t != nil && !t.IsZero() {
			e.Timestamp = *t
			return
		}
	}
}
