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
	"errors"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
)

// Expression is an Expression used to evaluate values.
type Expression struct {
	*vm.Program
}

// Match checks if an expression matches the supplied environment.
func (e *Expression) Match(env map[string]any) (bool, error) {
	matches, err := e.Evaluate(env)
	if err != nil {
		return false, err
	}

	matchesBool, ok := matches.(bool)
	if !ok {
		return false, errors.New("expression did not return a boolean")
	}

	return matchesBool, nil
}

// Evaluate evaluates an expression against the supplied environment.
func (e *Expression) Evaluate(env map[string]any) (any, error) {
	return vm.Run(e.Program, env)
}

// NewExpression creates an expression from a string.
func NewExpression(str string, opts ...expr.Option) (*Expression, error) {
	program, err := expr.Compile(str, opts...)
	if err != nil {
		return nil, err
	}

	return &Expression{program}, nil
}
