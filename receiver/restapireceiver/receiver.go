// Copyright observIQ, Inc.
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

package restapireceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

// restAPILogsReceiver is a receiver that pulls logs from a REST API.
type restAPILogsReceiver struct {
	// TODO: Implement receiver
}

// newRESTAPILogsReceiver creates a new REST API logs receiver.
func newRESTAPILogsReceiver(
	params receiver.Settings,
	cfg *Config,
	consumer consumer.Logs,
) (*restAPILogsReceiver, error) {
	return &restAPILogsReceiver{}, nil
}

// Start starts the receiver.
func (r *restAPILogsReceiver) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown stops the receiver.
func (r *restAPILogsReceiver) Shutdown(_ context.Context) error {
	return nil
}

// restAPIMetricsReceiver is a receiver that pulls metrics from a REST API.
type restAPIMetricsReceiver struct {
	// TODO: Implement receiver
}

// newRESTAPIMetricsReceiver creates a new REST API metrics receiver.
func newRESTAPIMetricsReceiver(
	params receiver.Settings,
	cfg *Config,
	consumer consumer.Metrics,
) (*restAPIMetricsReceiver, error) {
	return &restAPIMetricsReceiver{}, nil
}

// Start starts the receiver.
func (r *restAPIMetricsReceiver) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown stops the receiver.
func (r *restAPIMetricsReceiver) Shutdown(_ context.Context) error {
	return nil
}
