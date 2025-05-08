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
	"math"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

var _ exporterhelper.Request = &RequestBundle{}

func NewRequestBundle(requests []*api.BatchCreateLogsRequest) *RequestBundle {
	bundle := &RequestBundle{
		requests: make(map[string]*Request, len(requests)),
	}
	for _, request := range requests {
		request := NewRequest(request)
		bundle.requests[request.groupKey] = request
	}
	return bundle
}

// RequestBundle is kind of a hack. exporterhelper expects each plog.Logs to be converted
// into a single request, but we need to send one request per set of ingestion labels.
// This struct acts as a temporary container for multiple requests. In order to trigger it to
// be split apart, we rely on the exceeding the configured max size.
// TODO A max size of 0 may be problematic.
type RequestBundle struct {
	// map[groupKey]Request
	requests map[string]*Request
}

func (r *RequestBundle) ItemsCount() int {
	return math.MaxInt // Force a conversion to []Request
}

func (r *RequestBundle) MergeSplit(ctx context.Context, maxSize int, sizer exporterhelper.RequestSizerType, other exporterhelper.Request) ([]exporterhelper.Request, error) {
	if sizer != exporterhelper.RequestSizerTypeBytes {
		return nil, fmt.Errorf("unsupported sizer type: %s", sizer)
	}

	switch o := other.(type) {
	case *RequestBundle:
		// Merge the requests by groupKey and then split into individual requests.
		for groupKey, req := range o.requests {
			if rGroup, ok := r.requests[groupKey]; ok {
				// Merge the requests
				rGroup.request.Batch.Entries = append(rGroup.request.Batch.Entries, req.request.Batch.Entries...)
				if req.request.Batch.StartTime.AsTime().Before(rGroup.request.Batch.StartTime.AsTime()) {
					rGroup.request.Batch.StartTime = req.request.Batch.StartTime
				}
			} else {
				r.requests[groupKey] = req
			}
		}
		return r.MergeSplit(ctx, maxSize, sizer, r)
	case *Request:
		return append([]exporterhelper.Request{o}, r.expand()...), nil
	default:
		return nil, fmt.Errorf("invalid request type: expected *RequestBundle or *Request, got %T", other)
	}
}

func (m *RequestBundle) expand() []exporterhelper.Request {
	result := make([]exporterhelper.Request, 0, len(m.requests))
	for _, req := range m.requests {
		result = append(result, req)
	}
	return result
}
