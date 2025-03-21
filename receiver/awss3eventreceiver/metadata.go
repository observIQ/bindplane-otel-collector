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

package awss3eventreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver"

import (
	"go.opentelemetry.io/collector/component"
)

// The constants for the AWS S3 Event Receiver
const (
	TypeStr = "awss3event"
)

// Type is the component type for the receiver
var Type = component.MustNewType(TypeStr)

// Stability constants for the receiver
var (
	LogsStability = component.StabilityLevelDevelopment
)
