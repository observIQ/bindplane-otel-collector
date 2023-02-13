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

// Package aggregate implements structs that are used to aggregate datapoints.
package aggregate

import (
	"fmt"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

// Aggregate is an interface represents an aggregate of datapoints
type Aggregate interface {
	AddDatapoint(f pmetric.NumberDataPoint)
	SetDatapointValue(pmetric.NumberDataPoint)
}

// AggregationType represents a type of aggregate
type AggregationType string

// Types of aggregates
const (
	MinType   AggregationType = "min"
	MaxType   AggregationType = "max"
	FirstType AggregationType = "first"
	LastType  AggregationType = "last"
	AvgType   AggregationType = "avg"
)

type aggregateConstructor func(pmetric.NumberDataPoint) (Aggregate, error)

var aggregateConstructors = map[AggregationType]aggregateConstructor{
	MinType:   newMinAggregate,
	MaxType:   newMaxAggregate,
	FirstType: newFirstAggregate,
	LastType:  newLastAggregate,
	AvgType:   newAvgAggregate,
}

// New creates a new aggregate of the given type, using the initial datapoint
func (a AggregationType) New(initialVal pmetric.NumberDataPoint) (Aggregate, error) {
	constructor, ok := aggregateConstructors[a]
	if !ok {
		return nil, fmt.Errorf("invalid aggregation type: %s", a)
	}
	return constructor(initialVal)
}

// Valid returns true if this Type is a valid aggregate type, false otherwise
func (a AggregationType) Valid() bool {
	_, ok := aggregateConstructors[a]
	return ok
}
