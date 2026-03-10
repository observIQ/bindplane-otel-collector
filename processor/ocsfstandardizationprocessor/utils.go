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

package ocsfstandardizationprocessor

import (
	v100 "github.com/observiq/bindplane-otel-collector/processor/ocsfstandardizationprocessor/ocsf/v1_0_0"
	v110 "github.com/observiq/bindplane-otel-collector/processor/ocsfstandardizationprocessor/ocsf/v1_1_0"
	v120 "github.com/observiq/bindplane-otel-collector/processor/ocsfstandardizationprocessor/ocsf/v1_2_0"
	v130 "github.com/observiq/bindplane-otel-collector/processor/ocsfstandardizationprocessor/ocsf/v1_3_0"
	v140 "github.com/observiq/bindplane-otel-collector/processor/ocsfstandardizationprocessor/ocsf/v1_4_0"
	v150 "github.com/observiq/bindplane-otel-collector/processor/ocsfstandardizationprocessor/ocsf/v1_5_0"
	v160 "github.com/observiq/bindplane-otel-collector/processor/ocsfstandardizationprocessor/ocsf/v1_6_0"
	v170 "github.com/observiq/bindplane-otel-collector/processor/ocsfstandardizationprocessor/ocsf/v1_7_0"
)

// OCSFSchema is the interface for OCSF schema implementations.
type OCSFSchema interface {
	// ValidateClass validates the body of a class.
	ValidateClass(classUID int, body any) error
	// LookupFieldType returns the expected OCSF type for a field path in a given class.
	LookupFieldType(classUID int, fieldPath string) string
	// ValidateFieldCoverage checks that fieldPaths cover all required fields for the class identified by classUID.
	ValidateFieldCoverage(classUID int, fieldPaths []string) error
}

func getOCSFSchema(ocsfVersion OCSFVersion) OCSFSchema {
	switch ocsfVersion {
	case OCSFVersion1_0_0:
		return &v100.Schema{}
	case OCSFVersion1_1_0:
		return &v110.Schema{}
	case OCSFVersion1_2_0:
		return &v120.Schema{}
	case OCSFVersion1_3_0:
		return &v130.Schema{}
	case OCSFVersion1_4_0:
		return &v140.Schema{}
	case OCSFVersion1_5_0:
		return &v150.Schema{}
	case OCSFVersion1_6_0:
		return &v160.Schema{}
	case OCSFVersion1_7_0:
		return &v170.Schema{}
	default:
		return nil
	}
}
