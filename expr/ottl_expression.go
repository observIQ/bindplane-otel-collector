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

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"go.opentelemetry.io/collector/component"
)

// OTTLExpression evaluates an OTTL expression, returning a resultant value.
type OTTLExpression[T any] struct {
	statement *ottl.Statement[T]
}

// Execute executes the expression with the given context, returning the value of the expression.
func (e OTTLExpression[T]) Execute(ctx context.Context, tCtx T) (any, error) {
	val, _, err := e.statement.Execute(ctx, tCtx)
	return val, err
}

// NewOTTLSpanExpression creates a new expression for spans.
// The expression is wrapped in an editor function, so only Converter functions and target expressions can be used.
func NewOTTLSpanExpression(expression string, set component.TelemetrySettings) (*OTTLExpression[ottlspan.TransformContext], error) {
	// Wrap the expression in the "value" function, since the ottl grammar expects a function first.
	statementStr := fmt.Sprintf("value(%s)", expression)
	statement, err := NewOTTLSpanStatement(statementStr, set)
	if err != nil {
		return nil, err
	}

	return &OTTLExpression[ottlspan.TransformContext]{
		statement: statement,
	}, nil
}

// NewOTTLDatapointExpression creates a new expression for datapoints.
// The expression is wrapped in an editor function, so only Converter functions and target expressions can be used.
func NewOTTLDatapointExpression(expression string, set component.TelemetrySettings) (*OTTLExpression[ottldatapoint.TransformContext], error) {
	// Wrap the expression in the "value" function, since the ottl grammar expects a function first.
	statementStr := fmt.Sprintf("value(%s)", expression)
	statement, err := NewOTTLDatapointStatement(statementStr, set)
	if err != nil {
		return nil, err
	}

	return &OTTLExpression[ottldatapoint.TransformContext]{
		statement: statement,
	}, nil
}

// NewOTTLLogRecordExpression creates a new expression for log records.
// The expression is wrapped in an editor function, so only Converter functions and target expressions can be used.
func NewOTTLLogRecordExpression(expression string, set component.TelemetrySettings) (*OTTLExpression[ottllog.TransformContext], error) {
	// Wrap the expression in the "value" function, since the ottl grammar expects a function first.
	statementStr := fmt.Sprintf("value(%s)", expression)
	statement, err := NewOTTLLogRecordStatement(statementStr, set)
	if err != nil {
		return nil, err
	}

	return &OTTLExpression[ottllog.TransformContext]{
		statement: statement,
	}, nil
}
