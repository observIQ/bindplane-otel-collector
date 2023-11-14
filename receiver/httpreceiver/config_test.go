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

package httpreceiver

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtls"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		desc        string
		expectedErr error
		config      Config
	}{
		{
			desc: "pass simple",
			config: Config{
				HTTPServerSettings: confighttp.HTTPServerSettings{
					Endpoint: "localhost:12345",
				},
			},
		},
		{
			desc: "pass complex",
			config: Config{
				Path: "/api/v2/logs",
				HTTPServerSettings: confighttp.HTTPServerSettings{
					Endpoint: "localhost:12345",
					TLSSetting: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							CertFile: "some_cert_file",
							KeyFile:  "some_key_file",
						},
					},
				},
			},
		},
		{
			desc:        "fail no endpoint",
			expectedErr: errNoEndpoint,
			config: Config{
				Path: "/api/v2/logs",
				HTTPServerSettings: confighttp.HTTPServerSettings{
					TLSSetting: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							CertFile: "some_cert_file",
							KeyFile:  "some_key_file",
						},
					},
				},
			},
		},
		{
			desc:        "fail bad endpoint",
			expectedErr: errBadEndpoint,
			config: Config{
				Path: "/api/v2/logs",
				HTTPServerSettings: confighttp.HTTPServerSettings{
					Endpoint: "localhost12345",
					TLSSetting: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							CertFile: "some_cert_file",
							KeyFile:  "some_key_file",
						},
					},
				},
			},
		},
		{
			desc:        "fail tls but no CertFile",
			expectedErr: errNoCert,
			config: Config{
				Path: "/api/v2/logs",
				HTTPServerSettings: confighttp.HTTPServerSettings{
					Endpoint: "localhost:12345",
					TLSSetting: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							KeyFile: "some_key_file",
						},
					},
				},
			},
		},
		{
			desc:        "fail tls but no KeyFile",
			expectedErr: errNoKey,
			config: Config{
				Path: "/api/v2/logs",
				HTTPServerSettings: confighttp.HTTPServerSettings{
					Endpoint: "localhost:12345",
					TLSSetting: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							CertFile: "some_cert_file",
						},
					},
				},
			},
		},
		{
			desc:        "fail bad path",
			expectedErr: errBadPath,
			config: Config{
				Path: "/api , /v2//",
				HTTPServerSettings: confighttp.HTTPServerSettings{
					Endpoint: "localhost:12345",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := component.ValidateConfig(tc.config)
			if tc.expectedErr != nil {
				require.EqualError(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
