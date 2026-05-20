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

package collector

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/featuregate"
	"go.uber.org/zap"
)

func TestSetFeatureFlags(t *testing.T) {
	t.Run("no additional feature gates", func(t *testing.T) {
		require.NoError(t, SetFeatureFlags([]string{}, zap.NewNop()))
	})
	t.Run("override hardcoded feature gates", func(t *testing.T) {
		require.NoError(t, SetFeatureFlags([]string{"-filelog.allowFileDeletion"}, zap.NewNop()))
	})

	t.Run("custom feature gates", func(t *testing.T) {
		testFG, err := featuregate.GlobalRegistry().Register("test.feature", featuregate.StageAlpha)
		require.NoError(t, err)

		require.NoError(t, SetFeatureFlags([]string{"+test.feature"}, zap.NewNop()))
		require.True(t, testFG.IsEnabled())

		require.NoError(t, SetFeatureFlags([]string{"-test.feature"}, zap.NewNop()))
		require.False(t, testFG.IsEnabled())

		require.NoError(t, SetFeatureFlags([]string{"test.feature"}, zap.NewNop()))
		require.True(t, testFG.IsEnabled())
	})
}
