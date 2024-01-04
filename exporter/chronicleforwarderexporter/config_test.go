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

package chronicleforwarderexporter

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confignet"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "Valid syslog config",
			cfg: Config{
				ExportType: exportTypeSyslog,
				Syslog: SyslogConfig{
					NetAddr: confignet.NetAddr{
						Endpoint:  "localhost:514",
						Transport: "tcp",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid syslog config - missing host",
			cfg: Config{
				ExportType: exportTypeSyslog,
				Syslog: SyslogConfig{
					NetAddr: confignet.NetAddr{
						Endpoint:  "",
						Transport: "tcp",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Valid file config",
			cfg: Config{
				ExportType: exportTypeFile,
				File: File{
					Path: "/path/to/file",
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid file config - missing path",
			cfg: Config{
				ExportType: exportTypeFile,
				File:       File{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
