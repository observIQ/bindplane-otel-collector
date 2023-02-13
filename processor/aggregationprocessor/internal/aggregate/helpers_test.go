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

package aggregate

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestGetDatapointValueDouble(t *testing.T) {
	t.Run("with int val", func(t *testing.T) {
		dp := pmetric.NewNumberDataPoint()
		dp.SetIntValue(10)
		require.Equal(t, float64(10), getDatapointValueDouble(dp))
	})

	t.Run("with double val", func(t *testing.T) {
		dp := pmetric.NewNumberDataPoint()
		dp.SetDoubleValue(14.5)
		require.Equal(t, float64(14.5), getDatapointValueDouble(dp))
	})

	t.Run("with no val", func(t *testing.T) {
		dp := pmetric.NewNumberDataPoint()
		require.Equal(t, float64(0), getDatapointValueDouble(dp))
	})
}

func TestGetDatapointValueInt(t *testing.T) {
	t.Run("with int val", func(t *testing.T) {
		dp := pmetric.NewNumberDataPoint()
		dp.SetIntValue(10)
		require.Equal(t, int64(10), getDatapointValueInt(dp))
	})

	t.Run("with double val", func(t *testing.T) {
		dp := pmetric.NewNumberDataPoint()
		dp.SetDoubleValue(14.5)
		require.Equal(t, int64(14), getDatapointValueInt(dp))
	})

	t.Run("with no val", func(t *testing.T) {
		dp := pmetric.NewNumberDataPoint()
		require.Equal(t, int64(0), getDatapointValueInt(dp))
	})
}
