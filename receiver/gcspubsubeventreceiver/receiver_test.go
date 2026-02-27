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

	subscriber "cloud.google.com/go/pubsub/apiv1"
	"cloud.google.com/go/pubsub/apiv1/pubsubpb"
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

// testEnv bundles the pstest server, high-level helpers, and an apiv1 subscriber
// client suitable for the receiver's pull-based architecture.
type testEnv struct {
	subClient *subscriber.SubscriberClient
	subPath   string
}

// setupTestEnv creates an in-process fake Pub/Sub server with a topic and
// subscription, and returns a low-level SubscriberClient connected to it.
func setupTestEnv(t *testing.T) testEnv {
	t.Helper()

	srv := pstest.NewServer()
	t.Cleanup(func() { _ = srv.Close() })

	conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	ctx := context.Background()

	// Use the publisher/admin API to create a topic + subscription on the fake server.
	pubClient, err := subscriber.NewSubscriberClient(ctx, option.WithGRPCConn(conn), option.WithoutAuthentication())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pubClient.Close() })

	return testEnv{
		subClient: pubClient,
		subPath:   "projects/test-project/subscriptions/test-sub",
	}
}

// newTestReceiver builds a minimal logsReceiver suitable for unit testing
// pullMessages and batch/cross-batch deduplication.
func newTestReceiver(t *testing.T, env testEnv) *logsReceiver {
	t.Helper()

	params := receivertest.NewNopSettings(metadata.Type)
	sink := new(consumertest.LogsSink)

	r := &logsReceiver{
		id: params.ID,
		cfg: &Config{
			Workers:      5,
			MaxExtension: time.Hour,
			PollInterval: 250 * time.Millisecond,
			DedupTTL:     5 * time.Minute,
			MaxLogSize:   1024 * 1024,
		},
		telemetry:        params.TelemetrySettings,
		next:             sink,
		offsetStorage:    storageclient.NewNopStorage(),
		subClient:        env.subClient,
		subscriptionPath: env.subPath,
		recent:           newRecentTracker(5 * time.Minute),
		msgChan:          make(chan *workerMessage, 10),
	}
	return r
}

// TestBatchDedup_DuplicateObjectInSameBatch verifies that when two messages
// for the same (bucket, object, generation) appear in the same Pull response,
// only one is dispatched to the worker channel.
func TestBatchDedup_DuplicateObjectInSameBatch(t *testing.T) {
	t.Parallel()

	msgs := []*worker.PullMessage{
		{
			AckID:     "ack-1",
			MessageID: "msg-1",
			Attributes: map[string]string{
				worker.AttrEventType:        worker.EventTypeObjectFinalize,
				worker.AttrBucketID:         "my-bucket",
				worker.AttrObjectID:         "my-object.txt",
				worker.AttrObjectGeneration: "12345",
			},
		},
		{
			AckID:     "ack-2",
			MessageID: "msg-2",
			Attributes: map[string]string{
				worker.AttrEventType:        worker.EventTypeObjectFinalize,
				worker.AttrBucketID:         "my-bucket",
				worker.AttrObjectID:         "my-object.txt",
				worker.AttrObjectGeneration: "12345",
			},
		},
	}

	env := setupTestEnv(t)
	r := newTestReceiver(t, env)

	unique := r.batchDedup(context.Background(), msgs)
	require.Len(t, unique, 1, "batch dedup must collapse duplicate (bucket, object, generation)")
	require.Equal(t, "ack-1", unique[0].AckID)
}

// TestBatchDedup_DifferentGenerationsNotDeduped verifies that two messages for
// the same object but different generations (legitimate overwrite) are both kept.
func TestBatchDedup_DifferentGenerationsNotDeduped(t *testing.T) {
	t.Parallel()

	msgs := []*worker.PullMessage{
		{
			AckID:     "ack-1",
			MessageID: "msg-1",
			Attributes: map[string]string{
				worker.AttrBucketID:         "bucket",
				worker.AttrObjectID:         "object",
				worker.AttrObjectGeneration: "100",
			},
		},
		{
			AckID:     "ack-2",
			MessageID: "msg-2",
			Attributes: map[string]string{
				worker.AttrBucketID:         "bucket",
				worker.AttrObjectID:         "object",
				worker.AttrObjectGeneration: "200",
			},
		},
	}

	env := setupTestEnv(t)
	r := newTestReceiver(t, env)

	unique := r.batchDedup(context.Background(), msgs)
	require.Len(t, unique, 2, "different generations must not be deduped")
}

// TestBatchDedup_UnrelatedObjectsNotDeduped verifies that messages for
// different objects pass through batch dedup unchanged.
func TestBatchDedup_UnrelatedObjectsNotDeduped(t *testing.T) {
	t.Parallel()

	msgs := []*worker.PullMessage{
		{
			AckID:     "ack-1",
			MessageID: "msg-1",
			Attributes: map[string]string{
				worker.AttrBucketID:         "bucket-a",
				worker.AttrObjectID:         "file-a.txt",
				worker.AttrObjectGeneration: "1",
			},
		},
		{
			AckID:     "ack-2",
			MessageID: "msg-2",
			Attributes: map[string]string{
				worker.AttrBucketID:         "bucket-b",
				worker.AttrObjectID:         "file-b.txt",
				worker.AttrObjectGeneration: "1",
			},
		},
	}

	env := setupTestEnv(t)
	r := newTestReceiver(t, env)

	unique := r.batchDedup(context.Background(), msgs)
	require.Len(t, unique, 2, "unrelated objects must not be deduped")
}

