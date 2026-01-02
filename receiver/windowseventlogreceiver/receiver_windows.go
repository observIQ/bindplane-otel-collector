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

package windowseventlogreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/windowseventlogreceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventlogreceiver/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/receiver/windowseventlogreceiver/internal/sidcache"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
)

// newFactoryAdapter creates a factory for windowseventlog receiver
func newFactoryAdapter() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
	)
}

// createLogsReceiver creates a logs receiver with SID enrichment support
func createLogsReceiver(
	ctx context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	receiverCfg := cfg.(*WindowsLogConfig)

	// Create SID cache if enabled
	var cache sidcache.Cache
	if receiverCfg.ResolveSIDs.Enabled {
		cacheConfig := sidcache.Config{
			Size: receiverCfg.ResolveSIDs.CacheSize,
			TTL:  receiverCfg.ResolveSIDs.CacheTTL,
		}

		var err error
		cache, err = sidcache.New(cacheConfig)
		if err != nil {
			return nil, err
		}

		set.Logger.Info("SID resolution enabled",
			zap.Int("cache_size", cacheConfig.Size),
			zap.Duration("cache_ttl", cacheConfig.TTL))
	}

	// Wrap the consumer with SID enrichment
	enrichedConsumer := newSIDEnrichingConsumer(nextConsumer, cache, set.Logger)

	// Create the underlying Stanza receiver with the enriched consumer
	stanzaFactory := adapter.NewFactory(receiverType{}, metadata.LogsStability)
	return stanzaFactory.CreateLogs(ctx, set, cfg, enrichedConsumer)
}

// receiverType implements adapter.LogReceiverType
// to create a file tailing receiver
type receiverType struct{}

var _ adapter.LogReceiverType = (*receiverType)(nil)

// Type is the receiver type
func (receiverType) Type() component.Type {
	return metadata.Type
}

// CreateDefaultConfig creates a config with type and version
func (receiverType) CreateDefaultConfig() component.Config {
	return createDefaultConfig()
}

// BaseConfig gets the base config from config, for now
func (receiverType) BaseConfig(cfg component.Config) adapter.BaseConfig {
	return cfg.(*WindowsLogConfig).BaseConfig
}

// InputConfig unmarshals the input operator
func (receiverType) InputConfig(cfg component.Config) operator.Config {
	return operator.NewConfig(&cfg.(*WindowsLogConfig).InputConfig)
}
