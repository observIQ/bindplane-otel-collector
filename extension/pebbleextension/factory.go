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

package pebbleextension

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"

	"github.com/observiq/bindplane-otel-collector/extension/pebbleextension/internal/metadata"
)

// NewFactory creates a new factory for the pebble extension
func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		createExtension,
		metadata.ExtensionStability,
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Directory: &DirectoryConfig{
			PathPrefix: "pebble",
		},
		Sync: true,
		// by default we will manually start a compaction job every 30 minutes with a concurrency of 3
		Compaction: &CompactionConfig{
			Interval:    30 * time.Minute,
			Concurrency: 3,
		},
		CloseTimeout: 10 * time.Second,
	}
}

func createExtension(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	oCfg, ok := cfg.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid config type: %T", cfg)
	}
	return newPebbleExtension(set.Logger, oCfg)
}
