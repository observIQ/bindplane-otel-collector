package orphandetectorextension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.uber.org/zap"
)

func TestNewFactory(t *testing.T) {
	fact := NewFactory()
	require.NotNil(t, fact)
}

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	require.NotNil(t, cfg)
	require.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateExtension(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	params := component.ExtensionCreateSettings{Logger: zap.NewNop(), BuildInfo: component.DefaultBuildInfo()}

	_, err := createExtension(context.Background(), params, cfg)

	require.NoError(t, err)
}

func TestCreateExtensionNilConfig(t *testing.T) {
	params := component.ExtensionCreateSettings{Logger: zap.NewNop(), BuildInfo: component.DefaultBuildInfo()}

	_, err := createExtension(context.Background(), params, nil)

	require.Error(t, err)
}
