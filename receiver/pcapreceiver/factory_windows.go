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

//go:build windows

package pcapreceiver

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/observiq/bindplane-otel-collector/receiver/pcapreceiver/internal/metadata"
)

// createLogsReceiver creates a logs receiver for Windows
func createLogsReceiver(
	_ context.Context,
	params receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {

	settings := receivertest.NewNopSettings(metadata.Type)
	tb, err := metadata.NewTelemetryBuilder(params.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	receiverCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type")
	}

	if err := receiverCfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	params.Logger.Warn("PCAP receiver on Windows requires Npcap. Ensure Npcap is installed and the interface is available.")
	receiver, err := newReceiver(settings, receiverCfg, params.Logger, consumer, tb)
	if err != nil {
		return nil, err
	}
	return receiver, nil
}
