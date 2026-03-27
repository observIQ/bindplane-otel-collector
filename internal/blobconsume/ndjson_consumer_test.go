// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package blobconsume //import "github.com/observiq/bindplane-otel-collector/internal/blobconsume"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func Test_ndjsonLogsConsumer_SingleLine(t *testing.T) {
	sink := &consumertest.LogsSink{}
	con := NewNDJSONLogsConsumer(sink, zap.NewNop())

	input := []byte(`{"host":"server1","message":"hello"}`)
	err := con.Consume(context.Background(), input)
	require.NoError(t, err)

	require.Equal(t, 1, sink.LogRecordCount())
	record := sink.AllLogs()[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	body := record.Body().Map()
	hostVal, ok := body.Get("host")
	require.True(t, ok)
	require.Equal(t, "server1", hostVal.Str())
}

func Test_ndjsonLogsConsumer_MultiLine(t *testing.T) {
	sink := &consumertest.LogsSink{}
	con := NewNDJSONLogsConsumer(sink, zap.NewNop())

	input := []byte(`{"host":"server1","message":"line1"}
{"host":"server2","message":"line2"}
{"host":"server3","message":"line3"}`)

	err := con.Consume(context.Background(), input)
	require.NoError(t, err)

	require.Equal(t, 3, sink.LogRecordCount())
}

func Test_ndjsonLogsConsumer_EmptyLines(t *testing.T) {
	sink := &consumertest.LogsSink{}
	con := NewNDJSONLogsConsumer(sink, zap.NewNop())

	input := []byte(`{"host":"server1"}

{"host":"server2"}

`)

	err := con.Consume(context.Background(), input)
	require.NoError(t, err)

	require.Equal(t, 2, sink.LogRecordCount())
}

func Test_ndjsonLogsConsumer_MalformedLinesSkipped(t *testing.T) {
	sink := &consumertest.LogsSink{}
	logger := zaptest.NewLogger(t)
	con := NewNDJSONLogsConsumer(sink, logger)

	input := []byte(`{"host":"server1"}
not valid json
{"host":"server2"}`)

	err := con.Consume(context.Background(), input)
	require.NoError(t, err)

	require.Equal(t, 2, sink.LogRecordCount())
}

func Test_ndjsonLogsConsumer_AllMalformed(t *testing.T) {
	sink := &consumertest.LogsSink{}
	con := NewNDJSONLogsConsumer(sink, zap.NewNop())

	input := []byte(`not json
also not json`)

	err := con.Consume(context.Background(), input)
	require.NoError(t, err)

	require.Equal(t, 0, sink.LogRecordCount())
}

func Test_ndjsonLogsConsumer_EmptyContent(t *testing.T) {
	sink := &consumertest.LogsSink{}
	con := NewNDJSONLogsConsumer(sink, zap.NewNop())

	err := con.Consume(context.Background(), []byte(""))
	require.NoError(t, err)

	require.Equal(t, 0, sink.LogRecordCount())
}
