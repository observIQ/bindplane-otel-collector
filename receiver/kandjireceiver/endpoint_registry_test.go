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

package kandjireceiver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEndpointRegistryInitialized(t *testing.T) {
	// Verify the registry is initialized (not empty)
	require.NotEmpty(t, EndpointRegistry, "EndpointRegistry should not be empty after init")

	// Verify the audit events endpoint exists
	spec, ok := EndpointRegistry[EPAuditEventsList]
	require.True(t, ok, "EPAuditEventsList should be in registry")
	require.NotNil(t, spec)
}

func TestEndpointRegistryAuditEventsSpec(t *testing.T) {
	spec, ok := EndpointRegistry[EPAuditEventsList]
	require.True(t, ok)

	// Verify spec fields
	require.Equal(t, "GET", spec.Method)
	require.Equal(t, "/audit/events", spec.Path)
	require.Equal(t, "List audit log events.", spec.Description)
	require.True(t, spec.SupportsPagination)
	require.IsType(t, AuditEventsResponse{}, spec.ResponseType)

	// Verify params
	require.NotEmpty(t, spec.Params)

	// Check for limit param
	hasLimit := false
	hasSortBy := false
	hasCursor := false
	for _, p := range spec.Params {
		switch p.Name {
		case "limit":
			hasLimit = true
			require.Equal(t, ParamInt, p.Type)
			require.NotNil(t, p.Constraints)
			require.NotNil(t, p.Constraints.MaxInt)
			require.Equal(t, 500, *p.Constraints.MaxInt)
		case "sort_by":
			hasSortBy = true
			require.Equal(t, ParamString, p.Type)
			require.Equal(t, []string{"occurred_at", "-occurred_at"}, p.AllowedVals)
		case "cursor":
			hasCursor = true
			require.Equal(t, ParamString, p.Type)
			require.False(t, p.Required)
		}
	}

	require.True(t, hasLimit, "should have limit param")
	require.True(t, hasSortBy, "should have sort_by param")
	require.True(t, hasCursor, "should have cursor param")
}

func TestEndpointRegistryConstants(t *testing.T) {
	// Verify endpoint constant
	require.Equal(t, KandjiEndpoint("GET /audit/events"), EPAuditEventsList)

	// Verify param types
	require.Equal(t, ParamType(0), ParamString)
	require.Equal(t, ParamType(1), ParamInt)
	require.Equal(t, ParamType(2), ParamBool)
	require.Equal(t, ParamType(3), ParamTime)
	require.Equal(t, ParamType(4), ParamEnum)
	require.Equal(t, ParamType(5), ParamUUID)
	require.Equal(t, ParamType(6), ParamFloat)
}

func TestCursorParam(t *testing.T) {
	require.Equal(t, "cursor", CursorParam.Name)
	require.Equal(t, ParamString, CursorParam.Type)
	require.False(t, CursorParam.Required)
}

