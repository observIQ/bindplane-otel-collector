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

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver/internal/worker"
)

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

// processTestMessage constructs a PullMessage with the given attributes and
// calls w.ProcessMessage directly (no Pub/Sub server needed for early-return
// paths that don't touch GCS).
func processTestMessage(t *testing.T, attrs map[string]string, w *worker.Worker) bool {
	t.Helper()

	msg := &worker.PullMessage{
		AckID:      "test-ack-id",
		MessageID:  "test-msg-id",
		Attributes: attrs,
	}

	ctx := context.Background()
	return w.ProcessMessage(ctx, msg, "projects/test/subscriptions/test-sub", func() {})
}

// TestProcessMessage_NonFinalizeEventSkipped verifies that events other than
// OBJECT_FINALIZE are acked immediately without touching GCS (nil storage client).
func TestProcessMessage_NonFinalizeEventSkipped(t *testing.T) {
	t.Parallel()

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
	processed := processTestMessage(t, attrs, w)

	require.False(t, processed, "non-finalize event should not count as processed")
	require.Equal(t, 0, sink.LogRecordCount())
}

// TestProcessMessage_MissingBucketID verifies that messages without a bucketId
// are acked immediately without touching GCS.
func TestProcessMessage_MissingBucketID(t *testing.T) {
	t.Parallel()

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
	processed := processTestMessage(t, attrs, w)

	require.False(t, processed, "missing bucket should not count as processed")
	require.Equal(t, 0, sink.LogRecordCount())
}

// TestProcessMessage_MissingObjectID verifies that messages without an objectId
// are acked immediately without touching GCS.
func TestProcessMessage_MissingObjectID(t *testing.T) {
	t.Parallel()

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
	processed := processTestMessage(t, attrs, w)

	require.False(t, processed, "missing object should not count as processed")
	require.Equal(t, 0, sink.LogRecordCount())
}

// TestProcessMessage_BucketNameFilterNoMatch verifies that a message whose
// bucket name does not match the configured filter is acked without touching GCS.
func TestProcessMessage_BucketNameFilterNoMatch(t *testing.T) {
	t.Parallel()

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
	processed := processTestMessage(t, attrs, w)

	require.False(t, processed, "filtered bucket should not count as processed")
	require.Equal(t, 0, sink.LogRecordCount())
}

// TestProcessMessage_ObjectKeyFilterNoMatch verifies that a message whose
// object key does not match the configured filter is acked without touching GCS.
func TestProcessMessage_ObjectKeyFilterNoMatch(t *testing.T) {
	t.Parallel()

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
	processed := processTestMessage(t, attrs, w)

	require.False(t, processed, "filtered object key should not count as processed")
	require.Equal(t, 0, sink.LogRecordCount())
}
