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
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Span specific fields for use in expressions
const (
	SpanKindField          = "trace_kind"
	SpanStatusCodeField    = "trace_status_code"
	SpanStatusMessageField = "trace_status_message"
	SpanDurationField      = "span_duration_ms"
)

var spanKindToString = map[ptrace.SpanKind]string{
	ptrace.SpanKindUnspecified: "unspecified",
	ptrace.SpanKindInternal:    "internal",
	ptrace.SpanKindClient:      "client",
	ptrace.SpanKindServer:      "server",
	ptrace.SpanKindConsumer:    "consumer",
	ptrace.SpanKindProducer:    "producer",
}

var spanStatusCodeToString = map[ptrace.StatusCode]string{
	ptrace.StatusCodeError: "error",
	ptrace.StatusCodeOk:    "ok",
	ptrace.StatusCodeUnset: "unset",
}

// Span is the simplified representation of a metric datapoint.
type Span = map[string]any

func convertToSpan(span ptrace.Span, resource map[string]any) Span {
	return Span{
		ResourceField:          resource,
		AttributesField:        span.Attributes().AsRaw(),
		SpanDurationField:      span.EndTimestamp().AsTime().Sub(span.StartTimestamp().AsTime()).Milliseconds(),
		SpanKindField:          spanKindToString[span.Kind()],
		SpanStatusCodeField:    spanStatusCodeToString[span.Status().Code()],
		SpanStatusMessageField: span.Status().Message(),
	}
}

// SpanResourceGroup represents a ptrace.ResourceSpans as native go types
type SpanResourceGroup struct {
	Resource map[string]any
	Spans    []Span
}

// ConvertToSpanResourceGroups converts a ptrace.Traces into a slice of SpanResourceGroup
func ConvertToSpanResourceGroups(traces ptrace.Traces) []SpanResourceGroup {
	groups := make([]SpanResourceGroup, 0, traces.ResourceSpans().Len())

	for i := 0; i < traces.ResourceSpans().Len(); i++ {
		resourceSpans := traces.ResourceSpans().At(i)
		resource := resourceSpans.Resource().Attributes().AsRaw()
		group := SpanResourceGroup{
			Resource: resource,
			Spans:    make([]Span, 0, resourceSpans.ScopeSpans().Len()),
		}
		for j := 0; j < resourceSpans.ScopeSpans().Len(); j++ {
			spans := resourceSpans.ScopeSpans().At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				group.Spans = append(group.Spans, convertToSpan(span, resource))
			}
		}
		groups = append(groups, group)
	}

	return groups
}
