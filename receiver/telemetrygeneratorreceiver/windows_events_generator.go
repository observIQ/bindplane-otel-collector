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

package telemetrygeneratorreceiver //import "github.com/observiq/bindplane-agent/receiver/telemetrygeneratorreceiver"

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// windowsEventsMetricsGenerator is a generator for Windows Event Log metrics. It generates a sampling of Windows Event Log metrics
// emulating the Windows Event Log receiver: github.com/open-telemetry/opentelemetry-collector-contrib/receiver/windowseventlogreceiver
type windowsEventsMetricsGenerator struct {
	cfg    GeneratorConfig
	logger *zap.Logger
}

func newWindowsEventsGenerator(cfg GeneratorConfig, logger *zap.Logger) logGenerator {
	return &windowsEventsMetricsGenerator{
		cfg:    cfg,
		logger: logger,
	}
}

func (g *windowsEventsMetricsGenerator) generateLogs() plog.Logs {
	return plog.NewLogs()
}
