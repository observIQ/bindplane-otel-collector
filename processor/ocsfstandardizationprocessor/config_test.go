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

package ocsfstandardizationprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// accountChangeFieldMappings provides the minimum required field mappings for
// all versions of the AccountChange class (3001). class_uid and metadata.version are auto-added.
var accountChangeFieldMappings []FieldMapping = []FieldMapping{
	{From: "body.activity", To: "activity_id"},
	{From: "body.category", To: "category_uid"},
	{From: "body.severity", To: "severity_id"},
	{From: "body.time", To: "time"},
	{From: "body.type", To: "type_uid"},
	{From: "body.user", To: "user"},
	{From: "body.product", To: "metadata.product"},
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "valid config with all fields",
			cfg: Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						Filter:        "true",
						ClassID:       3001,
						FieldMappings: accountChangeFieldMappings,
					},
				},
			},
		},
		{
			name: "valid config with no event mappings",
			cfg: Config{
				OCSFVersion: OCSFVersion1_0_0,
			},
		},
		{
			name: "valid config with default value and from",
			cfg: Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						ClassID: 3001,
						FieldMappings: []FieldMapping{
							{From: "body.activity", To: "activity_id", Default: 1},
							{From: "body.category", To: "category_uid"},
							{From: "body.severity", To: "severity_id"},
							{From: "body.time", To: "time"},
							{From: "body.type", To: "type_uid"},
							{From: "body.user", To: "user"},
							{From: "body.product", To: "metadata.product"},
						},
					},
				},
			},
		},
		{
			name: "valid config with empty filter",
			cfg: Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						ClassID:       3001,
						FieldMappings: accountChangeFieldMappings,
					},
				},
			},
		},
		{
			name:    "missing OCSF version",
			cfg:     Config{},
			wantErr: "must provide an OCSF version",
		},
		{
			name: "invalid OCSF version",
			cfg: Config{
				OCSFVersion: "2.0.0",
			},
			wantErr: "invalid OCSF version: 2.0.0",
		},
		{
			name: "zero class_id",
			cfg: Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 0,
					},
				},
			},
			wantErr: "event_mappings[0]: class_id must be non-zero",
		},
		{
			name: "invalid filter expression",
			cfg: Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						Filter:  "|||invalid|||",
						ClassID: 1001,
					},
				},
			},
			wantErr: "event_mappings[0]: invalid filter expression",
		},
		{
			name: "field mapping missing to",
			cfg: Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{From: "body.src"},
						},
					},
				},
			},
			wantErr: "event_mappings[0].field_mappings[0]: to is required",
		},
		{
			name: "field mapping missing both from and default",
			cfg: Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{To: "dst_endpoint.ip"},
						},
					},
				},
			},
			wantErr: "event_mappings[0].field_mappings[0]: must have either from or default set",
		},
		{
			name: "invalid from expression",
			cfg: Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{From: "|||invalid|||", To: "message"},
						},
					},
				},
			},
			wantErr: "event_mappings[0].field_mappings[0]: invalid from expression",
		},
		{
			name: "error in second event mapping",
			cfg: Config{
				OCSFVersion: OCSFVersion1_0_0,
				EventMappings: []EventMapping{
					{
						ClassID:       3001,
						FieldMappings: accountChangeFieldMappings,
					},
					{
						ClassID: 0,
					},
				},
			},
			wantErr: "event_mappings[1]: class_id must be non-zero",
		},
		{
			name: "error in second field mapping",
			cfg: Config{
				OCSFVersion: OCSFVersion1_3_0,
				EventMappings: []EventMapping{
					{
						ClassID: 1001,
						FieldMappings: []FieldMapping{
							{From: "body.src", To: "message"},
							{To: "severity"},
						},
					},
				},
			},
			wantErr: "event_mappings[0].field_mappings[1]: must have either from or default set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigValidateFieldCoverage(t *testing.T) {
	tests := []struct {
		name          string
		eventMappings []EventMapping
		wantErr       string
	}{
		{
			name: "unknown class UID",
			eventMappings: []EventMapping{
				{
					ClassID: 9999,
					FieldMappings: []FieldMapping{
						{From: "body.msg", To: "message"},
					},
				},
			},
			wantErr: "OCSF Class 9999 has validation errors",
		},
		{
			name: "missing required fields",
			eventMappings: []EventMapping{
				{
					ClassID: 3001,
					FieldMappings: []FieldMapping{
						{From: "body.msg", To: "message"},
					},
				},
			},
			wantErr: "missing required field",
		},
		{
			name: "all required fields covered",
			eventMappings: []EventMapping{
				{
					ClassID:       3001,
					FieldMappings: accountChangeFieldMappings,
				},
			},
		},
		{
			name: "error in second event mapping",
			eventMappings: []EventMapping{
				{
					ClassID:       3001,
					FieldMappings: accountChangeFieldMappings,
				},
				{
					ClassID: 9999,
					FieldMappings: []FieldMapping{
						{From: "body.msg", To: "message"},
					},
				},
			},
			wantErr: "OCSF Class 9999 has validation errors",
		},
	}

	for _, version := range OCSFVersions {
		for _, tt := range tests {
			t.Run(string(version)+"/"+tt.name, func(t *testing.T) {
				cfg := Config{
					OCSFVersion:   version,
					EventMappings: tt.eventMappings,
				}
				err := cfg.Validate()
				if tt.wantErr != "" {
					require.ErrorContains(t, err, tt.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}
