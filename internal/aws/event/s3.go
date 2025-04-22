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

// Package event defines the types of events that can be processed by the extension.
package event // import "github.com/observiq/bindplane-otel-collector/internal/aws/event"

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

// S3Object is a generic representation of an S3 object which needs to be downloaded.
type S3Object struct {
	EventType string
	Bucket    string
	Key       string
	Size      int64
}

// NewS3Unmarshaler provides an implementation of Unmarshaler
// that unmarshals an S3 event into a slice of Object.
func NewS3Unmarshaler(set component.TelemetrySettings) Unmarshaler {
	return &s3Unmarshaler{set: set}
}

type s3Unmarshaler struct {
	set component.TelemetrySettings
}

func (u *s3Unmarshaler) Unmarshal(body []byte) ([]S3Object, error) {
	event := new(events.S3Event)
	err := json.Unmarshal([]byte(body), event)
	if err != nil {
		return nil, err
	}

	if len(event.Records) == 0 {
		return nil, ErrNoObjects
	}

	// Filter records to only include s3:ObjectCreated:* events
	objects := make([]S3Object, 0, len(event.Records))
	for _, record := range event.Records {
		// S3 UI shows the prefix as "s3:ObjectCreated:", but the event name is unmarshaled as "ObjectCreated:"
		if strings.Contains(record.EventName, "ObjectCreated:") {
			objects = append(objects, S3Object{
				EventType: record.EventName,
				Bucket:    record.S3.Bucket.Name,
				Key:       record.S3.Object.Key,
				Size:      record.S3.Object.Size,
			})
			continue
		}
		u.set.Logger.Warn("unexpected event: expected s3:ObjectCreated:*",
			zap.String("event_name", record.EventName),
			zap.String("bucket", record.S3.Bucket.Name),
			zap.String("key", record.S3.Object.Key),
		)
	}
	return objects, nil
}

// NewS3Marshaler provides an implementation of Marshaler
// that marshals a []Object into an S3 event.
func NewS3Marshaler(set component.TelemetrySettings) Marshaler {
	return &s3Marshaler{set: set}
}

type s3Marshaler struct {
	set component.TelemetrySettings
}

func (m *s3Marshaler) Marshal(objects []S3Object) ([][]byte, error) {
	records := make([]events.S3EventRecord, 0, len(objects))
	for _, object := range objects {
		records = append(records, events.S3EventRecord{
			EventName:   object.EventType,
			EventSource: "aws:s3",
			EventTime:   time.Now(),
			S3: events.S3Entity{
				Bucket: events.S3Bucket{
					Name: object.Bucket,
				},
				Object: events.S3Object{
					Key:  object.Key,
					Size: object.Size,
				},
			},
		})
	}
	body, err := json.Marshal(events.S3Event{Records: records})
	if err != nil {
		return nil, err
	}
	return [][]byte{body}, nil
}
