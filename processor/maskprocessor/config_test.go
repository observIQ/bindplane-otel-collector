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

package maskprocessor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		cfg         Config
		expectedErr error
	}{
		{
			desc: "Invalid rule",
			cfg: Config{
				Rules: map[string]string{
					"invalid": `\K`,
				},
			},
			expectedErr: errors.New("rule 'invalid' does not compile as valid regex"),
		},
		{
			desc:        "No rules",
			cfg:         Config{},
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
