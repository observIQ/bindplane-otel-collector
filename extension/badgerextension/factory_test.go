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

package badgerextension

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCreateDefaultConfig tests that the default configuration is set correctly as they are somewhat intelligently set.
func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	require.NotNil(t, cfg)

	badgerCfg, ok := cfg.(*Config)
	require.True(t, ok, "config should be of type *Config")

	// Verify default values
	require.True(t, badgerCfg.SyncWrites)
	require.NotNil(t, badgerCfg.Directory)
	require.Empty(t, badgerCfg.Directory.Path)
	require.Equal(t, "badger", badgerCfg.Directory.PathPrefix)

	require.NotNil(t, badgerCfg.Memory)
	require.Equal(t, int64(64*1024*1024), badgerCfg.Memory.TableSize)
	require.Equal(t, int64(256*1024*1024), badgerCfg.Memory.BlockCacheSize)

	require.NotNil(t, badgerCfg.BlobGarbageCollection)
	require.Equal(t, 5*time.Minute, badgerCfg.BlobGarbageCollection.Interval)
	require.Equal(t, 0.5, badgerCfg.BlobGarbageCollection.DiscardRatio)

	require.NotNil(t, badgerCfg.Telemetry)
	require.False(t, badgerCfg.Telemetry.Enabled)
	require.Equal(t, 1*time.Minute, badgerCfg.Telemetry.UpdateInterval)
}
