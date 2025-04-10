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

package azureloganalyticsexporter

import (
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "Valid configuration",
			config: Config{
				Endpoint:     "https://example.com",
				ClientID:     "client123",
				ClientSecret: "secret123",
				TenantID:     "tenant123",
				RuleID:       "rule123",
				StreamName:   "stream123",
			},
			wantErr: "",
		},
		{
			name: "Missing endpoint",
			config: Config{
				ClientID:     "client123",
				ClientSecret: "secret123",
				TenantID:     "tenant123",
				RuleID:       "rule123",
				StreamName:   "stream123",
			},
			wantErr: "endpoint is required",
		},
		{
			name: "Missing client ID",
			config: Config{
				Endpoint:     "https://example.com",
				ClientSecret: "secret123",
				TenantID:     "tenant123",
				RuleID:       "rule123",
				StreamName:   "stream123",
			},
			wantErr: "client id is required",
		},
		{
			name: "Missing client secret",
			config: Config{
				Endpoint:   "https://example.com",
				ClientID:   "client123",
				TenantID:   "tenant123",
				RuleID:     "rule123",
				StreamName: "stream123",
			},
			wantErr: "client secret is required",
		},
		{
			name: "Missing tenant ID",
			config: Config{
				Endpoint:     "https://example.com",
				ClientID:     "client123",
				ClientSecret: "secret123",
				RuleID:       "rule123",
				StreamName:   "stream123",
			},
			wantErr: "tenant id is required",
		},
		{
			name: "Missing rule ID",
			config: Config{
				Endpoint:     "https://example.com",
				ClientID:     "client123",
				ClientSecret: "secret123",
				TenantID:     "tenant123",
				StreamName:   "stream123",
			},
			wantErr: "rule_id is required",
		},
		{
			name: "Missing stream name",
			config: Config{
				Endpoint:     "https://example.com",
				ClientID:     "client123",
				ClientSecret: "secret123",
				TenantID:     "tenant123",
				RuleID:       "rule123",
			},
			wantErr: "stream_name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() returned unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Validate() expected error but got nil")
				}
				if err.Error() != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}
