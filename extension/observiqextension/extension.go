package observiqextension

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

// Extension is the observiq extension for connecting to the control plane
type Extension struct {
	config *Config
	params component.ExtensionCreateSettings
}

// createExtension creates an observiq extension from the supplied parameters
func createExtension(ctx context.Context, params component.ExtensionCreateSettings, config config.Extension) (component.Extension, error) {
	observiqConfig, ok := config.(*Config)
	if !ok {
		return nil, errors.New("invalid config type")
	}

	observiqExtension := Extension{
		config: observiqConfig,
		params: params,
	}

	return &observiqExtension, nil
}

// Start will start the observiq extension
func (e *Extension) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown will shutdown the observiq extension
func (e *Extension) Shutdown(ctx context.Context) error {
	return nil
}
