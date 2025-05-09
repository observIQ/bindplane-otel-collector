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
	"google.golang.org/protobuf/types/known/timestamppb"
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

	// TODO pass in maxSize based on cfg.BatchRequestSizeLimitGRPC
	maxSize64 := int64(maxSize)
	if maxSize64 == 0 {
		maxSize64 = 1048576
	}

	if other == nil {
		result, dropped := split(r, maxSize64)
		// TODO just emit a metric about this
		if dropped > 0 {
			return result, fmt.Errorf("dropped %d entries", dropped)
		}
		return result, nil
	}

	switch o := other.(type) {
	case *RequestBundle:
		o.mergeSingle(r)
		return o.expandSplit(maxSize64), nil
	case *Request:
		if r.groupKey != o.groupKey {
			return []exporterhelper.Request{r, o}, nil
		}
		r.merge(o)
		result, dropped := split(r, maxSize64)
		// TODO just emit a metric about this
		if dropped > 0 {
			return result, fmt.Errorf("dropped %d entries", dropped)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("invalid request type: expected *RequestBundle or *Request, got %T", other)
	}
}

func (r *Request) merge(other *Request) {
	r.request.Batch.Entries = append(r.request.Batch.Entries, other.request.Batch.Entries...)
}

// Returns a slice of requests and the number of entries that were dropped.
// An entry is only dropped if a request containing only that entry exceeds the max size.
func split(r *Request, maxSize int64) ([]exporterhelper.Request, int) {
	size := globalByteSizer.Sizeof(r)
	if size <= int64(maxSize) {
		return []exporterhelper.Request{r}, 0
	}

	if len(r.request.Batch.Entries) < 2 {
		return []exporterhelper.Request{}, 1
	}

	// split request into two
	mid := len(r.request.Batch.Entries) / 2
	other := NewRequest(&api.BatchCreateLogsRequest{
		Batch: &api.LogEntryBatch{
			StartTime: timestamppb.New(r.request.Batch.StartTime.AsTime()),
			Entries:   r.request.Batch.Entries[mid:],
			LogType:   r.request.Batch.LogType,
			Source: &api.EventSource{
				CollectorId: r.request.Batch.Source.CollectorId,
				CustomerId:  r.request.Batch.Source.CustomerId,
				Labels:      r.request.Batch.Source.Labels,
				Namespace:   r.request.Batch.Source.Namespace,
			},
		},
	})
	r.request.Batch.Entries = r.request.Batch.Entries[:mid]

	// re-enforce max size restriction on each half
	rSplit, rDropped := split(r, maxSize)
	otherSplit, otherDropped := split(other, maxSize)

	return append(rSplit, otherSplit...), rDropped + otherDropped
}
