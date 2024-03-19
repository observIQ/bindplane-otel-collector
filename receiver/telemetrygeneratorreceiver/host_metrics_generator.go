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
	"errors"

	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

// hostMetricsGenerator is a generator for host metrics. It generates a sampling of host metrics
// emulating the Host Metrics receiver: github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver

var hostMetrics = []metric{
	{Name: "system.memory.utilization", ValueMin: 0, ValueMax: 100, Type: "Gauge", Unit: "1", Attributes: map[string]any{"state": "slab_unreclaimable"}},
	{Name: "system.memory.utilization", ValueMin: 0, ValueMax: 100, Type: "Gauge", Unit: "1", Attributes: map[string]any{"state": "cached"}},
	{Name: "system.memory.utilization", ValueMin: 0, ValueMax: 100, Type: "Gauge", Unit: "1", Attributes: map[string]any{"state": "slab_reclaimable"}},
	{Name: "system.memory.utilization", ValueMin: 0, ValueMax: 100, Type: "Gauge", Unit: "1", Attributes: map[string]any{"state": "buffered"}},
	{Name: "system.memory.usage", ValueMin: 100000, ValueMax: 1000000000, Type: "Sum", Unit: "By", Attributes: map[string]any{"state": "buffered"}},
	{Name: "system.memory.usage", ValueMin: 100000, ValueMax: 1000000000, Type: "Sum", Unit: "By", Attributes: map[string]any{"state": "slab_reclaimable"}},
	{Name: "system.memory.usage", ValueMin: 100000, ValueMax: 1000000000, Type: "Sum", Unit: "By", Attributes: map[string]any{"state": "slab_unreclaimable"}},
	{Name: "system.memory.usage", ValueMin: 100000, ValueMax: 1000000000, Type: "Sum", Unit: "By", Attributes: map[string]any{"state": "cached"}},
	{Name: "system.cpu.load_average.1m", ValueMin: 0, ValueMax: 1, Type: "Gauge", Unit: "{thread}"},
	{Name: "system.filesystem.usage", ValueMin: 0, ValueMax: 15616700416, Type: "Sum", Unit: "By", Attributes: map[string]any{"device": "/dev/vda1", "mode": "rw", "mountpoint": "/etc/hosts", "state": "reserved", "type": "ext4"}},
	{Name: "system.filesystem.usage", ValueMin: 0, ValueMax: 15616700416, Type: "Sum", Unit: "By", Attributes: map[string]any{"device": "/dev/vda1", "mode": "rw", "mountpoint": "/etc/hosts", "state": "free", "type": "ext4"}},
	{Name: "system.filesystem.utilization", ValueMin: 0, ValueMax: 1, Type: "Gauge", Unit: "1", Attributes: map[string]any{"device": "/dev/vda1", "mode": "rw", "mountpoint": "/etc/hosts", "state": "free", "type": "ext4"}},
	{Name: "system.network.packets", ValueMin: 0, ValueMax: 1000000, Type: "Sum", Unit: "{packets}", Attributes: map[string]any{"device": "eth0", "direction": "receive"}},
	{Name: "system.network.packets", ValueMin: 0, ValueMax: 1000000, Type: "Sum", Unit: "{packets}", Attributes: map[string]any{"device": "eth0", "direction": "send"}},
	{Name: "system.network.io", ValueMin: 0, ValueMax: 100000000, Type: "Sum", Unit: "By", Attributes: map[string]any{"device": "eth0", "direction": "send"}},
	{Name: "system.network.io", ValueMin: 0, ValueMax: 100000000, Type: "Sum", Unit: "By", Attributes: map[string]any{"device": "eth0", "direction": "receive"}},
	{Name: "system.network.errors", ValueMin: 0, ValueMax: 1000, Type: "Sum", Unit: "{errors}", Attributes: map[string]any{"device": "eth0", "direction": "receive"}},
	{Name: "system.network.errors", ValueMin: 0, ValueMax: 1000, Type: "Sum", Unit: "{errors}", Attributes: map[string]any{"device": "eth0", "direction": "transmit"}},
	{Name: "system.network.dropped", ValueMin: 0, ValueMax: 1000, Type: "Sum", Unit: "{packets}", Attributes: map[string]any{"device": "eth0", "direction": "transmit"}},
	{Name: "system.network.dropped", ValueMin: 0, ValueMax: 1000, Type: "Sum", Unit: "{packets}", Attributes: map[string]any{"device": "eth0", "direction": "receive"}},
	{Name: "system.network.conntrack.max", ValueMin: 65536, ValueMax: 65536, Type: "Sum", Unit: "{entries}"},
	{Name: "system.network.conntrack.count", ValueMin: 8, ValueMax: 64, Type: "Sum", Unit: "{entries}"},
	{Name: "system.network.connections", ValueMin: 0, ValueMax: 64, Type: "Sum", Unit: "{connections}", Attributes: map[string]any{"protocol": "tcp", "state": "ESTABLISHED"}},
	{Name: "system.network.connections", ValueMin: 0, ValueMax: 64, Type: "Sum", Unit: "{connections}", Attributes: map[string]any{"protocol": "tcp", "state": "LISTEN"}},
	{Name: "system.cpu.time", ValueMin: 0, ValueMax: 10000, Type: "Sum", Unit: "s", Attributes: map[string]any{"state": "user", "cpu": "cpu0"}},
	{Name: "system.cpu.time", ValueMin: 0, ValueMax: 10000, Type: "Sum", Unit: "s", Attributes: map[string]any{"state": "system", "cpu": "cpu0"}},
	{Name: "system.cpu.time", ValueMin: 0, ValueMax: 10000, Type: "Sum", Unit: "s", Attributes: map[string]any{"state": "idle", "cpu": "cpu0"}},
	{Name: "system.cpu.time", ValueMin: 0, ValueMax: 10000, Type: "Sum", Unit: "s", Attributes: map[string]any{"state": "interrupt", "cpu": "cpu0"}},
	{Name: "system.cpu.time", ValueMin: 0, ValueMax: 10000, Type: "Sum", Unit: "s", Attributes: map[string]any{"state": "nice", "cpu": "cpu0"}},
	{Name: "system.cpu.time", ValueMin: 0, ValueMax: 10000, Type: "Sum", Unit: "s", Attributes: map[string]any{"state": "softirq", "cpu": "cpu0"}},
	{Name: "system.cpu.time", ValueMin: 0, ValueMax: 10000, Type: "Sum", Unit: "s", Attributes: map[string]any{"state": "steal", "cpu": "cpu0"}},
	{Name: "system.cpu.time", ValueMin: 0, ValueMax: 10000, Type: "Sum", Unit: "s", Attributes: map[string]any{"state": "wait", "cpu": "cpu0"}},
}

func newHostMetricsGenerator(cfg GeneratorConfig, logger *zap.Logger) metricGenerator {
	cfg, err := fillHostMetricsConfig(cfg)
	if err != nil {
		logger.Error("Error filling host metrics config", zap.Error(err))
	}
	g := newMetricsGenerator(cfg, logger)
	return g
}

func fillHostMetricsConfig(cfg GeneratorConfig) (GeneratorConfig, error) {
	metrics := make([]any, 0, len(hostMetrics))

	var errs error
	for _, m := range hostMetrics {
		var metric map[string]any
		err := mapstructure.Decode(m, &metric)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		metrics = append(metrics, metric)
	}
	cfg.AdditionalConfig = map[string]any{"metrics": metrics}
	return cfg, errs
}
