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

package worker_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver/internal/worker"
)

// setupPubSub creates an in-process fake Pub/Sub server and returns a topic and
// subscription that are ready to use. All resources are cleaned up when t finishes.
func setupPubSub(t *testing.T) (*pubsub.Topic, *pubsub.Subscription) {
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

// newObsrecv creates an ObsReport suitable for use in tests.
func newObsrecv(t *testing.T) *receiverhelper.ObsReport {
	t.Helper()

	params := receivertest.NewNopSettings(metadata.Type)
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             params.ID,
		Transport:              "pubsub",
		ReceiverCreateSettings: params,
	})
	require.NoError(t, err)
	return obsrecv
}

// processMessage publishes a single message with the given attributes, then
// drives sub.Receive to deliver that message to w.ProcessMessage.  It returns
// after ProcessMessage has returned (or after a 10-second timeout).
func processMessage(t *testing.T, topic *pubsub.Topic, sub *pubsub.Subscription, attrs map[string]string, w *worker.Worker) {
	t.Helper()

	ctx := context.Background()

	// Publish one message.
	res := topic.Publish(ctx, &pubsub.Message{
		Data:       []byte("test-payload"),
		Attributes: attrs,
	})
	_, err := res.Get(ctx)
	require.NoError(t, err)

	// Receive the message and hand it to the worker.  Cancel the receive context
	// as soon as the first message has been processed so that sub.Receive returns.
	receiveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	done := make(chan struct{}, 1)

	go func() {
		_ = sub.Receive(receiveCtx, func(innerCtx context.Context, msg *pubsub.Message) {
			w.ProcessMessage(innerCtx, msg)
			done <- struct{}{}
			cancel()
		})
	}()

	select {
	case <-done:
		// ProcessMessage completed normally.
	case <-receiveCtx.Done():
		t.Fatal("timed out waiting for ProcessMessage to be called")
	}
}

// TestProcessMessage_NonFinalizeEventSkipped verifies that events other than
// OBJECT_FINALIZE are acked immediately without touching GCS (nil storage client).
func TestProcessMessage_NonFinalizeEventSkipped(t *testing.T) {
	t.Parallel()

	topic, sub := setupPubSub(t)
	sink := new(consumertest.LogsSink)

	// nil storage client is intentional — GCS must never be called.
	w := worker.New(
		receivertest.NewNopSettings(metadata.Type).TelemetrySettings,
		sink,
		nil, // storageClient
		newObsrecv(t),
		4096,
		1000,
	)

	attrs := map[string]string{
		worker.AttrEventType: "OBJECT_DELETE",
		worker.AttrBucketID:  "bucket",
		worker.AttrObjectID:  "object",
	}
	processMessage(t, topic, sub, attrs, w)

	require.Equal(t, 0, sink.LogRecordCount())
}

// TestProcessMessage_MissingBucketID verifies that messages without a bucketId
// are acked immediately without touching GCS.
func TestProcessMessage_MissingBucketID(t *testing.T) {
	t.Parallel()

	topic, sub := setupPubSub(t)
	sink := new(consumertest.LogsSink)

	w := worker.New(
		receivertest.NewNopSettings(metadata.Type).TelemetrySettings,
		sink,
		nil, // storageClient
		newObsrecv(t),
		4096,
		1000,
	)

	attrs := map[string]string{
		worker.AttrEventType: worker.EventTypeObjectFinalize,
		worker.AttrBucketID:  "",
		worker.AttrObjectID:  "test",
	}
	processMessage(t, topic, sub, attrs, w)

	require.Equal(t, 0, sink.LogRecordCount())
}

// TestProcessMessage_MissingObjectID verifies that messages without an objectId
// are acked immediately without touching GCS.
func TestProcessMessage_MissingObjectID(t *testing.T) {
	t.Parallel()

	topic, sub := setupPubSub(t)
	sink := new(consumertest.LogsSink)

	w := worker.New(
		receivertest.NewNopSettings(metadata.Type).TelemetrySettings,
		sink,
		nil, // storageClient
		newObsrecv(t),
		4096,
		1000,
	)

	attrs := map[string]string{
		worker.AttrEventType: worker.EventTypeObjectFinalize,
		worker.AttrBucketID:  "test",
		worker.AttrObjectID:  "",
	}
	processMessage(t, topic, sub, attrs, w)

	require.Equal(t, 0, sink.LogRecordCount())
}

// TestProcessMessage_BucketNameFilterNoMatch verifies that a message whose
// bucket name does not match the configured filter is acked without touching GCS.
func TestProcessMessage_BucketNameFilterNoMatch(t *testing.T) {
	t.Parallel()

	topic, sub := setupPubSub(t)
	sink := new(consumertest.LogsSink)

	w := worker.New(
		receivertest.NewNopSettings(metadata.Type).TelemetrySettings,
		sink,
		nil, // storageClient
		newObsrecv(t),
		4096,
		1000,
		worker.WithBucketNameFilter(regexp.MustCompile(`^xyz$`)),
	)

	attrs := map[string]string{
		worker.AttrEventType: worker.EventTypeObjectFinalize,
		worker.AttrBucketID:  "mybucket",
		worker.AttrObjectID:  "myobj",
	}
	processMessage(t, topic, sub, attrs, w)

	require.Equal(t, 0, sink.LogRecordCount())
}

// TestProcessMessage_ObjectKeyFilterNoMatch verifies that a message whose
// object key does not match the configured filter is acked without touching GCS.
func TestProcessMessage_ObjectKeyFilterNoMatch(t *testing.T) {
	t.Parallel()

	topic, sub := setupPubSub(t)
	sink := new(consumertest.LogsSink)

	w := worker.New(
		receivertest.NewNopSettings(metadata.Type).TelemetrySettings,
		sink,
		nil, // storageClient
		newObsrecv(t),
		4096,
		1000,
		worker.WithObjectKeyFilter(regexp.MustCompile(`^xyz$`)),
	)

	attrs := map[string]string{
		worker.AttrEventType: worker.EventTypeObjectFinalize,
		worker.AttrBucketID:  "mybucket",
		worker.AttrObjectID:  "myobj",
	}
	processMessage(t, topic, sub, attrs, w)

	require.Equal(t, 0, sink.LogRecordCount())
}
