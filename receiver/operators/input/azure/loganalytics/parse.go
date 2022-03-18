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

package loganalytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	azhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/observiq/observiq-otel-collector/receiver/operators/input/azure"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/errors"
	"go.uber.org/zap"
)

// handleBatchedEvents handles an event received by an Event Hub consumer.
func (l *Input) handleBatchedEvents(ctx context.Context, event *azhub.Event) error {
	l.WG.Add(1)
	defer l.WG.Done()

	type record struct {
		Records []map[string]interface{} `json:"records"`
	}

	// Create a "base" event by capturing the batch log records from the event's Data field.
	// If Unmarshalling fails, fallback on handling the event as a single log entry.
	records := record{}
	if err := json.Unmarshal(event.Data, &records); err != nil {
		id := event.ID
		if id == "" {
			event.ID = "unknown"
		}
		l.Warnw(fmt.Sprintf("Failed to parse event '%s' as JSON. Expcted key 'records' in event.Data.", string(event.Data)), zap.Error(err))
		l.handleEvent(ctx, *event, nil)
		return nil
	}
	event.Data = nil

	// Create an entry for each log in the batch, using the origonal event's fields
	// as a starting point for each entry
	wg := sync.WaitGroup{}
	max := 10
	guard := make(chan struct{}, max)
	for i := 0; i < len(records.Records); i++ {
		r := records.Records[i]
		wg.Add(1)
		guard <- struct{}{}
		go func() {
			defer func() {
				wg.Done()
				<-guard
			}()
			l.handleEvent(ctx, *event, r)
		}()
	}
	wg.Wait()
	return nil
}

func (l *Input) handleEvent(ctx context.Context, event azhub.Event, records map[string]interface{}) {
	e, err := l.NewEntry(nil)
	if err != nil {
		l.Errorw("Failed to parse event as an entry", zap.Error(err))
		return
	}

	if err = l.parse(event, records, e); err != nil {
		l.Errorw("Failed to parse event as an entry", zap.Error(err))
		return
	}
	l.Write(ctx, e)
}

// parse returns an entry from an event and set of records
func (l *Input) parse(event azhub.Event, records map[string]interface{}, e *entry.Entry) error {
	// make sure all keys are lower case
	for k, v := range records {
		delete(records, k)
		records[strings.ToLower(k)] = v
	}

	// Add base fields shared among all log records from the event
	err := azure.ParseEvent(event, e)
	if err != nil {
		return err
	}

	// set label azure_log_analytics_table
	records, err = l.setType(e, records)
	if err != nil {
		return err
	}

	if err := l.setTimestamp(e, records); err != nil {
		return err
	}

	// Add remaining records to record.<azure_log_analytics_table> map
	return l.setField(e, e.Attributes["azure_log_analytics_table"], records)
}

// setType sets the label 'azure_log_analytics_table'
func (l *Input) setType(e *entry.Entry, records map[string]interface{}) (map[string]interface{}, error) {
	const typeField = "type"

	for key, value := range records {
		if key == typeField {
			if v, ok := value.(string); ok {
				v = strings.ToLower(v)

				// Set the log table label
				if err := l.setLabel(e, "azure_log_analytics_table", v); err != nil {
					return nil, err
				}

				delete(records, key)
				return records, nil
			}
			return nil, fmt.Errorf("expected '%s' field to be a string", typeField)
		}
	}
	return nil, fmt.Errorf("expected to find field with name '%s'", typeField)
}

// setTimestamp set the entry's timestamp using the timegenerated log analytics field
func (l *Input) setTimestamp(e *entry.Entry, records map[string]interface{}) error {
	for key, value := range records {
		if key == "timegenerated" {
			if v, ok := value.(string); ok {
				t, err := time.Parse("2006-01-02T15:04:05.0000000Z07", v)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("failed to promote timestamp from %s field", key))
				}
				e.Timestamp = t
				return nil
			}
		}
	}
	return nil
}

func (l *Input) setLabel(e *entry.Entry, key string, value interface{}) error {
	r := entry.NewAttributeField(key)
	return r.Set(e, value)
}

func (l *Input) setField(e *entry.Entry, key string, value interface{}) error {
	r := entry.BodyField{
		Keys: []string{key},
	}
	return r.Set(e, value)
}
