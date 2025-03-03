// Copyright Bindplane, Inc.
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

package rehydration //import "github.com/observiq/bindplane-otel-collector/internal/rehydration"

import (
	"context"
	"testing"

	"github.com/observiq/bindplane-otel-collector/internal/testutils"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

func Test_metricsConsumer(t *testing.T) {
	testConsumer := &consumertest.MetricsSink{}
	con := NewMetricsConsumer(testConsumer)

	metrics, jsonBytes := testutils.GenerateTestMetrics(t)

	err := con.Consume(context.Background(), jsonBytes)
	require.NoError(t, err)

	require.Equal(t, metrics.DataPointCount(), testConsumer.DataPointCount())

	// Test case of failed unmarshal
	err = con.Consume(context.Background(), []byte("nope"))
	require.Error(t, err)
}

func Test_logsConsumer(t *testing.T) {
	testConsumer := &consumertest.LogsSink{}
	con := NewLogsConsumer(testConsumer)

	logs, jsonBytes := testutils.GenerateTestLogs(t)

	err := con.Consume(context.Background(), jsonBytes)
	require.NoError(t, err)

	require.Equal(t, logs.LogRecordCount(), testConsumer.LogRecordCount())

	// Test case of failed unmarshal
	err = con.Consume(context.Background(), []byte("nope"))
	require.Error(t, err)
}

func Test_tracesConsumer(t *testing.T) {
	testConsumer := &consumertest.TracesSink{}
	con := NewTracesConsumer(testConsumer)

	traces, jsonBytes := testutils.GenerateTestTraces(t)

	err := con.Consume(context.Background(), jsonBytes)
	require.NoError(t, err)

	require.Equal(t, traces.SpanCount(), testConsumer.SpanCount())

	// Test case of failed unmarshal
	err = con.Consume(context.Background(), []byte("nope"))
	require.Error(t, err)
}
