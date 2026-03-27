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
)

func Test_rawTextLogsConsumer(t *testing.T) {
	sink := &consumertest.LogsSink{}
	con := NewRawTextLogsConsumer(sink)

	input := "This is some raw log text\nwith multiple lines\n"
	err := con.Consume(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Equal(t, 1, sink.LogRecordCount())
	record := sink.AllLogs()[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	require.Equal(t, input, record.Body().Str())
}

func Test_rawTextLogsConsumer_EmptyContent(t *testing.T) {
	sink := &consumertest.LogsSink{}
	con := NewRawTextLogsConsumer(sink)

	err := con.Consume(context.Background(), []byte(""))
	require.NoError(t, err)

	require.Equal(t, 1, sink.LogRecordCount())
	record := sink.AllLogs()[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	require.Equal(t, "", record.Body().Str())
}
