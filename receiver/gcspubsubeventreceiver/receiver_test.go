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

package gcspubsubeventreceiver

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
	"github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver/internal/worker"
)

// setupTestPubSub creates an in-process fake Pub/Sub server and returns a topic
// and subscription ready to use. All resources are cleaned up when t finishes.
func setupTestPubSub(t *testing.T) (*pubsub.Topic, *pubsub.Subscription) {
	t.Helper()

	srv := pstest.NewServer()
	t.Cleanup(func() { _ = srv.Close() })

	conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, "test-project", option.WithGRPCConn(conn))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	topic, err := client.CreateTopic(ctx, "test-topic")
	require.NoError(t, err)

	sub, err := client.CreateSubscription(ctx, "test-sub", pubsub.SubscriptionConfig{
		Topic:       topic,
		AckDeadline: 10 * time.Second,
	})
	require.NoError(t, err)

	return topic, sub
}

// newTestReceiver builds a minimal logsReceiver suitable for unit-testing
// handleMessage. The worker pool panics if invoked, so tests that expect
// the deduplication path to short-circuit will catch unintended worker usage.
func newTestReceiver(t *testing.T, workerPoolFn func() any) *logsReceiver {
	t.Helper()

	params := receivertest.NewNopSettings(metadata.Type)
	sink := new(consumertest.LogsSink)

	r := &logsReceiver{
		id: params.ID,
		cfg: &Config{
			Workers:        5,
			MaxExtension:   time.Hour,
			MaxLogSize:     1024 * 1024,
			MaxLogsEmitted: 1000,
		},
		telemetry:     params.TelemetrySettings,
		next:          sink,
		offsetStorage: storageclient.NewNopStorage(),
	}
	r.workerPool.New = workerPoolFn
	return r
}

// TestHandleMessage_DuplicateObjectIsAckedWithoutProcessing verifies that when
// handleMessage receives a message whose (bucket, object) pair is already
// in-flight, the message is acked immediately and the worker pool is never
// invoked — no log records are emitted.
func TestHandleMessage_DuplicateObjectIsAckedWithoutProcessing(t *testing.T) {
	t.Parallel()

	topic, sub := setupTestPubSub(t)

	workerPoolInvoked := false
	r := newTestReceiver(t, func() any {
		workerPoolInvoked = true
		t.Error("worker pool must not be invoked for a duplicate message")
		return nil
	})

	// Simulate the first message already being processed by pre-populating inFlight.
	r.inFlight.Store("test-bucket/test-object.txt", struct{}{})

	attrs := map[string]string{
		worker.AttrEventType: worker.EventTypeObjectFinalize,
		worker.AttrBucketID:  "test-bucket",
		worker.AttrObjectID:  "test-object.txt",
	}

	ctx := context.Background()
	res := topic.Publish(ctx, &pubsub.Message{Data: []byte("payload"), Attributes: attrs})
	_, err := res.Get(ctx)
	require.NoError(t, err)

	receiveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = sub.Receive(receiveCtx, func(innerCtx context.Context, msg *pubsub.Message) {
			r.handleMessage(innerCtx, msg)
			close(done)
			cancel()
		})
	}()

	select {
	case <-done:
	case <-receiveCtx.Done():
		t.Fatal("timed out waiting for handleMessage to return")
	}

	require.False(t, workerPoolInvoked, "worker pool must not be invoked for a duplicate message")
	require.Equal(t, 0, r.next.(*consumertest.LogsSink).LogRecordCount(),
		"duplicate message must not produce log records")
}

// TestHandleMessage_InFlightKeyReleasedAfterProcessing verifies that the
// inFlight entry for a (bucket, object) pair is removed after handleMessage
// returns, so subsequent messages for the same object are processed normally.
func TestHandleMessage_InFlightKeyReleasedAfterProcessing(t *testing.T) {
	t.Parallel()

	topic, sub := setupTestPubSub(t)
	r := newTestReceiver(t, func() any { return nil })

	attrs := map[string]string{
		worker.AttrEventType: worker.EventTypeObjectFinalize,
		worker.AttrBucketID:  "my-bucket",
		worker.AttrObjectID:  "my-object.txt",
	}

	ctx := context.Background()
	res := topic.Publish(ctx, &pubsub.Message{Data: []byte("payload"), Attributes: attrs})
	_, err := res.Get(ctx)
	require.NoError(t, err)

	receiveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = sub.Receive(receiveCtx, func(innerCtx context.Context, msg *pubsub.Message) {
			// handleMessage will panic inside the worker (nil pool item), but the
			// defer r.inFlight.Delete inside handleMessage still runs before the
			// panic propagates. Recover here so we can assert on inFlight state.
			defer func() {
				recover() //nolint:errcheck
				close(done)
				cancel()
			}()
			r.handleMessage(innerCtx, msg)
		})
	}()

	select {
	case <-done:
	case <-receiveCtx.Done():
		t.Fatal("timed out waiting for handleMessage to return")
	}

	_, stillInFlight := r.inFlight.Load("my-bucket/my-object.txt")
	require.False(t, stillInFlight, "inFlight key must be removed after handleMessage returns")
}
