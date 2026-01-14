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

package badgerextension

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"

	"github.com/observiq/bindplane-otel-collector/extension/badgerextension/internal/metadata"
)

// NewFactory creates a new extension factory for the badger storage extension
func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		createExtension,
		metadata.ExtensionStability,
	)
}

// createDefaultConfig creates a default configuration for the badger storage extension
func createDefaultConfig() component.Config {
	return &Config{
		SyncWrites: true,
		Directory: &DirectoryConfig{
			PathPrefix: "badger",
		},
		Memory: &MemoryConfig{
			TableSize:      64 * 1024 * 1024,  // Default: 64MB
			BlockCacheSize: 256 * 1024 * 1024, // Default: 256MB
		},
		BlobGarbageCollection: &BlobGarbageCollectionConfig{
			Interval:     3 * time.Minute,
			DiscardRatio: 0.3,
		},
		// Telemetry is disabled by default
		Telemetry: &TelemetryConfig{
			Enabled:        false,
			UpdateInterval: 1 * time.Minute,
		},
		Compaction: &CompactionConfig{
			NumCompactors:      12,
			NumLevelZeroTables: 2,
		},
	}
}

// createExtension creates a new extension for the badger storage extension
func createExtension(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	oCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type: %T", cfg)
	}
	return newBadgerExtension(set.Logger, oCfg, set.TelemetrySettings, set.ID), nil
}
