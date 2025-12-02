package badgerextension

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"

	"github.com/observiq/bindplane-otel-collector/extension/badgerextension/internal/metadata"
)

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
		BlobGarbageCollection: &BlobGarbageCollectionConfig{
			Interval:     5 * time.Minute,
			DiscardRatio: 0.5,
		},
	}
}

func createExtension(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	oCfg := cfg.(*Config)

	return newBadgerExtension(set.Logger, oCfg), nil
}
