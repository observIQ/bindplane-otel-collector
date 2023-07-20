// Copyright observIQ, Inc.
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

package m365receiver // import "github.com/observiq/bindplane-agent/receiver/m365receiver"

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		desc         string
		errExpected  bool
		errText      string
		tenantID     string
		clientID     string
		clientSecret string
	}{
		{
			desc:         "expected case, correct",
			errExpected:  false,
			tenantID:     "b243371b-5514-4951-840f-152d525b6555",
			clientID:     "0b110766-c91d-4fe5-bac5-14a16ce3c351",
			clientSecret: "FZu8Q~gFWqIo_vo8tE36TYsymPGk9VfWvTlS7b-I",
		},
		{
			desc:         "correct, randomized guid's",
			errExpected:  false,
			tenantID:     "e1da6cba-0230-41c5-99d6-575011a48b55",
			clientID:     "6aed100f-5463-4d7f-b02f-1c89de31acb8",
			clientSecret: "FZu8Q~gFWqIo_vo8tE36TYsymPGVfWvTlS7b-I",
		},
		{
			desc:         "incorrect clientID: empty",
			errExpected:  true,
			errText:      "missing client_id; required",
			tenantID:     "b243371b-5514-4951-840f-152d525b6555",
			clientID:     "",
			clientSecret: "FZu8Q~gFWqIo_vo8tE36TYsymPGk9VfWvTlS7b-I",
		},
		{
			desc:         "incorrect clientID: too short",
			errExpected:  true,
			errText:      "client_id is invalid; must be a GUID",
			tenantID:     "b243371b-5514-4951-840f-152d525b6555",
			clientID:     "foo-test",
			clientSecret: "FZu8Q~gFWqIo_vo8tE36TYsymPGk9VfWvTlS7b-I",
		},
		{
			desc:         "incorrect tenantID: empty",
			errExpected:  true,
			errText:      "missing tenant_id; required",
			tenantID:     "",
			clientID:     "0b110766-c91d-4fe5-bac5-14a16ce3c351",
			clientSecret: "FZu8Q~gFWqIo_vo8tE36TYsymPGk9VfWvTlS7b-I",
		},
		{
			desc:         "incorrect tenantID: too short",
			errExpected:  true,
			errText:      "tenant_id is invalid; must be a GUID",
			tenantID:     "b243371b",
			clientID:     "0b110766-c91d-4fe5-bac5-14a16ce3c351",
			clientSecret: "FZu8Q~gFWqIo_vo8tE36TYsymPGk9VfWvTlS7b-I",
		},
		{
			desc:         "incorrect clientID: empty",
			errExpected:  true,
			errText:      "missing client_secret; required",
			tenantID:     "b243371b-5514-4951-840f-152d525b6555",
			clientID:     "0b110766-c91d-4fe5-bac5-14a16ce3c351",
			clientSecret: "",
		},
		{
			desc:         "incorrect clientID: invalid character",
			errExpected:  true,
			errText:      "client_secret is invalid; does not follow correct structure",
			tenantID:     "b243371b-5514-4951-840f-152d525b6555",
			clientID:     "0b110766-c91d-4fe5-bac5-14a16ce3c351",
			clientSecret: "FZu8Q~gFWqIo_vo8tE36T=ymPGk9VfWvTlS7b-I",
		},
		{
			desc:         "incorrect clientID: too long",
			errExpected:  true,
			errText:      "client_secret is invalid; does not follow correct structure",
			tenantID:     "b243371b-5514-4951-840f-152d525b6555",
			clientID:     "0b110766-c91d-4fe5-bac5-14a16ce3c351",
			clientSecret: "FZu8Q~gFWqIo_vo8tE36TsdfsfsdfsdfsdfymPGk9VfWvTlS7b-I",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := NewFactory().CreateDefaultConfig().(*Config)
			cfg.TenantID = tc.tenantID
			cfg.ClientID = tc.clientID
			cfg.ClientSecret = tc.clientSecret

			err := component.ValidateConfig(cfg)

			if tc.errExpected {
				require.EqualError(t, err, tc.errText)
				return
			}

			require.NoError(t, err)
		})
	}
}
