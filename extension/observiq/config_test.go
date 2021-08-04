package observiq

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
)

func TestCreateDefaultConfig(t *testing.T) {
	config := createDefaultConfig()
	require.Equal(t, typeStr, config.ID().String())

	observiqConfig, ok := config.(*Config)
	require.True(t, ok)
	require.Equal(t, endpoint, observiqConfig.Endpoint)
	require.Equal(t, statusInterval, observiqConfig.StatusInterval)
	require.Equal(t, reconnectInterval, observiqConfig.ReconnectInterval)
}

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	factory := NewFactory()
	factories.Extensions[typeStr] = factory
	cfg, err := configtest.LoadConfigAndValidate(path.Join(".", "testdata", "config.yaml"), factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, len(cfg.Extensions), 2)

	expected1 := factory.CreateDefaultConfig().(*Config)
	config1 := cfg.Extensions[config.NewID(typeStr)]
	require.Equal(t, expected1, config1)

	config2 := cfg.Extensions[config.NewIDWithName(typeStr, "2")].(*Config)
	expected2 := factory.CreateDefaultConfig().(*Config)
	expected2.ExtensionSettings = config.NewExtensionSettings(config.NewIDWithName(typeStr, "2"))
	expected2.StatusInterval = time.Second * 5
	expected2.ReconnectInterval = time.Minute * 30
	expected2.Endpoint = "ws://localhost"
	require.Equal(t, expected2, config2)
}
