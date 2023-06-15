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

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottldatapoint"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
)

// NewOTTLSpanStatement parses the given statement into an ottl.Statement for a span transform context.
func NewOTTLSpanStatement(statementStr string, set component.TelemetrySettings) (*ottl.Statement[ottlspan.TransformContext], error) {
	parser, err := ottlspan.NewParser(functions[ottlspan.TransformContext](), set)
	if err != nil {
		return nil, err
	}

	statement, err := parser.ParseStatement(statementStr)
	if err != nil {
		return nil, err
	}

	return statement, nil
}

// NewOTTLDatapointStatement parses the given statement into an ottl.Statement for a datapoint transform context.
func NewOTTLDatapointStatement(statementStr string, set component.TelemetrySettings) (*ottl.Statement[ottldatapoint.TransformContext], error) {
	parser, err := ottldatapoint.NewParser(functions[ottldatapoint.TransformContext](), set)
	if err != nil {
		return nil, err
	}
	statement, err := parser.ParseStatement(statementStr)
	if err != nil {
		return nil, err
	}

	return statement, nil
}

// NewOTTLLogRecordStatement parses the given statement into an ottl.Statement for a log transform context.
func NewOTTLLogRecordStatement(statementStr string, set component.TelemetrySettings) (*ottl.Statement[ottllog.TransformContext], error) {
	parser, err := ottllog.NewParser(functions[ottllog.TransformContext](), set)
	if err != nil {
		return nil, err
	}

	statement, err := parser.ParseStatement(statementStr)
	if err != nil {
		return nil, err
	}

	return statement, nil
}

// functions is the list of available functions for OTTL statements.
// We include all the converter functions here (functions that do not edit telemetry),
// as well as two custom functions, noop and value.
func functions[T any]() map[string]ottl.Factory[T] {
	return ottl.CreateFactoryMap[T](
		ottlfuncs.NewConcatFactory[T](),
		ottlfuncs.NewConvertCaseFactory[T](),
		ottlfuncs.NewIntFactory[T](),
		ottlfuncs.NewIsMatchFactory[T](),
		ottlfuncs.NewLogFactory[T](),
		ottlfuncs.NewParseJSONFactory[T](),
		ottlfuncs.NewSpanIDFactory[T](),
		ottlfuncs.NewSplitFactory[T](),
		ottlfuncs.NewSubstringFactory[T](),
		ottlfuncs.NewTraceIDFactory[T](),
		ottlfuncs.NewUUIDFactory[T](),

		newNoopFactory[T](),
		newValueFactory[T](),
	)
}

// newNoopFactory returns a factory for the noop function, which does nothing.
// It's used to implement conditions.
func newNoopFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("noop", nil, createNoopFunction[K])
}

func createNoopFunction[K any](_ ottl.FunctionContext, _ ottl.Arguments) (ottl.ExprFunc[K], error) {
	return noopFn[K]()
}

func noopFn[K any]() (ottl.ExprFunc[K], error) {
	return func(context.Context, K) (interface{}, error) {
		return true, nil
	}, nil
}

type valueArguments[K any] struct {
	Target ottl.Getter[K] `ottlarg:"0"`
}

// newValueFactory returns a factory for the value function, which returns the value of it's first argument.
// We need this function because OTTL does not allow direct access to fields on the context, instead
// expecting a function as the first token.
func newValueFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("value", &valueArguments[K]{}, createValueFunction[K])
}

func createValueFunction[K any](_ ottl.FunctionContext, a ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := a.(*valueArguments[K])
	if !ok {
		return nil, fmt.Errorf("valueFactory args must be of type *valueArguments[K]")
	}

	return valueFn[K](args)
}

func valueFn[K any](c *valueArguments[K]) (ottl.ExprFunc[K], error) {
	return func(ctx context.Context, tCtx K) (interface{}, error) {
		return c.Target.Get(ctx, tCtx)
	}, nil
}
