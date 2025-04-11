// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package regexmatchprocessor_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor"
	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor/internal/matcher"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		cfgMod      func(*regexmatchprocessor.Config)
		expectedErr string
	}{
		{
			desc: "valid config",
			cfgMod: func(cfg *regexmatchprocessor.Config) {
				cfg.Regexes = []matcher.NamedRegex{
					{
						Name:  "test",
						Regex: regexp.MustCompile("test"),
					},
				}
			},
		},
		{
			desc: "config with default value",
			cfgMod: func(cfg *regexmatchprocessor.Config) {
				cfg.Regexes = []matcher.NamedRegex{
					{
						Name:  "test",
						Regex: regexp.MustCompile("test"),
					},
				}
				cfg.DefaultValue = "default"
			},
		},
		{
			desc: "config with empty default value",
			cfgMod: func(cfg *regexmatchprocessor.Config) {
				cfg.Regexes = []matcher.NamedRegex{
					{
						Name:  "test",
						Regex: regexp.MustCompile("test"),
					},
				}
				cfg.DefaultValue = ""
			},
		},
		{
			desc:        "config with no regexes",
			cfgMod:      func(cfg *regexmatchprocessor.Config) {},
			expectedErr: "at least one regex is required",
		},
		{
			desc: "config with empty regex name",
			cfgMod: func(cfg *regexmatchprocessor.Config) {
				cfg.Regexes = []matcher.NamedRegex{
					{
						Name:  "",
						Regex: regexp.MustCompile("test"),
					},
				}
			},
			expectedErr: "regex name cannot be empty",
		},
		{
			desc: "config with duplicate regex name",
			cfgMod: func(cfg *regexmatchprocessor.Config) {
				cfg.Regexes = []matcher.NamedRegex{
					{
						Name:  "test",
						Regex: regexp.MustCompile("test"),
					},
					{
						Name:  "test",
						Regex: regexp.MustCompile("test"),
					},
				}
			},
			expectedErr: "regex name is duplicated: test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			f := regexmatchprocessor.NewFactory()
			cfg := f.CreateDefaultConfig().(*regexmatchprocessor.Config)
			tc.cfgMod(cfg)
			err := cfg.Validate()
			if tc.expectedErr != "" {
				require.ErrorContains(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
