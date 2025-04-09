//go:build windows

package windowseventtracereceiver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func createTestConfig() *Config {
	return &Config{
		SessionName: "TestSession",
		BufferSize:  64,
		Providers: []Provider{
			{Name: "TestProvider", Level: LevelInformational},
		},
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "providers cannot be empty")
}
