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

package expr

import (
	"fmt"
)

// ExpressionMap is a map of expressions.
type ExpressionMap struct {
	expressions map[string]*Expression
}

// Extract extracts a value map from the record using the expression map.
func (e *ExpressionMap) Extract(record Record) map[string]any {
	results := map[string]any{}
	for key, expression := range e.expressions {
		value, err := expression.Evaluate(record)
		if err != nil || value == nil {
			continue
		}
		results[key] = value
	}
	return results
}

// CreateExpressionMap creates an expression map from a string map.
func CreateExpressionMap(strMap map[string]string) (*ExpressionMap, error) {
	expressionMap := &ExpressionMap{
		expressions: map[string]*Expression{},
	}

	for key, value := range strMap {
		expr, err := CreateValueExpression(value)
		if err != nil {
			return nil, fmt.Errorf("failed to create expression for %s: %w", key, err)
		}
		expressionMap.expressions[key] = expr
	}

	return expressionMap, nil
}
