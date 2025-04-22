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

// Marshaler is an interface for marshaling SQS event bodies.
// It accepts multiple objects as input and allows multiple bodies
// as output in order to allow implementations to control how object
// notifications are batched into messages. This corresponds to realworld
// behaviors. e.g. S3 can be configured to batch multiple objects into a single
// message, while Crowdstrike FDR may require a message per object.
type Marshaler interface {
	Marshal([]S3Object) ([][]byte, error)
}
