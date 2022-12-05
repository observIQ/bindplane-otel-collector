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

package logcountprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
)

func TestCreateDefaultProcessorConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	require.Equal(t, defaultInterval, cfg.Interval)
	require.Equal(t, defaultMatch, cfg.Match)
	require.Equal(t, defaultMetricName, cfg.MetricName)
	require.Equal(t, defaultMetricUnit, cfg.MetricUnit)
	require.Equal(t, component.NewID(typeStr), cfg.ProcessorSettings.ID())
}

func TestCreateMatchExpr(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Match = "true"
	expr, err := cfg.createMatchExpr()
	require.NoError(t, err)
	require.NotNil(t, expr)

	cfg.Match = "++"
	expr, err = cfg.createMatchExpr()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create match expression")
}

func TestCreateAttrExprs(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.Attributes = map[string]string{"a": "true"}
	expr, err := cfg.createAttrExprs()
	require.NoError(t, err)
	require.NotNil(t, expr)

	cfg.Attributes = map[string]string{"a": "++"}
	expr, err = cfg.createAttrExprs()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create attribute expression for a")
}
