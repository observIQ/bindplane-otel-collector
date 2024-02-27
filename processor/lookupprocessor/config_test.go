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

package lookupprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name string
		cfg  Config
		err  error
	}{
		{
			name: "missing csv",
			cfg:  Config{},
			err:  errMissingCSV,
		},
		{
			name: "missing context",
			cfg:  Config{CSV: "csv"},
			err:  errMissingContext,
		},
		{
			name: "missing field",
			cfg:  Config{CSV: "csv", Context: "body"},
			err:  errMissingField,
		},
		{
			name: "invalid context",
			cfg:  Config{CSV: "csv", Context: "invalid", Field: "field"},
			err:  errInvalidContext,
		},
		{
			name: "valid body context",
			cfg:  Config{CSV: "csv", Context: "body", Field: "field"},
			err:  nil,
		},
		{
			name: "valid attributes context",
			cfg:  Config{CSV: "csv", Context: "attributes", Field: "field"},
			err:  nil,
		},
		{
			name: "valid resource context",
			cfg:  Config{CSV: "csv", Context: "resource.attributes", Field: "field"},
			err:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			switch tc.err {
			case nil:
				require.NoError(t, err)
			default:
				require.Equal(t, tc.err, err)
			}
		})
	}

}
