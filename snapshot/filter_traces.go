// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package snapshot

import (
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// filterTraces filters the traces by the given query and timestamp.
// The returned payload cannot be assumed to be a copy, so it should not be modified.
func filterTraces(traces ptrace.Traces, searchQuery *string, minimumTimestamp *time.Time) ptrace.Traces {
	// No filters specified, filtered traces are trivially the same as input traces
	if searchQuery == nil && minimumTimestamp == nil {
		return traces
	}

	filteredTraces := ptrace.NewTraces()

	resourceSpans := traces.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		filteredResourceSpans := filterResourceSpans(resourceSpans.At(i), searchQuery, minimumTimestamp)

		// Don't append empty resource traces
		if filteredResourceSpans.ScopeSpans().Len() != 0 {
			filteredResourceSpans.CopyTo(filteredTraces.ResourceSpans().AppendEmpty())
		}
	}

	return filteredTraces
}

func filterResourceSpans(resourceSpan ptrace.ResourceSpans, searchQuery *string, minimumTimestamp *time.Time) ptrace.ResourceSpans {
	filteredResourceSpans := ptrace.NewResourceSpans()

	// Copy old resource to filtered resource
	resource := resourceSpan.Resource()
	resource.CopyTo(filteredResourceSpans.Resource())

	// Apply query to resource
	queryMatchesResource := true // default to true if no query specified
	if searchQuery != nil {
		queryMatchesResource = queryMatchesMap(resource.Attributes(), *searchQuery)
	}

	scopeSpans := resourceSpan.ScopeSpans()
	for i := 0; i < scopeSpans.Len(); i++ {
		filteredScopeSpans := filterScopeSpans(scopeSpans.At(i), queryMatchesResource, searchQuery, minimumTimestamp)

		// Don't append empty scope spans
		if filteredScopeSpans.Spans().Len() != 0 {
			filteredScopeSpans.CopyTo(filteredResourceSpans.ScopeSpans().AppendEmpty())
		}
	}

	return filteredResourceSpans
}

// filterScopeSpans filters out spans that do not match the query and minimumTimestamp, returning a new ptrace.ScopeSpans without the filtered spans.
// queryMatchesResource indicates if the query string matches the resource associated with this ScopeSpans.
func filterScopeSpans(scopeSpans ptrace.ScopeSpans, queryMatchesResource bool, searchQuery *string, minimumTimestamp *time.Time) ptrace.ScopeSpans {
	filteredTraceSpans := ptrace.NewScopeSpans()
	spans := scopeSpans.Spans()
	for i := 0; i < spans.Len(); i++ {
		span := spans.At(i)
		if spanMatches(span, queryMatchesResource, searchQuery, minimumTimestamp) {
			span.CopyTo(filteredTraceSpans.Spans().AppendEmpty())
		}
	}

	return filteredTraceSpans
}

// spanMatches returns true if the query matches either the resource or span, AND the min timestamp.
func spanMatches(s ptrace.Span, queryMatchesResource bool, searchQuery *string, minimumTimestamp *time.Time) bool {
	queryMatchesSpan := true // default to true if no query specified
	// Skip this check if we already know the query matches the resource
	if !queryMatchesResource && searchQuery != nil {
		queryMatchesSpan = spanMatchesQuery(s, *searchQuery)
	}

	timestampMatches := true // default to true if no timestamp specified
	if minimumTimestamp != nil {
		timestampMatches = spanMatchesTimestamp(s, *minimumTimestamp)
	}

	queryMatches := queryMatchesResource || queryMatchesSpan

	return queryMatches && timestampMatches
}

// spanMatchesTimestamp determines if the span came after the provided timestamp
func spanMatchesTimestamp(s ptrace.Span, minTime time.Time) bool {
	return s.EndTimestamp() >= pcommon.NewTimestampFromTime(minTime)
}

// spanMatchesQuery determines if the given span matches the given query string
func spanMatchesQuery(span ptrace.Span, searchQuery string) bool {
	return queryMatchesMap(span.Attributes(), searchQuery) ||
		strings.Contains(span.Name(), searchQuery) ||
		strings.Contains(span.TraceID().String(), searchQuery) ||
		strings.Contains(span.SpanID().String(), searchQuery) ||
		strings.Contains(span.ParentSpanID().String(), searchQuery) ||
		strings.Contains(span.Kind().String(), searchQuery)
}
