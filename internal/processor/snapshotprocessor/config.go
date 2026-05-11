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

// Package snapshotprocessor collects metrics, traces, and logs for
package snapshotprocessor

import "go.opentelemetry.io/collector/component"

// Config is the configuration for the processor.
//
// The processor always serves snapshots over the legacy report-manager
// path (Bindplane sends a snapshotConfig and the report manager POSTs
// the payload). If OpAMP is set, it additionally registers a custom
// capability on the named extension and serves snapshots in response
// to OpAMP custom messages over the same connection.
type Config struct {
	// Enabled controls whether snapshots are collected.
	Enabled bool `mapstructure:"enabled"`

	// OpAMP, if set, is the component ID of the opamp_connection
	// extension whose CustomCapabilityRegistry the processor will
	// register the snapshot capability with. Leave unset to run in
	// report-manager-only mode.
	OpAMP component.ID `mapstructure:"opamp"`
}

// Validate validates the processor configuration. Both an unset OpAMP
// (legacy mode) and a set OpAMP (legacy + custom-message mode) are
// valid; nothing else is checked.
func (cfg Config) Validate() error {
	return nil
}
