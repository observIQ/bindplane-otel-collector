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

import "errors"

// ErrNoObjects is returned when no objects are found in an event.
var ErrNoObjects = errors.New("no objects found in event")

// Unmarshaler is an interface for unmarshaling SQS events.
type Unmarshaler interface {
	Unmarshal(body []byte) ([]S3Object, error)
}
