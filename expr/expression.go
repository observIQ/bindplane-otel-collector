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

// Package expr provides utilities for evaluating expressions against otel data structures.
package expr

import (
	"errors"
	"fmt"
	"strconv"

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

// MatchRecord checks if an expression matches the supplied record.
func (e *Expression) MatchRecord(record Record) bool {
	matches, err := e.Evaluate(record)
	if err != nil {
		return false
	}

	matchesBool, ok := matches.(bool)
	if !ok {
		return false
	}

	return matchesBool
}

// ExtractFloat extracts a float from the record.
func (e *Expression) ExtractFloat(record Record) (float64, error) {
	value, err := e.Evaluate(record)
	if err != nil {
		return 0, err
	}

	switch value := value.(type) {
	case int:
		return float64(value), nil
	case int32:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case float64:
		return value, nil
	case string:
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f, nil
		}
		return 0, fmt.Errorf("failed to convert string to float: %s", value)
	default:
		return 0, fmt.Errorf("invalid value type: %T", value)
	}
}

// ExtractInt extracts an integer from the record.
func (e *Expression) ExtractInt(record Record) (int64, error) {
	value, err := e.Evaluate(record)
	if err != nil {
		return 0, err
	}

	switch value := value.(type) {
	case int:
		return int64(value), nil
	case int32:
		return int64(value), nil
	case int64:
		return value, nil
	case float64:
		return int64(value), nil
	case string:
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i, nil
		}
		return 0, fmt.Errorf("failed to convert string to int: %s", value)
	default:
		return 0, fmt.Errorf("invalid value type: %T", value)
	}
}

// CreateExpression creates an expression from a string.
func CreateExpression(str string, opts ...expr.Option) (*Expression, error) {
	program, err := expr.Compile(str, opts...)
	if err != nil {
		return nil, err
	}

	return &Expression{program}, nil
}

// CreateBoolExpression creates an expression from a string that returns a boolean.
func CreateBoolExpression(str string) (*Expression, error) {
	return CreateExpression(str, expr.AsBool(), expr.AllowUndefinedVariables())
}

// CreateValueExpression creates an expression from a string that returns a value.
func CreateValueExpression(str string) (*Expression, error) {
	return CreateExpression(str, expr.AllowUndefinedVariables())
}
