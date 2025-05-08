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

package grpcq

import (
	"context"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

var _ exporterhelper.Request = &Request{}

func NewRequest(request *api.BatchCreateLogsRequest) *Request {
	return &Request{
		request:  request,
		groupKey: computeGroupKey(request.GetBatch()),
	}
}

type Request struct {
	request  *api.BatchCreateLogsRequest
	groupKey string // Hash key for quick compatibility checks
}

func (r *Request) Proto() *api.BatchCreateLogsRequest {
	return r.request
}

// ItemsCount returns a number of basic items in the request where item is the smallest piece of data that can be
// sent. For example, for OTLP exporter, this value represents the number of spans,
// metric data points or log records.
func (r *Request) ItemsCount() int {
	if r.request == nil {
		return 0
	}
	return len(r.request.GetBatch().GetEntries())
}

// MergeSplit is a function that merge and/or splits this request with another one into multiple requests based on the
// configured limit provided in MaxSizeConfig.
// MergeSplit does not split if all fields in MaxSizeConfig are not initialized (zero).
// All the returned requests MUST have a number of items that does not exceed the maximum number of items.
// Size of the last returned request MUST be less or equal than the size of any other returned request.
// The original request MUST not be mutated if error is returned after mutation or if the exporter is
// marked as not mutable. The length of the returned slice MUST not be 0.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func (r *Request) MergeSplit(ctx context.Context, maxSize int, sizer exporterhelper.RequestSizerType, other exporterhelper.Request) ([]exporterhelper.Request, error) {
	if sizer != exporterhelper.RequestSizerTypeBytes {
		return nil, fmt.Errorf("unsupported sizer type: %s", sizer)
	}

	if maxSize <= 0 {
		return []exporterhelper.Request{r}, nil
	}

	switch o := other.(type) {
	case *RequestBundle:
		return append([]exporterhelper.Request{r}, o.expand()...), nil
	case *Request:
		return r.mergeSplitSingle(maxSize, o), nil
	default:
		return nil, fmt.Errorf("invalid request type: expected *RequestBundle or *Request, got %T", other)
	}
}

func (r *Request) mergeSplitSingle(maxSize int, o *Request) []exporterhelper.Request {
	if r.groupKey != o.groupKey {
		return []exporterhelper.Request{r, o}
	}

	defer func() {
		if len(o.request.Batch.Entries) == 0 {
			return
		}
		// Update the StartTime of other since we may have removed the earliest entries from it.
		earliest := o.request.Batch.Entries[0].GetTimestamp()
		for _, entry := range r.request.Batch.Entries {
			if entry.GetTimestamp().AsTime().Before(earliest.AsTime()) {
				earliest = entry.GetTimestamp()
			}
		}
		o.request.Batch.StartTime = earliest
	}()

	// Shift entries from o to r until r is just under maxSize or o is empty.
	// This could be more efficient but the immediate need is to reduce network requests.
	// Inefficiency in this algorithm is paid in the form of compute, which should be relatively cheap
	// compared to the network round trip.
	//
	// Use this even when purely merging two requests because the size of the merged request is unknown
	// unless actually measured.
	for len(o.request.Batch.Entries) > 0 {
		// Append the first entry of o to r. Do not remove it from o until
		// it is known that r is not over the maxSize.
		r.request.Batch.Entries = append(r.request.Batch.Entries, o.request.Batch.Entries[0])

		newSize := globalByteSizer.Sizeof(r)
		if newSize > int64(maxSize) {
			// Exceeded the maxSize. Roll back and return the result.
			r.request.Batch.Entries = r.request.Batch.Entries[:len(r.request.Batch.Entries)-1]
			return []exporterhelper.Request{r, o}
		}

		// Size is fine. Update everything else.
		// Since o.StartTime requires computation, we'll do it once at the end.
		if o.request.Batch.Entries[0].GetTimestamp().AsTime().Before(r.request.Batch.StartTime.AsTime()) {
			r.request.Batch.StartTime = o.request.Batch.Entries[0].GetTimestamp()
		}
		o.request.Batch.Entries = o.request.Batch.Entries[1:]
	}

	if len(o.request.Batch.Entries) == 0 {
		return []exporterhelper.Request{r}
	}
	return []exporterhelper.Request{r, o}
}
