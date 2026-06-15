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

//go:generate mdatagen metadata.yaml

// Package gcspubsubeventreceiver implements a receiver that consumes GCS object event
// notifications from a Google Cloud Pub/Sub subscription and processes the objects
// containing log data.
//
// The receiver uses synchronous Pub/Sub pull to receive GCS event notifications. When an
// OBJECT_FINALIZE event is received, the receiver downloads the GCS object and processes
// it as log data.
package gcspubsubeventreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver"
