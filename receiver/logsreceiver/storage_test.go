// Copyright The OpenTelemetry Authors
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

package logsreceiver

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap/zaptest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/storagetest"
)

func TestStorage(t *testing.T) {
	ctx := context.Background()
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	r := createReceiver(t)
	host := storagetest.NewStorageHost(t, tempDir, "test")
	err = r.Start(ctx, host)
	require.NoError(t, err)

	myBytes := []byte("my_value")

	require.NoError(t, r.storageClient.Set(ctx, "key", myBytes))
	val, err := r.storageClient.Get(ctx, "key")
	require.NoError(t, err)
	require.Equal(t, myBytes, val)

	// Cycle the receiver
	require.NoError(t, r.Shutdown(ctx))
	for _, e := range host.GetExtensions() {
		require.NoError(t, e.Shutdown(ctx))
	}

	r = createReceiver(t)
	err = r.Start(ctx, host)
	require.NoError(t, err)

	// Value has persisted
	val, err = r.storageClient.Get(ctx, "key")
	require.NoError(t, err)
	require.Equal(t, myBytes, val)

	err = r.storageClient.Delete(ctx, "key")
	require.NoError(t, err)

	// Value is gone
	val, err = r.storageClient.Get(ctx, "key")
	require.NoError(t, err)
	require.Nil(t, val)

	require.NoError(t, r.Shutdown(ctx))

	_, err = r.storageClient.Get(ctx, "key")
	require.Error(t, err)
	require.Equal(t, "database not open", err.Error())
}

func TestFailOnMultipleStorageExtensions(t *testing.T) {
	ctx := context.Background()
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	r := createReceiver(t)
	host := storagetest.NewStorageHost(t, tempDir, "one", "two")
	err = r.Start(ctx, host)
	require.Error(t, err)
	require.Equal(t, "storage client: multiple storage extensions found", err.Error())
}

func createReceiver(t *testing.T) *receiver {
	params := component.ReceiverCreateSettings{
		Logger: zaptest.NewLogger(t),
	}
	mockConsumer := mockLogsConsumer{}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Pipeline = []map[string]interface{}{
		map[string]interface{}{
			"type": "noop",
		},
	}

	logsReceiver, err := factory.CreateLogsReceiver(
		context.Background(),
		params,
		cfg,
		&mockConsumer,
	)
	require.NoError(t, err, "receiver should successfully build")

	r, ok := logsReceiver.(*receiver)
	require.True(t, ok)
	return r
}

func TestPersisterImplementation(t *testing.T) {
	ctx := context.Background()
	myBytes := []byte("string")
	p := newMockPersister()

	err := p.Set(ctx, "key", myBytes)
	require.NoError(t, err)

	val, err := p.Get(ctx, "key")
	require.NoError(t, err)
	require.Equal(t, myBytes, val)

	err = p.Delete(ctx, "key")
	require.NoError(t, err)
}

func TestCheckpoint(t *testing.T) {
	t.Parallel()

	const baseLog = "This is a simple log line with the number %3d"

	ctx := context.Background()

	logsDir := newTempDir(t)
	storageDir := newTempDir(t)

	f := NewFactory()
	params := component.ReceiverCreateSettings{Logger: zaptest.NewLogger(t)}

	cfg := testdataRotateTestYamlAsMap(logsDir)
	cfg.Converter.MaxFlushCount = 1
	cfg.Converter.FlushInterval = time.Millisecond
	cfg.Pipeline = []map[string]interface{}{
		map[string]interface{}{
			"type":     "file_input",
			"include":  []string{fmt.Sprintf("%s/*", logsDir)},
			"start_at": "beginning",
		},
	}

	logger := newRecallLogger(t, logsDir)

	host := storagetest.NewStorageHost(t, storageDir, "test")
	sink := new(consumertest.LogsSink)
	rcvr, err := f.CreateLogsReceiver(ctx, params, cfg, sink)
	require.NoError(t, err, "failed to create receiver")
	require.NoError(t, rcvr.Start(ctx, host))

	// Write 2 logs
	logger.log(fmt.Sprintf(baseLog, 0))
	logger.log(fmt.Sprintf(baseLog, 1))

	// Expect them now, since the receiver is running
	require.Eventually(t,
		expectLogs(sink, logger.recall()),
		time.Second,
		10*time.Millisecond,
		"expected 2 but got %d logs",
		sink.LogRecordCount(),
	)

	// Shut down the components
	require.NoError(t, rcvr.Shutdown(ctx))
	for _, e := range host.GetExtensions() {
		require.NoError(t, e.Shutdown(ctx))
	}

	// Write 3 more logs while the collector is not running
	logger.log(fmt.Sprintf(baseLog, 2))
	logger.log(fmt.Sprintf(baseLog, 3))
	logger.log(fmt.Sprintf(baseLog, 4))

	// Start the components again
	host = storagetest.NewStorageHost(t, storageDir, "test")
	rcvr, err = f.CreateLogsReceiver(ctx, params, cfg, sink)
	require.NoError(t, err, "failed to create receiver")
	require.NoError(t, rcvr.Start(ctx, host))
	sink.Reset()

	// Expect only the new 3
	require.Eventually(t,
		expectLogs(sink, logger.recall()),
		time.Second,
		10*time.Millisecond,
		"expected 3 but got %d logs",
		sink.LogRecordCount(),
	)
	sink.Reset()

	// Write 100 more, to ensure we're past the fingerprint size
	for i := 100; i < 200; i++ {
		logger.log(fmt.Sprintf(baseLog, i))
	}

	// Expect the new 100
	require.Eventually(t,
		expectLogs(sink, logger.recall()),
		time.Second,
		10*time.Millisecond,
		"expected 100 but got %d logs",
		sink.LogRecordCount(),
	)

	// Shut down the components
	require.NoError(t, rcvr.Shutdown(ctx))
	for _, e := range host.GetExtensions() {
		require.NoError(t, e.Shutdown(ctx))
	}

	// Write 5 more logs while the collector is not running
	logger.log(fmt.Sprintf(baseLog, 5))
	logger.log(fmt.Sprintf(baseLog, 6))
	logger.log(fmt.Sprintf(baseLog, 7))
	logger.log(fmt.Sprintf(baseLog, 8))
	logger.log(fmt.Sprintf(baseLog, 9))

	// Start the components again
	host = storagetest.NewStorageHost(t, storageDir, "test")
	rcvr, err = f.CreateLogsReceiver(ctx, params, cfg, sink)
	require.NoError(t, err, "failed to create receiver")
	require.NoError(t, rcvr.Start(ctx, host))
	sink.Reset()

	// Expect only the new 5
	require.Eventually(t,
		expectLogs(sink, logger.recall()),
		time.Second,
		10*time.Millisecond,
		"expected 5 but got %d logs",
		sink.LogRecordCount(),
	)

	// Shut down the components
	require.NoError(t, rcvr.Shutdown(ctx))
	for _, e := range host.GetExtensions() {
		require.NoError(t, e.Shutdown(ctx))
	}
}
