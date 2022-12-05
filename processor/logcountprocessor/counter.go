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

import "encoding/json"

// LogCounter tracks the number of times a set of resource and attribute dimensions have been seen.
type LogCounter struct {
	resources map[string]*ResourceCounter
}

// NewLogCounter creates a new LogCounter.
func NewLogCounter() *LogCounter {
	return &LogCounter{
		resources: make(map[string]*ResourceCounter),
	}
}

// Add increments the counter with the supplied dimensions.
func (l *LogCounter) Add(resource, attributes map[string]any) {
	key := getDimensionKey(resource)
	if _, ok := l.resources[key]; !ok {
		l.resources[key] = NewResourceCounter(resource)
	}

	l.resources[key].Add(attributes)
}

// Reset resets the counter.
func (l *LogCounter) Reset() {
	l.resources = make(map[string]*ResourceCounter)
}

// ResourceCounter dimensions the counter by resource.
type ResourceCounter struct {
	values     map[string]any
	attributes map[string]*AttributeCounter
}

// NewResourceCounter creates a new ResourceCounter.
func NewResourceCounter(values map[string]any) *ResourceCounter {
	return &ResourceCounter{
		values:     values,
		attributes: map[string]*AttributeCounter{},
	}
}

// Add increments the counter with the supplied dimensions.
func (r *ResourceCounter) Add(attributes map[string]any) {
	key := getDimensionKey(attributes)
	if _, ok := r.attributes[key]; !ok {
		r.attributes[key] = NewAttributeCounter(attributes)
	}

	r.attributes[key].Add()
}

// AttributeCounter dimensions the counter by attributes.
type AttributeCounter struct {
	values map[string]any
	count  int
}

// NewAttributeCounter creates a new AttributeCounter.
func NewAttributeCounter(values map[string]any) *AttributeCounter {
	return &AttributeCounter{
		values: values,
	}
}

// Add increments the counter.
func (a *AttributeCounter) Add() {
	a.count++
}

// getDimensionKey returns a unique key for the dimension.
func getDimensionKey(dimension map[string]any) string {
	dimensionJSON, _ := json.Marshal(dimension)
	return string(dimensionJSON)
}
