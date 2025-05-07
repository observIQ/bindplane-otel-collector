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

package chronicleexporter

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"sort"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"google.golang.org/protobuf/proto"
)

func grpcQueueBatchSettings() exporterhelper.QueueBatchSettings {
	return exporterhelper.QueueBatchSettings{
		Encoding: &grpcEncoding{},
		Sizers: map[exporterhelper.RequestSizerType]exporterhelper.RequestSizer{
			exporterhelper.RequestSizerTypeBytes: &grpcRequestSizer{},
		},
	}
}

// TODO define protobufs
type grpcEncoding struct{}

type storedRequest struct {
	Requests [][]byte `json:"requests"`
}

func (e *grpcEncoding) Marshal(r exporterhelper.Request) ([]byte, error) {
	var requests storedRequest
	switch r := r.(type) {
	case *grpcRequest:
		requestBytes, err := proto.Marshal(r.request)
		if err != nil {
			return nil, err
		}
		requests.Requests = append(requests.Requests, requestBytes)
	case *grpcMultiRequest:
		requests.Requests = make([][]byte, 0, len(r.requests))
		for _, req := range r.requests {
			requestBytes, err := proto.Marshal(req.request)
			if err != nil {
				return nil, err
			}
			requests.Requests = append(requests.Requests, requestBytes)
		}
	default:
		return nil, fmt.Errorf("unsupported request type: %T", r)
	}
	return json.Marshal(requests)
}

func (e *grpcEncoding) Unmarshal(data []byte) (exporterhelper.Request, error) {
	var requests storedRequest
	if err := json.Unmarshal(data, &requests); err != nil {
		return nil, err
	}
	requests.Requests = make([][]byte, 0, len(requests.Requests))

	if len(requests.Requests) == 0 {
		return &grpcRequest{}, nil
	}

	if len(requests.Requests) == 1 {
		request := new(api.BatchCreateLogsRequest)
		if err := proto.Unmarshal(requests.Requests[0], request); err != nil {
			return nil, err
		}
		return &grpcRequest{
			request:  request,
			groupKey: computeGroupKey(request.GetBatch()),
		}, nil
	}

	multiRequest := &grpcMultiRequest{
		requests: make(map[string]grpcRequest, len(requests.Requests)),
	}
	for _, requestBytes := range requests.Requests {
		request := new(api.BatchCreateLogsRequest)
		if err := proto.Unmarshal(requestBytes, request); err != nil {
			return nil, err
		}
		groupKey := computeGroupKey(request.GetBatch())
		multiRequest.requests[groupKey] = grpcRequest{
			request:  request,
			groupKey: groupKey,
		}
	}
	return multiRequest, nil
}

var grpcGlobalSizer = &grpcRequestSizer{}

type grpcRequestSizer struct{}

func (s *grpcRequestSizer) Sizeof(r exporterhelper.Request) int64 {
	if grpcReq, ok := r.(*grpcRequest); ok {
		return int64(proto.Size(grpcReq.request))
	}
	return 0
}

// grpcMultiRequest is kind of a hack. exporterhelper expects each plog.Logs to be converted
// into a single request, but we need to send one request per set of ingestion labels.
// This struct acts as a temporary container for multiple requests. In order to trigger it to
// be split apart, we rely on the exceeding the configured max size.
// TODO A max size of 0 may be problematic.
type grpcMultiRequest struct {
	// map[groupKey]grpcRequest
	requests map[string]grpcRequest
}

func (r *grpcMultiRequest) ItemsCount() int {
	return math.MaxInt // Force a conversion to []grpcRequest
}

func (r *grpcMultiRequest) MergeSplit(ctx context.Context, maxSize int, sizer exporterhelper.RequestSizerType, other exporterhelper.Request) ([]exporterhelper.Request, error) {
	if sizer != exporterhelper.RequestSizerTypeBytes {
		return nil, fmt.Errorf("unsupported sizer type: %s", sizer)
	}

	switch o := other.(type) {
	case *grpcMultiRequest:
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
	case *grpcRequest:
		return append([]exporterhelper.Request{o}, expandMulti(r)...), nil
	default:
		return nil, fmt.Errorf("invalid request type: expected *grpcMultiRequest or *grpcRequest, got %T", other)
	}
}

func expandMulti(m *grpcMultiRequest) []exporterhelper.Request {
	result := make([]exporterhelper.Request, 0, len(m.requests))
	for _, req := range m.requests {
		result = append(result, &req)
	}
	return result
}

type grpcRequest struct {
	request  *api.BatchCreateLogsRequest
	groupKey string // Hash key for quick compatibility checks
}

// ItemsCount returns a number of basic items in the request where item is the smallest piece of data that can be
// sent. For example, for OTLP exporter, this value represents the number of spans,
// metric data points or log records.
func (r *grpcRequest) ItemsCount() int {
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
func (r *grpcRequest) MergeSplit(ctx context.Context, maxSize int, sizer exporterhelper.RequestSizerType, other exporterhelper.Request) ([]exporterhelper.Request, error) {
	if sizer != exporterhelper.RequestSizerTypeBytes {
		return nil, fmt.Errorf("unsupported sizer type: %s", sizer)
	}

	if maxSize <= 0 {
		return []exporterhelper.Request{r}, nil
	}

	switch o := other.(type) {
	case *grpcMultiRequest:
		return append([]exporterhelper.Request{r}, expandMulti(o)...), nil
	case *grpcRequest:
		return mergeSplitSingles(maxSize, r, o), nil
	default:
		return nil, fmt.Errorf("invalid request type: expected *grpcMultiRequest or *grpcRequest, got %T", other)
	}
}

func mergeSplitSingles(maxSize int, r *grpcRequest, o *grpcRequest) []exporterhelper.Request {
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

		newSize := grpcGlobalSizer.Sizeof(r)
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

// computeGroupKey creates a unique key for grouping logs based on namespace, logType and ingestionLabels
func computeGroupKey(batch *api.LogEntryBatch) string {
	h := fnv.New64a()
	h.Write([]byte(batch.GetLogType()))
	h.Write([]byte(batch.GetSource().GetNamespace()))

	labels := batch.GetSource().GetLabels()
	if len(labels) == 0 {
		return fmt.Sprintf("%x", h.Sum64())
	}

	pairs := make([]string, 0, len(labels))
	for _, label := range labels {
		pairs = append(pairs, fmt.Sprintf("%s=%s", label.Key, label.Value))
	}

	sort.Strings(pairs)

	for _, pair := range pairs {
		h.Write([]byte(pair))
	}
	return fmt.Sprintf("%x", h.Sum64())
}
