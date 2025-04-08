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

func TestConfigValidate(t *testing.T) {
	cfg := createTestConfig()
	err := cfg.Validate()
	require.NoError(t, err)
}
