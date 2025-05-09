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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/grpcq"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func TestRequest_ItemsCount(t *testing.T) {
	tests := []struct {
		name     string
		entries  []*api.LogEntry
		expected int
	}{
		{
			name:     "empty",
			entries:  []*api.LogEntry{},
			expected: 0,
		},
		{
			name:     "one entry",
			entries:  []*api.LogEntry{newLogEntry("test")},
			expected: 1,
		},
		{
			name:     "multiple entries",
			entries:  []*api.LogEntry{newLogEntry("test1"), newLogEntry("test2"), newLogEntry("test3")},
			expected: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := grpcq.NewRequest(newProto(tc.entries, nil))
			assert.Equal(t, tc.expected, req.ItemsCount())
		})
	}
}

func TestRequestMergeSplitNil(t *testing.T) {
	req := grpcq.NewRequest(newProto([]*api.LogEntry{newLogEntry("test1")}, nil))
	result, err := req.MergeSplit(context.Background(), 10000, exporterhelper.RequestSizerTypeBytes, nil)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, result[0].ItemsCount())
}

func TestRequestMergeDistinctGroups(t *testing.T) {
	req1 := grpcq.NewRequest(newProto([]*api.LogEntry{newLogEntry("test1")}, nil))
	req2 := grpcq.NewRequest(newProto([]*api.LogEntry{newLogEntry("test2")}, []*api.Label{{Key: "key", Value: "value"}}))

	result, err := req1.MergeSplit(context.Background(), 10000, exporterhelper.RequestSizerTypeBytes, req2)
	assert.NoError(t, err)

	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].ItemsCount())
	assert.Equal(t, 1, result[1].ItemsCount())

	reqMerged0 := result[0].(*grpcq.Request)
	entries0 := reqMerged0.Proto().Batch.Entries
	assert.Equal(t, 1, len(entries0))
	assert.Equal(t, []byte("test1"), entries0[0].Data)

	reqMerged1 := result[1].(*grpcq.Request)
	entries1 := reqMerged1.Proto().Batch.Entries
	assert.Equal(t, 1, len(entries1))
	assert.Equal(t, []byte("test2"), entries1[0].Data)
}

func TestRequestMergeSameGroup(t *testing.T) {
	req1 := grpcq.NewRequest(newProto([]*api.LogEntry{newLogEntry("test1")}, nil))
	req2 := grpcq.NewRequest(newProto([]*api.LogEntry{newLogEntry("test2")}, nil))

	result, err := req1.MergeSplit(context.Background(), 10000, exporterhelper.RequestSizerTypeBytes, req2)
	assert.NoError(t, err)

	assert.Len(t, result, 1)
	assert.Equal(t, 2, result[0].ItemsCount())

	reqMerged := result[0].(*grpcq.Request)
	entries := reqMerged.Proto().Batch.Entries
	assert.Equal(t, 2, len(entries))
	assert.Equal(t, []byte("test1"), entries[0].Data)
	assert.Equal(t, []byte("test2"), entries[1].Data)
}

func TestRequestMergeSplitLargeRequest(t *testing.T) {
	smallRequest := grpcq.NewRequest(newProto([]*api.LogEntry{newLogEntry("test1")}, nil))
	largeRequest := grpcq.NewRequest(newProto([]*api.LogEntry{newLogEntry("test2"), newLogEntry("test3")}, nil))

	maxSize := proto.Size(largeRequest.Proto())

	result, err := smallRequest.MergeSplit(context.Background(), maxSize, exporterhelper.RequestSizerTypeBytes, largeRequest)
	assert.NoError(t, err)

	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].ItemsCount())
	assert.Equal(t, 2, result[1].ItemsCount())

	// Ensure the other order works the same
	result, err = largeRequest.MergeSplit(context.Background(), maxSize, exporterhelper.RequestSizerTypeBytes, smallRequest)
	assert.NoError(t, err)

	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].ItemsCount())
	assert.Equal(t, 2, result[1].ItemsCount())
}

func TestRequestMergeSplitWithBundleSameGroup(t *testing.T) {
	request := grpcq.NewRequest(newProto([]*api.LogEntry{newLogEntry("test1")}, nil))
	bundle := grpcq.NewRequestBundle([]*api.BatchCreateLogsRequest{
		newProto([]*api.LogEntry{newLogEntry("test2")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test3")}, []*api.Label{{Key: "key", Value: "value"}}),
	})

	result, err := request.MergeSplit(context.Background(), 10000, exporterhelper.RequestSizerTypeBytes, bundle)
	assert.NoError(t, err)

	assert.Len(t, result, 2)

	reqWithLabels := result[0].(*grpcq.Request)
	reqWithoutLabels := result[1].(*grpcq.Request)
	if reqWithLabels.ItemsCount() == 2 {
		reqWithLabels, reqWithoutLabels = reqWithoutLabels, reqWithLabels
	}

	assert.Equal(t, 1, reqWithLabels.ItemsCount())
	assert.Equal(t, 2, reqWithoutLabels.ItemsCount())

	entriesWithoutLabels := reqWithoutLabels.Proto().Batch.Entries
	datas := map[string]string{}
	for _, entry := range entriesWithoutLabels {
		datas[string(entry.Data)] = string(entry.Data)
	}
	assert.Equal(t, 2, len(datas))
	assert.Contains(t, datas, "test1")
	assert.Contains(t, datas, "test2")

	entriesWithLabels := reqWithLabels.Proto().Batch.Entries
	assert.Equal(t, 1, len(entriesWithLabels))
	assert.Equal(t, []byte("test3"), entriesWithLabels[0].Data)
}
