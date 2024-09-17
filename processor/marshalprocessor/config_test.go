// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package marshalprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDefaultProcessorConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	require.Equal(t, defaultMarshalTo, cfg.MarshalTo)
	require.Equal(t, defaultKVSeparator, cfg.KVSeparator)
	require.Equal(t, defaultKVPairSeparator, cfg.KVPairSeparator)
	require.Equal(t, defaultMapKVSeparator, cfg.MapKVSeparator)
	require.Equal(t, defaultMapKVPairSeparator, cfg.MapKVPairSeparator)
}

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		cfg         *Config
		expectedErr error
	}{
		{
			desc:        "default",
			cfg:         createDefaultConfig().(*Config),
			expectedErr: nil,
		},
		{
			desc: "JSON",
			cfg: &Config{
				MarshalTo: "JSON",
			},
			expectedErr: nil,
		},
		{
			desc: "XML",
			cfg: &Config{
				MarshalTo: "XML",
			},
			expectedErr: errXMLNotSupported,
		},
		{
			desc: "KV",
			cfg: &Config{
				MarshalTo: "KV",
			},
			expectedErr: nil,
		},
		{
			desc: "JSON lowercase",
			cfg: &Config{
				MarshalTo: "json",
			},
			expectedErr: nil,
		},
		{
			desc: "XML lowercase",
			cfg: &Config{
				MarshalTo: "xml",
			},
			expectedErr: errXMLNotSupported,
		},
		{
			desc: "KV lowercase",
			cfg: &Config{
				MarshalTo: "kv",
			},
			expectedErr: nil,
		},
		{
			desc: "error",
			cfg: &Config{
				MarshalTo: "TOML",
			},
			expectedErr: errInvalidMarshalTo,
		},
		{
			desc: "KV separator fields do not cause a validation error if not marshaling to KV",
			cfg: &Config{
				MarshalTo:          "JSON",
				KVSeparator:        ':',
				KVPairSeparator:    ':',
				MapKVSeparator:     '!',
				MapKVPairSeparator: '!',
			},
			expectedErr: nil,
		},
		{
			desc: "Identical KV separator fields are not allowed",
			cfg: &Config{
				MarshalTo:       "KV",
				KVSeparator:     ':',
				KVPairSeparator: ':',
			},
			expectedErr: errKVSeparatorsEqual,
		},
		{
			desc: "Identical KV separator fields are not allowed with default KVPairSeparator",
			cfg: &Config{
				MarshalTo:       "KV",
				KVSeparator:     ' ',
				KVPairSeparator: ' ',
			},
			expectedErr: errKVSeparatorsEqual,
		},
		{
			desc: "Identical KV separator fields are not allowed with default KVSeparator",
			cfg: &Config{
				MarshalTo:       "KV",
				KVSeparator:     '=',
				KVPairSeparator: '=',
			},
			expectedErr: errKVSeparatorsEqual,
		},
		{
			desc: "Identical Map KV separator fields are not allowed",
			cfg: &Config{
				MarshalTo:          "KV",
				MapKVSeparator:     ':',
				MapKVPairSeparator: ':',
			},
			expectedErr: errMapKVSeparatorsEqual,
		},
		{
			desc: "Identical Map KV separator fields are not allowed with default MapKVPairSeparator",
			cfg: &Config{
				MarshalTo:          "KV",
				MapKVSeparator:     ',',
				MapKVPairSeparator: ',',
			},
			expectedErr: errMapKVSeparatorsEqual,
		},
		{
			desc: "Identical Map KV separator fields are not allowed with default MapKVSeparator",
			cfg: &Config{
				MarshalTo:          "KV",
				MapKVSeparator:     '=',
				MapKVPairSeparator: '=',
			},
			expectedErr: errMapKVSeparatorsEqual,
		},
		{
			desc: "Map KV separators can match their KV counterparts",
			cfg: &Config{
				MarshalTo:          "KV",
				KVSeparator:        ':',
				KVPairSeparator:    ' ',
				MapKVSeparator:     ':',
				MapKVPairSeparator: ' ',
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actualErr := tc.cfg.Validate()
			assert.Equal(t, tc.expectedErr, actualErr)
		})
	}
}
