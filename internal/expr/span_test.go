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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestConvertToSpanResourceGroups(t *testing.T) {
	now := time.Now().UTC()
	oneMinuteAgo := now.Add(-time.Minute)
	testResource1 := map[string]any{
		"resource": "attributes",
	}
	testResource2 := map[string]any{
		"resource": "attributes2",
	}
	testAttrs := map[string]any{
		"attributes": "attributes",
	}

	traces := ptrace.NewTraces()
	resourceSpans := traces.ResourceSpans().AppendEmpty()
	resourceSpans.Resource().Attributes().FromRaw(testResource1)

	span1 := resourceSpans.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span1.Attributes().FromRaw(testAttrs)
	span1.SetStartTimestamp(pcommon.NewTimestampFromTime(oneMinuteAgo))
	span1.SetEndTimestamp(pcommon.NewTimestampFromTime(now))
	span1.SetKind(ptrace.SpanKindClient)
	span1.Status().SetCode(ptrace.StatusCodeOk)
	span1.Status().SetMessage("Status Message")

	resourceSpans = traces.ResourceSpans().AppendEmpty()
	resourceSpans.Resource().Attributes().FromRaw(testResource2)

	span2 := resourceSpans.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span2.Attributes().FromRaw(testAttrs)
	span2.SetStartTimestamp(pcommon.NewTimestampFromTime(oneMinuteAgo))
	span2.SetEndTimestamp(pcommon.NewTimestampFromTime(now))
	span2.SetKind(ptrace.SpanKindInternal)
	span2.Status().SetCode(ptrace.StatusCodeError)
	span2.Status().SetMessage("Second Status Message")

	groups := ConvertToSpanResourceGroups(traces)

	require.Equal(t, groups, []SpanResourceGroup{
		{
			Resource: testResource1,
			Spans: []Span{
				{
					ResourceField:          testResource1,
					AttributesField:        testAttrs,
					SpanKindField:          "client",
					SpanStatusCodeField:    "ok",
					SpanStatusMessageField: "Status Message",
					SpanDurationField:      time.Minute.Milliseconds(),
				},
			},
		},
		{
			Resource: testResource2,
			Spans: []Span{
				{
					ResourceField:          testResource2,
					AttributesField:        testAttrs,
					SpanKindField:          "internal",
					SpanStatusCodeField:    "error",
					SpanStatusMessageField: "Second Status Message",
					SpanDurationField:      time.Minute.Milliseconds(),
				},
			},
		},
	})
}
