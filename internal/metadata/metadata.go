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

// Package metadata contains the metadata for the AWS S3 Event Receiver
package metadata // import "github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/metadata"

import (
	"go.opentelemetry.io/collector/component"
)

// TypeStr is the string representation of the receiver type
const TypeStr = "awss3event"

// Type is the type of the AWS S3 event receiver
var Type = component.MustNewType(TypeStr)

// MetricsStability is the stability level of the metrics in the receiver
var MetricsStability = component.StabilityLevelDevelopment

// LogsStability is the stability level of the logs in the receiver
var LogsStability = component.StabilityLevelDevelopment

// TracesStability is the stability level of the traces in the receiver
var TracesStability = component.StabilityLevelDevelopment
