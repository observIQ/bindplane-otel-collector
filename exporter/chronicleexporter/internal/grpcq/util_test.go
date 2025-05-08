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

package grpcq_test

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
)

var now = time.Now()

func newLogEntry(data string) *api.LogEntry {
	return &api.LogEntry{
		Timestamp:      timestamppb.New(now),
		CollectionTime: timestamppb.New(now),
		Data:           []byte(data),
	}
}

func newProto(entries []*api.LogEntry, ingestionLabels []*api.Label) *api.BatchCreateLogsRequest {
	return &api.BatchCreateLogsRequest{
		Batch: &api.LogEntryBatch{
			StartTime: timestamppb.New(now),
			Entries:   entries,
			LogType:   "test-log-type",
			Source: &api.EventSource{
				Labels:      ingestionLabels,
				CollectorId: []byte("test-collector-id"),
				CustomerId:  []byte("test-customer-id"),
				Namespace:   "test-namespace",
			},
		},
	}
}
