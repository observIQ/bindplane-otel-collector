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

var _ exporterhelper.Request = &RequestBundle{}

func NewRequestBundle(requests []*api.BatchCreateLogsRequest) *RequestBundle {
	bundle := &RequestBundle{
		requests: make(map[string]*Request, len(requests)),
	}
	for _, request := range requests {
		request := NewRequest(request)
		if group, ok := bundle.requests[request.groupKey]; ok {
			group.merge(request)
		} else {
			bundle.requests[request.groupKey] = request
		}
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
	count := 0
	for _, req := range r.requests {
		count += req.ItemsCount()
	}
	return count
}

func (r *RequestBundle) MergeSplit(ctx context.Context, maxSize int, sizer exporterhelper.RequestSizerType, other exporterhelper.Request) ([]exporterhelper.Request, error) {
	if sizer != exporterhelper.RequestSizerTypeBytes {
		return nil, fmt.Errorf("unsupported sizer type: %s", sizer)
	}

	// TODO pass in maxSize based on cfg.BatchRequestSizeLimitGRPC
	maxSize64 := int64(maxSize)
	if maxSize64 == 0 {
		maxSize64 = 1048576
	}

	if other == nil {
		return r.expandSplit(maxSize64), nil
	}

	switch o := other.(type) {
	case *RequestBundle:
		r.mergeBundle(o)
		return r.expandSplit(maxSize64), nil
	case *Request:
		r.mergeSingle(o)
		return r.expandSplit(maxSize64), nil
	default:
		return nil, fmt.Errorf("invalid request type: expected *RequestBundle or *Request, got %T", other)
	}
}

func (m *RequestBundle) mergeBundle(other *RequestBundle) {
	for _, req := range other.requests {
		m.mergeSingle(req)
	}
}

func (m *RequestBundle) mergeSingle(other *Request) {
	if r, ok := m.requests[other.groupKey]; ok {
		r.merge(other)
	} else {
		m.requests[other.groupKey] = other
	}
}

func (m *RequestBundle) expandSplit(maxSize int64) []exporterhelper.Request {
	result := make([]exporterhelper.Request, 0, len(m.requests))
	for _, req := range m.requests {
		splitRequest, dropped := split(req, maxSize)
		if dropped > 0 {
			// TODO emit a metric about this
		}
		result = append(result, splitRequest...)
	}
	return result
}
