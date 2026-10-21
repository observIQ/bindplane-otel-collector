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

//go:build aixgates

package main

import (
	"testing"

	"github.com/observiq/bindplane-otel-collector/internal/extension/opampconnectionextension/runtime"
	"github.com/stretchr/testify/require"
)

// TestAIXFeatureGates runs in the ocb build of the AIX manifest, where the
// registry holds exactly the AIX components' gates (make check-aix-feature-gates).
func TestAIXFeatureGates(t *testing.T) {
	_, err := components()
	require.NoError(t, err)
	require.NoError(t, runtime.CheckAIXFeatureGates())
}
