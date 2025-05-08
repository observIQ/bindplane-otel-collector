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
	"encoding/json"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"google.golang.org/protobuf/proto"
)

type encoding struct{}

type storedRequest struct {
	Requests [][]byte `json:"requests"`
}

func (e *encoding) Marshal(r exporterhelper.Request) ([]byte, error) {
	var requests storedRequest
	switch r := r.(type) {
	case *Request:
		requestBytes, err := proto.Marshal(r.request)
		if err != nil {
			return nil, err
		}
		requests.Requests = append(requests.Requests, requestBytes)
	case *RequestBundle:
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

func (e *encoding) Unmarshal(data []byte) (exporterhelper.Request, error) {
	var stored storedRequest
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}
	requests := make([]*api.BatchCreateLogsRequest, 0, len(stored.Requests))

	if len(stored.Requests) == 0 {
		return new(Request), nil
	}

	if len(stored.Requests) == 1 {
		request := new(api.BatchCreateLogsRequest)
		if err := proto.Unmarshal(stored.Requests[0], request); err != nil {
			return nil, err
		}
		return NewRequest(request), nil
	}

	for _, requestBytes := range stored.Requests {
		request := new(api.BatchCreateLogsRequest)
		if err := proto.Unmarshal(requestBytes, request); err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return NewRequestBundle(requests), nil
}
