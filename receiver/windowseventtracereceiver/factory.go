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

//go:build windows

package windowseventtracereceiver

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/metadata"
)

const typeName = "windowseventtrace"

// NewFactory creates a new receiver factory
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
	)
}

func createLogsReceiver(ctx context.Context, params receiver.Settings, cfg component.Config, nextConsumer consumer.Logs) (receiver.Logs, error) {
	cfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("not a valid microsoft event trace config")
	}

	lr, err := newLogsReceiver(cfg.(*Config), nextConsumer, params.Logger)
	if err != nil {
		return nil, err
	}
	return lr, nil
}
