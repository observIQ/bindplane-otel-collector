package orphandetectorextension

import (
	"context"
	"errors"
	"os"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/extension/extensionhelper"
)

const (
	typeStr         = "orphandetector"
	defaultInterval = 5 * time.Second
)

// Returns a factory for the orphandetector extension
func NewFactory() component.ExtensionFactory {
	return extensionhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		createExtension,
	)
}

func createDefaultConfig() config.Extension {
	return &Config{
		ExtensionSettings: config.NewExtensionSettings(config.NewID(typeStr)),
		Interval:          defaultInterval,
		Ppid:              os.Getppid(),
	}
}

func createExtension(_ context.Context, params component.ExtensionCreateSettings, config config.Extension) (component.Extension, error) {

	if config == nil {
		return nil, errors.New("nil config")
	}

	extConfig := config.(*Config)

	return newParentWatcher(extConfig.Interval, extConfig.DieOnInitParent, extConfig.Ppid, params.Logger), nil
}
