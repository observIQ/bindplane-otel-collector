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
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/grpcq"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
)

func TestEncodingRoundTrip(t *testing.T) {
	enc := new(grpcq.Encoding)

	t.Run("request/singleentry/nolabels", func(t *testing.T) {
		request := grpcq.NewRequest(
			newProto([]*api.LogEntry{
				newLogEntry("test-data"),
			}, nil),
		)

		encoded, err := enc.Marshal(request)
		require.NoError(t, err)

		decoded, err := enc.Unmarshal(encoded)
		require.NoError(t, err)

		require.Equal(t, request, decoded)
	})

	t.Run("request/singleentry/labels", func(t *testing.T) {
		request := grpcq.NewRequest(
			newProto([]*api.LogEntry{
				newLogEntry("test-data"),
			}, []*api.Label{
				{Key: "key", Value: "value"},
				{Key: "key2", Value: "value2"},
			}),
		)

		encoded, err := enc.Marshal(request)
		require.NoError(t, err)

		decoded, err := enc.Unmarshal(encoded)
		require.NoError(t, err)

		require.Equal(t, request, decoded)
	})

	t.Run("request/multipleentries/nolabels", func(t *testing.T) {
		request := grpcq.NewRequest(
			newProto([]*api.LogEntry{
				newLogEntry("test-data"),
				newLogEntry("test-data2"),
			}, nil),
		)

		encoded, err := enc.Marshal(request)
		require.NoError(t, err)

		decoded, err := enc.Unmarshal(encoded)
		require.NoError(t, err)

		require.Equal(t, request, decoded)
	})

	t.Run("request/multipleentries/labels", func(t *testing.T) {
		request := grpcq.NewRequest(
			newProto([]*api.LogEntry{
				newLogEntry("test-data"),
				newLogEntry("test-data2"),
			}, []*api.Label{
				{Key: "key", Value: "value"},
				{Key: "key2", Value: "value2"},
			}),
		)

		encoded, err := enc.Marshal(request)
		require.NoError(t, err)

		decoded, err := enc.Unmarshal(encoded)
		require.NoError(t, err)

		require.Equal(t, request, decoded)
	})

	t.Run("bundle/singlegroup", func(t *testing.T) {
		bundle := grpcq.NewRequestBundle(
			[]*api.BatchCreateLogsRequest{
				newProto([]*api.LogEntry{
					newLogEntry("test-data"),
				}, nil),
			},
		)

		encoded, err := enc.Marshal(bundle)
		require.NoError(t, err)

		decoded, err := enc.Unmarshal(encoded)
		require.NoError(t, err)

		// Expect a single Request object (not a bundle)
		expectedProto := newProto([]*api.LogEntry{
			newLogEntry("test-data"),
		}, nil)

		// Use deep equality to ignore internal fields like sizeCache
		decodedReq, ok := decoded.(*grpcq.Request)
		require.True(t, ok, "expected decoded value to be a *Request")

		// Use proto.Equal to compare the actual protocol buffer messages
		require.True(t,
			proto.Equal(expectedProto, decodedReq.Proto()),
			"Protocol buffer messages should be equal",
		)
	})

	t.Run("bundle/multiplegroups", func(t *testing.T) {
		bundle := grpcq.NewRequestBundle(
			[]*api.BatchCreateLogsRequest{
				newProto([]*api.LogEntry{
					newLogEntry("test-data"),
				}, []*api.Label{
					{Key: "key", Value: "value"},
				}),
				newProto([]*api.LogEntry{
					newLogEntry("test-data2"),
				}, []*api.Label{
					{Key: "key2", Value: "value2"},
				}),
			},
		)

		encoded, err := enc.Marshal(bundle)
		require.NoError(t, err)

		decoded, err := enc.Unmarshal(encoded)
		require.NoError(t, err)

		// Expect a bundle because multiple groups must fit into a single request
		_, ok := decoded.(*grpcq.RequestBundle)
		require.True(t, ok, "Expected a RequestBundle")
	})
}
