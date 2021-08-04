package observiq

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionhelper"
)

// NewFactory returns a new observiq extension factory
func NewFactory() component.ExtensionFactory {
	return extensionhelper.NewFactory(typeStr, createDefaultConfig, createExtension)
}