// TestRecentTracker_CrossBatchDedup verifies that a message whose object key
// was recently processed is skipped (acked without dispatching to the worker channel).
func TestRecentTracker_CrossBatchDedup(t *testing.T) {
	t.Parallel()

	key := objectKey{Bucket: "b", Object: "o", Generation: "1"}
	rt := newRecentTracker(5 * time.Minute)

	require.False(t, rt.IsDuplicate(key), "fresh key must not be duplicate")

	rt.Mark(key)
	require.True(t, rt.IsDuplicate(key), "recently marked key must be duplicate")
}

// TestRecentTracker_Expiry verifies that entries expire after the TTL.
func TestRecentTracker_Expiry(t *testing.T) {
	t.Parallel()

	key := objectKey{Bucket: "b", Object: "o", Generation: "1"}
	rt := newRecentTracker(1 * time.Millisecond) // tiny TTL for test

	rt.Mark(key)
	time.Sleep(5 * time.Millisecond) // wait for expiry

	require.False(t, rt.IsDuplicate(key), "expired key must not be duplicate")
}

// TestRecentTracker_Evict verifies that Evict removes expired entries.
func TestRecentTracker_Evict(t *testing.T) {
	t.Parallel()

	rt := newRecentTracker(1 * time.Millisecond)
	rt.Mark(objectKey{Bucket: "b", Object: "o", Generation: "1"})

	time.Sleep(5 * time.Millisecond)
	rt.Evict()

	rt.mu.Lock()
	defer rt.mu.Unlock()
	require.Empty(t, rt.seen, "Evict must remove expired entries")
}

// TestPullMessages_DispatchesUniqueBatched verifies the full pullMessages path:
// publish two messages for the same object, call pullMessages, and verify only
// one message reaches the worker channel.
func TestPullMessages_DispatchesUniqueBatched(t *testing.T) {
	t.Parallel()

	srv := pstest.NewServer()
	t.Cleanup(func() { _ = srv.Close() })

	conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	ctx := context.Background()
	const project = "test-project"
	const topicID = "test-topic"
	const subID = "test-sub"
	subPath := "projects/" + project + "/subscriptions/" + subID

	topicPath := "projects/" + project + "/topics/" + topicID

	// Create topic using the publisher admin API.
	pubClient, err := subscriber.NewPublisherClient(ctx, option.WithGRPCConn(conn), option.WithoutAuthentication())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pubClient.Close() })

	_, err = pubClient.CreateTopic(ctx, &pubsubpb.Topic{Name: topicPath})
	require.NoError(t, err)

	// Create subscription using the subscriber admin API.
	subClient, err := subscriber.NewSubscriberClient(ctx, option.WithGRPCConn(conn), option.WithoutAuthentication())
	require.NoError(t, err)
	t.Cleanup(func() { _ = subClient.Close() })

	_, err = subClient.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name:  subPath,
		Topic: topicPath,
	})
	require.NoError(t, err)

	// Publish two messages for the same (bucket, object, generation).
	attrs := map[string]string{
		worker.AttrEventType:        worker.EventTypeObjectFinalize,
		worker.AttrBucketID:         "bucket",
		worker.AttrObjectID:         "object.txt",
		worker.AttrObjectGeneration: "42",
	}
	for i := 0; i < 2; i++ {
		_, err = pubClient.Publish(ctx, &pubsubpb.PublishRequest{
			Topic: topicPath,
			Messages: []*pubsubpb.PubsubMessage{
				{Data: []byte("payload"), Attributes: attrs},
			},
		})
		require.NoError(t, err)
	}

	// Build a receiver wired to the pstest server.
	params := receivertest.NewNopSettings(metadata.Type)
	tb, err := metadata.NewTelemetryBuilder(params.TelemetrySettings)
	require.NoError(t, err)

	r := &logsReceiver{
		id: params.ID,
		cfg: &Config{
			Workers:      5,
			MaxExtension: time.Hour,
			PollInterval: 250 * time.Millisecond,
			DedupTTL:     5 * time.Minute,
			MaxLogSize:   1024 * 1024,
		},
		telemetry:        params.TelemetrySettings,
		metrics:          tb,
		next:             new(consumertest.LogsSink),
		offsetStorage:    storageclient.NewNopStorage(),
		subClient:        subClient,
		subscriptionPath: subPath,
		recent:           newRecentTracker(5 * time.Minute),
		msgChan:          make(chan *workerMessage, 10),
	}

	n := r.pullMessages(ctx)
	require.Equal(t, 1, n, "only one unique message should be dispatched after batch dedup")
}
