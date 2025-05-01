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
	"errors"
	"net/url"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

// FDREvent represents a Crowdstrike FDR event.
type FDREvent struct {
	CID        string    `json:"cid"`
	Timestamp  int64     `json:"timestamp"`
	FileCount  int64     `json:"fileCount"`
	TotalSize  int64     `json:"totalSize"`
	Bucket     string    `json:"bucket"`
	PathPrefix string    `json:"pathPrefix"`
	Files      []FDRFile `json:"files"`
}

// FDRFile represents a file in the FDR event.
type FDRFile struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
}

// NewFDRUnmarshaler provides an implementation of Unmarshaler
// that unmarshals a Crowdstrike FDR event into a slice of Object.
func NewFDRUnmarshaler(set component.TelemetrySettings) Unmarshaler {
	return &FDRUnmarshaler{set: set}
}

// FDRUnmarshaler is an implementation of Unmarshaler that unmarshals a
// Crowdstrike FDR event into a slice of Object.
type FDRUnmarshaler struct {
	set component.TelemetrySettings
}

// Unmarshal unmarshals a Crowdstrike FDR event into a slice of Object.
func (u *FDRUnmarshaler) Unmarshal(body []byte) ([]S3Object, error) {
	event := new(FDREvent)
	err := json.Unmarshal(body, event)
	if err != nil {
		return nil, err
	}

	if len(event.Files) == 0 {
		return nil, ErrNoObjects
	}

	objects := make([]S3Object, len(event.Files))
	for i, file := range event.Files {

		// Reverse SQS URL escaping to get the original S3 object key
		path, err := url.PathUnescape(file.Path)
		if err != nil {
			u.set.Logger.Error("failed to unescape S3 object key",
				zap.String("key", file.Path),
				zap.Error(err),
			)
			continue
		}

		objects[i] = S3Object{
			Bucket: event.Bucket,
			Key:    path,
			Size:   file.Size,
		}
	}
	return objects, nil
}

// NewFDRMarshaler provides an implementation of Marshaler
// that marshals a []Object into a Crowdstrike FDR event.
func NewFDRMarshaler(set component.TelemetrySettings) Marshaler {
	return &fdrMarshaler{set: set}
}

type fdrMarshaler struct {
	set component.TelemetrySettings
}

func (m *fdrMarshaler) Marshal(objects []S3Object) ([][]byte, error) {
	var errs []error
	// The FDR format appears to intend one object per message.
	var bodies [][]byte
	for _, object := range objects {
		fdrEvent := &FDREvent{
			CID:        "test-cid",
			Timestamp:  time.Now().Unix(),
			FileCount:  1,
			TotalSize:  100,
			Bucket:     object.Bucket,
			PathPrefix: "test-path-prefix",
			Files: []FDRFile{
				{
					Path:     object.Key,
					Size:     object.Size,
					Checksum: "test-checksum",
				},
			},
		}
		body, err := json.Marshal(fdrEvent)
		if err != nil {
			errs = append(errs, err)
		} else {
			bodies = append(bodies, body)
		}
	}
	if errs != nil {
		return nil, errors.Join(errs...)
	}
	return bodies, nil
}
