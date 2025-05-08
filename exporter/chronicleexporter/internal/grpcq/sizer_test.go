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
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/grpcq"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
)

func TestByteSizer_Sizeof(t *testing.T) {
	sizer := new(grpcq.ByteSizer)

	t.Run("Request", func(t *testing.T) {
		req := grpcq.NewRequest(newProto([]*api.LogEntry{
			newLogEntry("test payload"),
		}, nil))
		expectedSize := int64(proto.Size(req.Proto()))
		actualSize := sizer.Sizeof(req)
		assert.Equal(t, expectedSize, actualSize, "Sizeof should return correct size for Request")
	})

	t.Run("RequestBundle", func(t *testing.T) {
		bundle := grpcq.NewRequestBundle([]*api.BatchCreateLogsRequest{
			newProto([]*api.LogEntry{newLogEntry("test1")}, nil),
			newProto([]*api.LogEntry{newLogEntry("test2")}, []*api.Label{{Key: "key", Value: "value"}}),
		})

		actualSize := sizer.Sizeof(bundle)
		assert.Equal(t, int64(math.MaxInt64), actualSize, "Sizeof should return math.MaxInt64 for RequestBundle")
	})
}
