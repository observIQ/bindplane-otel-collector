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

package topologyprocessor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
)

func TestConfigValidate(t *testing.T) {
	t.Run("Valid config", func(t *testing.T) {
		bindplaneExtensionID := component.MustNewID("bindplane")
		cfg := Config{
			AccountID:          "myacct",
			Configuration:      "myConfig",
			OrganizationID:     "myorg",
			BindplaneExtension: &bindplaneExtensionID,
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("Valid config w/ interval", func(t *testing.T) {
		bindplaneExtensionID := component.MustNewID("bindplane")
		cfg := Config{
			AccountID:          "myacct",
			Configuration:      "myConfig",
			OrganizationID:     "myorg",
			BindplaneExtension: &bindplaneExtensionID,
			Interval:           time.Minute,
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("Valid config no BindplaneExtension", func(t *testing.T) {
		cfg := Config{
			AccountID:      "myacct",
			Configuration:  "myConfig",
			OrganizationID: "myorg",
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("Empty configuration", func(t *testing.T) {
		cfg := Config{
			AccountID:      "myacct",
			OrganizationID: "myorg",
		}
		err := cfg.Validate()
		require.Error(t, err)
	})

	t.Run("Empty AccountID", func(t *testing.T) {
		cfg := Config{
			OrganizationID: "myorg",
			Configuration:  "myconfig",
		}
		err := cfg.Validate()
		require.Error(t, err)
	})

	t.Run("Empty OrganizationID", func(t *testing.T) {
		cfg := Config{
			AccountID:     "myacct",
			Configuration: "myconfig",
		}
		err := cfg.Validate()
		require.Error(t, err)
	})
}
