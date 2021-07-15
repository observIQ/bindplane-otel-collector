package orphandetectorextension

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	factory := NewFactory()
	factories.Extensions[typeStr] = factory
	cfg, err := configtest.LoadConfigAndValidate(path.Join(".", "testdata", "config.yaml"), factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, len(cfg.Extensions), 2)

	// Loaded config should be equal to default config (with APIKey filled in)
	defaultCfg := factory.CreateDefaultConfig()
	r0 := cfg.Extensions[config.NewID(typeStr)]
	require.Equal(t, r0, defaultCfg)

	r1 := cfg.Extensions[config.NewIDWithName(typeStr, "2")].(*Config)
	require.Equal(t, r1, &Config{
		ExtensionSettings: config.NewExtensionSettings(config.NewIDWithName(typeStr, "2")),
		Interval:          500 * time.Millisecond,
		Ppid:              1001,
		DieOnInitParent:   true,
	})
}
