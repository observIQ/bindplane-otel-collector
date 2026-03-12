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

// Package ocsfstandardizationprocessor provides a processor that can be used to create
// OCSF compliant log bodies from OTEL logs.
package ocsfstandardizationprocessor

import (
	"fmt"
	"slices"

	"github.com/observiq/bindplane-otel-collector/expr"
)

var (
	// OCSFVersion1_0_0 is the OCSF version 1.0.0
	OCSFVersion1_0_0 OCSFVersion = "1.0.0"
	// OCSFVersion1_1_0 is the OCSF version 1.1.0
	OCSFVersion1_1_0 OCSFVersion = "1.1.0"
	// OCSFVersion1_2_0 is the OCSF version 1.2.0
	OCSFVersion1_2_0 OCSFVersion = "1.2.0"
	// OCSFVersion1_3_0 is the OCSF version 1.3.0
	OCSFVersion1_3_0 OCSFVersion = "1.3.0"
	// OCSFVersion1_4_0 is the OCSF version 1.4.0
	OCSFVersion1_4_0 OCSFVersion = "1.4.0"
	// OCSFVersion1_5_0 is the OCSF version 1.5.0
	OCSFVersion1_5_0 OCSFVersion = "1.5.0"
	// OCSFVersion1_6_0 is the OCSF version 1.6.0
	OCSFVersion1_6_0 OCSFVersion = "1.6.0"
	// OCSFVersion1_7_0 is the OCSF version 1.7.0
	OCSFVersion1_7_0 OCSFVersion = "1.7.0"

	// OCSFVersions is the list of supported OCSF versions
	OCSFVersions = []OCSFVersion{
		OCSFVersion1_0_0,
		OCSFVersion1_1_0,
		OCSFVersion1_2_0,
		OCSFVersion1_3_0,
		OCSFVersion1_4_0,
		OCSFVersion1_5_0,
		OCSFVersion1_6_0,
		OCSFVersion1_7_0,
	}
)

// OCSFVersion is the version of the OCSF specification
type OCSFVersion string

// FieldMapping is a mapping of a field from the log body to a field in the OCSF body
type FieldMapping struct {
	From    string `mapstructure:"from"`
	To      string `mapstructure:"to"`
	Default any    `mapstructure:"default,omitempty"`
}

// EventMapping is a mapping of an event to a class ID and a list of field mappings
type EventMapping struct {
	Filter        string         `mapstructure:"filter"`
	ClassID       int            `mapstructure:"class_id"`
	Profiles      []string       `mapstructure:"profiles"`
	FieldMappings []FieldMapping `mapstructure:"field_mappings"`
}

// Config is the configuration for the processor
type Config struct {
	OCSFVersion       OCSFVersion    `mapstructure:"ocsf_version"`
	EventMappings     []EventMapping `mapstructure:"event_mappings"`
	RuntimeValidation *bool          `mapstructure:"runtime_validation"`
}

// Validate validates the processor configuration
func (cfg Config) Validate() error {
	if cfg.OCSFVersion == "" {
		return fmt.Errorf("must provide an OCSF version")
	}
	validVersion := slices.Contains(OCSFVersions, cfg.OCSFVersion)
	if !validVersion {
		return fmt.Errorf("invalid OCSF version: %s", cfg.OCSFVersion)
	}

	for i, em := range cfg.EventMappings {
		if em.ClassID == 0 {
			return fmt.Errorf("event_mappings[%d]: class_id must be non-zero", i)
		}

		if em.Filter != "" {
			if _, err := expr.CreateBoolExpression(em.Filter); err != nil {
				return fmt.Errorf("event_mappings[%d]: invalid filter expression: %w", i, err)
			}
		}

		schema := getOCSFSchema(cfg.OCSFVersion)

		if em.Profiles != nil && len(em.Profiles) > 0 {
			for _, profile := range em.Profiles {
				if err := schema.ValidateProfile(em.ClassID, profile); err != nil {
					return fmt.Errorf("event_mappings[%d]: invalid profile: %w", i, err)
				}
			}
		}

		defaultFieldCount := 4
		fieldPaths := make([]string, len(em.FieldMappings)+defaultFieldCount)
		// We always automatically add the class_uid field and the metadata.version field
		fieldPaths[0] = "class_uid"
		fieldPaths[1] = "metadata.version"
		fieldPaths[2] = "category_uid"
		fieldPaths[3] = "type_uid"
		for j, fm := range em.FieldMappings {
			if fm.To == "" {
				return fmt.Errorf("event_mappings[%d].field_mappings[%d]: to is required", i, j)
			}
			if fm.From == "" && fm.Default == nil {
				return fmt.Errorf("event_mappings[%d].field_mappings[%d]: must have either from or default set", i, j)
			}
			if fm.From != "" {
				_, err := expr.CreateValueExpression(fm.From)
				if err != nil {
					return fmt.Errorf("event_mappings[%d].field_mappings[%d]: invalid from expression: %w", i, j, err)
				}
			}

			fieldPaths[j+defaultFieldCount] = fm.To
		}

		coverageErr := schema.ValidateFieldCoverage(em.ClassID, em.Profiles, fieldPaths)
		if coverageErr != nil {
			return fmt.Errorf("event_mappings[%d]: OCSF Class %d has validation errors\n%w", i, em.ClassID, coverageErr)
		}
	}

	return nil
}
