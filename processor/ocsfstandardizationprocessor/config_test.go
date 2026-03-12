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
var accountChangeFieldMappings = []FieldMapping{
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

func TestConfigValidateAutoAddedFields(t *testing.T) {
	t.Run("category_uid and type_uid are auto-added to field coverage", func(t *testing.T) {
		// This config does NOT explicitly map category_uid or type_uid,
		// but validation should pass because they are auto-added to fieldPaths.
		cfg := Config{
			OCSFVersion: OCSFVersion1_0_0,
			EventMappings: []EventMapping{
				{
					ClassID: 3001,
					FieldMappings: []FieldMapping{
						{From: "body.activity", To: "activity_id"},
						{From: "body.severity", To: "severity_id"},
						{From: "body.time", To: "time"},
						{From: "body.user", To: "user"},
						{From: "body.product", To: "metadata.product"},
					},
				},
			},
		}
		err := cfg.Validate()
		require.NoError(t, err, "validation should pass with category_uid and type_uid auto-added")
	})
}

func TestConfigValidateProfile(t *testing.T) {
	tests := []struct {
		name          string
		eventMappings []EventMapping
		wantErr       string
	}{
		{
			name: "unknown profile",
			eventMappings: []EventMapping{
				{
					ClassID:       3001,
					Profiles:      []string{"test"},
					FieldMappings: accountChangeFieldMappings,
				},
			},
			wantErr: "invalid profile",
		},
		{
			name: "valid profile with required profile fields",
			eventMappings: []EventMapping{
				{
					ClassID:  3001,
					Profiles: []string{"cloud"},
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.cloud", To: "cloud"},
					),
				},
			},
		},
		{
			name: "multiple profiles",
			eventMappings: []EventMapping{
				{
					ClassID:  3001,
					Profiles: []string{"cloud", "datetime"},
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.cloud", To: "cloud"},
					),
				},
			},
		},
		{
			name: "profile with no required fields",
			eventMappings: []EventMapping{
				{
					ClassID:       3001,
					Profiles:      []string{"datetime"},
					FieldMappings: accountChangeFieldMappings,
				},
			},
		},
		{
			name: "profile missing required field",
			eventMappings: []EventMapping{
				{
					ClassID:       3001,
					Profiles:      []string{"cloud"},
					FieldMappings: accountChangeFieldMappings,
				},
			},
			wantErr: "missing required field",
		},
		{
			name: "empty profiles array",
			eventMappings: []EventMapping{
				{
					ClassID:       3001,
					Profiles:      []string{},
					FieldMappings: accountChangeFieldMappings,
				},
			},
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

func TestConfigValidateProfileObjectFields(t *testing.T) {
	// device.type_id is required on the device object, so all device mappings include it.
	tests := []struct {
		name          string
		eventMappings []EventMapping
		wantErr       string
	}{
		{
			name: "host profile with device object fields passes",
			eventMappings: []EventMapping{
				{
					ClassID:  3001,
					Profiles: []string{"host"},
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.device_type", To: "device.type_id"},
						FieldMapping{From: "body.device_ip", To: "device.ip"},
					),
				},
			},
		},
		{
			name: "host and datetime profiles with datetime object field passes",
			eventMappings: []EventMapping{
				{
					ClassID:  3001,
					Profiles: []string{"host", "datetime"},
					FieldMappings: append(accountChangeFieldMappings,
						FieldMapping{From: "body.device_type", To: "device.type_id"},
						FieldMapping{From: "body.device_ip", To: "device.ip"},
					),
				},
			},
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

func TestLookupFieldTypeProfileObjectFields(t *testing.T) {
	// Verify that LookupFieldType resolves profile-specific fields on objects.
	// device.namespace_pid is a container-profile field (type=integer_t).
	// device.created_time_dt is a datetime-profile field (type=datetime_t).
	// container profile on device object was added in v1.1.0
	versionsWithContainer := OCSFVersions[1:] // skip v1.0.0

	for _, version := range versionsWithContainer {
		t.Run(string(version)+"/without_profile_namespace_pid_not_found", func(t *testing.T) {
			schema := getOCSFSchema(version)
			typeName := schema.LookupFieldType(3001, []string{"host"}, "device.namespace_pid")
			require.Empty(t, typeName, "device.namespace_pid should not resolve without container profile")
		})

		t.Run(string(version)+"/with_container_profile_namespace_pid_found", func(t *testing.T) {
			schema := getOCSFSchema(version)
			typeName := schema.LookupFieldType(3001, []string{"host", "container"}, "device.namespace_pid")
			require.Equal(t, "integer", typeName, "device.namespace_pid should resolve to integer with container profile")
		})
	}

	for _, version := range OCSFVersions {
		t.Run(string(version)+"/with_datetime_profile_created_time_dt_found", func(t *testing.T) {
			schema := getOCSFSchema(version)
			typeName := schema.LookupFieldType(3001, []string{"host", "datetime"}, "device.created_time_dt")
			require.Equal(t, "datetime", typeName, "device.created_time_dt should resolve to datetime with datetime profile")
		})

		t.Run(string(version)+"/without_datetime_profile_created_time_dt_not_found", func(t *testing.T) {
			schema := getOCSFSchema(version)
			typeName := schema.LookupFieldType(3001, []string{"host"}, "device.created_time_dt")
			require.Empty(t, typeName, "device.created_time_dt should not resolve without datetime profile")
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
