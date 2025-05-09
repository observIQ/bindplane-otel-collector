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
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/grpcq"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func TestNewRequestBundle(t *testing.T) {
	requests := []*api.BatchCreateLogsRequest{
		newProto([]*api.LogEntry{newLogEntry("test1")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test2")}, []*api.Label{{Key: "key", Value: "value"}}),
	}

	bundle := grpcq.NewRequestBundle(requests)
	assert.NotNil(t, bundle)
	assert.Equal(t, bundle.ItemsCount(), 2)
}

func TestBundleMergeSplitWithNil(t *testing.T) {
	bundle1 := grpcq.NewRequestBundle([]*api.BatchCreateLogsRequest{
		newProto([]*api.LogEntry{newLogEntry("test1")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test2")}, []*api.Label{{Key: "key", Value: "value"}}),
	})

	result, err := bundle1.MergeSplit(context.Background(), 1000, exporterhelper.RequestSizerTypeBytes, nil)
	require.NoError(t, err)

	assert.Len(t, result, 2, "bundle contained two distinct log groups")

	for _, req := range result {
		r, ok := req.(*grpcq.Request)
		require.True(t, ok)
		assert.Equal(t, 1, r.ItemsCount(), "each log group contained one log entry")
	}
}
func TestBundleMergeSplitWithRequestBundleSameGroups(t *testing.T) {
	bundle1 := grpcq.NewRequestBundle([]*api.BatchCreateLogsRequest{
		newProto([]*api.LogEntry{newLogEntry("test1")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test2")}, []*api.Label{{Key: "key", Value: "value"}}),
	})

	bundle2 := grpcq.NewRequestBundle([]*api.BatchCreateLogsRequest{
		newProto([]*api.LogEntry{newLogEntry("test3")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test4")}, []*api.Label{{Key: "key", Value: "value"}}),
	})

	result, err := bundle1.MergeSplit(context.Background(), 1000, exporterhelper.RequestSizerTypeBytes, bundle2)
	require.NoError(t, err)

	assert.Len(t, result, 2, "bundles contained a total of two distinct log groups")

	for _, req := range result {
		r, ok := req.(*grpcq.Request)
		require.True(t, ok)
		assert.Equal(t, 2, r.ItemsCount(), "each log group contained two log entries")
	}
}

func TestBundleMergeSplitWithRequestBundleDifferentGroups(t *testing.T) {
	bundle1 := grpcq.NewRequestBundle([]*api.BatchCreateLogsRequest{
		newProto([]*api.LogEntry{newLogEntry("test1")}, []*api.Label{{Key: "key", Value: "value"}}),
		newProto([]*api.LogEntry{newLogEntry("test2")}, []*api.Label{{Key: "key2", Value: "value2"}}),
	})

	bundle2 := grpcq.NewRequestBundle([]*api.BatchCreateLogsRequest{
		newProto([]*api.LogEntry{newLogEntry("test3")}, []*api.Label{{Key: "key3", Value: "value3"}}),
		newProto([]*api.LogEntry{newLogEntry("test4")}, []*api.Label{{Key: "key4", Value: "value4"}}),
	})

	result, err := bundle1.MergeSplit(context.Background(), 1000, exporterhelper.RequestSizerTypeBytes, bundle2)
	require.NoError(t, err)

	assert.Len(t, result, 4, "bundles contained a total of four distinct log groups")

	for _, req := range result {
		r, ok := req.(*grpcq.Request)
		require.True(t, ok)
		assert.Equal(t, 1, r.ItemsCount(), "each log group contained one log entry")
	}
}

func TestBundleMergeSplitWithRequestDistinctGroup(t *testing.T) {
	bundle := grpcq.NewRequestBundle([]*api.BatchCreateLogsRequest{
		newProto([]*api.LogEntry{newLogEntry("test1")}, nil),
	})

	req := grpcq.NewRequest(newProto([]*api.LogEntry{newLogEntry("test2")}, []*api.Label{{Key: "key", Value: "value"}}))

	result, err := bundle.MergeSplit(context.Background(), 1000, exporterhelper.RequestSizerTypeBytes, req)
	require.NoError(t, err)

	assert.Len(t, result, 2, "bundle and request contained two distinct log groups")

	bundleReq, ok := result[1].(*grpcq.Request)
	require.True(t, ok)
	assert.Equal(t, 1, bundleReq.ItemsCount())
}

func TestBundleMergeSplitWithRequestSameGroup(t *testing.T) {
	sameGroup := []*api.Label{{Key: "key", Value: "value"}}
	bundle := grpcq.NewRequestBundle([]*api.BatchCreateLogsRequest{
		newProto([]*api.LogEntry{newLogEntry("test1")}, sameGroup),
	})

	req := grpcq.NewRequest(newProto([]*api.LogEntry{newLogEntry("test2")}, sameGroup))

	result, err := bundle.MergeSplit(context.Background(), 1000, exporterhelper.RequestSizerTypeBytes, req)
	require.NoError(t, err)

	assert.Len(t, result, 1, "bundle and request contained one distinct log group")

	bundleReq, ok := result[0].(*grpcq.Request)
	require.True(t, ok)
	assert.Equal(t, 2, bundleReq.ItemsCount(), "each log group contained two log entries")
}

func TestBundleMergeSplitLargeRequest(t *testing.T) {
	requests := []*api.BatchCreateLogsRequest{
		newProto([]*api.LogEntry{newLogEntry("test1")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test2")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test3")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test4")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test5")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test6")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test7")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test8")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test9")}, nil),
		newProto([]*api.LogEntry{newLogEntry("test10")}, nil),
	}

	// Actual size varies per run
	var maxSize int
	for _, req := range requests {
		if size := proto.Size(req); size > maxSize {
			maxSize = size
		}
	}

	bundle1 := grpcq.NewRequestBundle(requests[:5])
	bundle2 := grpcq.NewRequestBundle(requests[5:])

	result, err := bundle1.MergeSplit(context.Background(), maxSize, exporterhelper.RequestSizerTypeBytes, bundle2)
	require.NoError(t, err)

	assert.Len(t, result, 10, "each log is max size")
}
