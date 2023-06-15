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

package expr

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestExtractAttributes(t *testing.T) {
	attrMap, err := MakeOTTLAttributeMap[ottllog.TransformContext](
		map[string]string{
			"key1":           `body["body_key"]`,
			"attr1":          `attributes["key1"]`,
			"does-not-exist": `attributes["dne"]`,
		},
		componenttest.NewNopTelemetrySettings(),
		NewOTTLLogRecordExpression,
	)
	require.NoError(t, err)

	tCtx := ottllog.NewTransformContext(testLogRecord(t), testScope(t), testResource(t))

	mapOut := attrMap.ExtractAttributes(context.Background(), tCtx)

	require.Equal(t, map[string]any{
		"key1":  "cool-thing",
		"attr1": "val1",
	}, mapOut)

}
