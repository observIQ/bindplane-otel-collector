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
	"fmt"

	"go.opentelemetry.io/collector/component"
)

// OTTLAttributeMap extracts attributes from telemetry using OTTL expressions. T is an ottl transform context.
type OTTLAttributeMap[T any] struct {
	expressionMap map[string]*OTTLExpression[T]
}

// ExtractAttributes extracts the attributes using the passed in transform context.
func (t OTTLAttributeMap[T]) ExtractAttributes(ctx context.Context, tCtx T) map[string]any {
	attrMap := make(map[string]any, len(t.expressionMap))
	for k, v := range t.expressionMap {
		attrVal, err := v.Execute(ctx, tCtx)
		if err != nil || attrVal == nil {
			continue
		}

		attrMap[k] = attrVal
	}

	return attrMap
}

// MakeOTTLAttributeMap compiles the expressions in the given map of attribute keys to ottl expression strings into an OTTLAttributeMap.
// createFunc is the function for creating the expression, see the NewOTTLxxxExpression functions in this package for functions that may be used here.
func MakeOTTLAttributeMap[T any](m map[string]string, set component.TelemetrySettings, createFunc func(string, component.TelemetrySettings) (*OTTLExpression[T], error)) (*OTTLAttributeMap[T], error) {
	exprMap := make(map[string]*OTTLExpression[T], len(m))

	for k, v := range m {
		expression, err := createFunc(v, set)
		if err != nil {
			return nil, fmt.Errorf("failed to create expr for attribute %q: %w", k, err)
		}

		exprMap[k] = expression
	}

	return &OTTLAttributeMap[T]{
		expressionMap: exprMap,
	}, nil
}
