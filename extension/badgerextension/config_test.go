package badgerextension

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name          string
		config        func() *Config
		expectedError error
	}{
		{
			name: "valid config",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.Directory = &DirectoryConfig{
					Path: t.TempDir(),
				}
				return cfg
			},
		},
		{
			name: "bad gc duration",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.Directory = &DirectoryConfig{
					Path: t.TempDir(),
				}
				cfg.BlobGarbageCollection = &BlobGarbageCollectionConfig{
					Interval: 0,
				}
				return cfg
			},
			expectedError: errors.New("blob garbage collection interval must be greater than 0"),
		},
		{
			name: "bad gc discard ratio",
			config: func() *Config {
				cfg := createDefaultConfig().(*Config)
				cfg.Directory = &DirectoryConfig{
					Path: t.TempDir(),
				}
				cfg.BlobGarbageCollection = &BlobGarbageCollectionConfig{
					Interval:     1 * time.Minute,
					DiscardRatio: 1.1,
				}
				return cfg
			},
			expectedError: errors.New("blob garbage collection discard ratio must be between 0 and 1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config().Validate()
			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedError, err)
			}
		})
	}
}
