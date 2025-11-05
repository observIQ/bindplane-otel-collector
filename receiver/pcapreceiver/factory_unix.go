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

//go:build !windows

package pcapreceiver

import (
	"context"

	"github.com/observiq/bindplane-otel-collector/receiver/pcapreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// createLogsReceiver creates a logs receiver for Unix systems (macOS/Linux)
func createLogsReceiver(
	_ context.Context,
	params receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	tb, err := metadata.NewTelemetryBuilder(params.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	receiverCfg := cfg.(*Config)

	params.Logger.Warn("PCAP receiver requires root privileges or tcpdump capabilities (cap_net_raw,cap_net_admin). Ensure the collector has sufficient privileges to capture network packets")

	settings := receivertest.NewNopSettings(metadata.Type)
	receiver, err := newReceiver(settings, receiverCfg, params.Logger, consumer, tb)
	if err != nil {
		return nil, err
	}
	return receiver, nil
}
