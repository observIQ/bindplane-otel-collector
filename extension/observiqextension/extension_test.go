package observiqextension

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

func TestCreateExtension(t *testing.T) {
	testCases := []struct {
		name          string
		params        component.ExtensionCreateSettings
		config        config.Extension
		expectedError error
	}{
		{
			name:   "With valid config",
			config: &Config{},
		},
		{
			name: "With invalid config",
			config: &struct {
				config.ExtensionSettings `mapstructure:",squash"`
			}{},
			expectedError: errors.New("invalid config type"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			extension, err := createExtension(context.Background(), tc.params, tc.config)
			if tc.expectedError != nil {
				require.Error(t, tc.expectedError, err)
				require.Equal(t, tc.expectedError.Error(), err.Error())
				return
			}

			require.NoError(t, err)

			_, ok := extension.(*Extension)
			require.True(t, ok)
		})
	}
}
