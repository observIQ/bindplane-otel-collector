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

package gcspubsubeventreceiver // import "github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver"

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	subscriber "cloud.google.com/go/pubsub/apiv1"
	"cloud.google.com/go/pubsub/apiv1/pubsubpb"
	"cloud.google.com/go/storage"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"
	"google.golang.org/api/option"

	"github.com/observiq/bindplane-otel-collector/internal/storageclient"
	"github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/receiver/gcspubsubeventreceiver/internal/worker"
)

type logsReceiver struct {
	id        component.ID
	cfg       *Config
	telemetry component.TelemetrySettings
	metrics   *metadata.TelemetryBuilder
	next      consumer.Logs
	startOnce sync.Once
	stopOnce  sync.Once

	subClient        *subscriber.SubscriberClient
	storageClient    *storage.Client
	subscriptionPath string
	workerPool       sync.Pool
	workerWg         sync.WaitGroup

	cancel   context.CancelFunc
	pollDone chan struct{}

	offsetStorage storageclient.StorageClient

	bucketNameFilter *regexp.Regexp
	objectKeyFilter  *regexp.Regexp

	// recent tracks recently-processed (bucket, object, generation) keys for
	// cross-batch deduplication. Batch deduplication catches duplicates within
	// a single Pull response; the recent tracker catches sequential duplicates
	// that arrive in separate Pull responses (typically ~1 s apart).
	recent *recentTracker

	// msgChan distributes pulled messages to the fixed worker pool.
	msgChan chan *workerMessage

	// observer for metrics about the receiver
	obsrecv *receiverhelper.ObsReport
}

type workerMessage struct {
	msg *worker.PullMessage
}

func newLogsReceiver(params receiver.Settings, cfg *Config, next consumer.Logs, tb *metadata.TelemetryBuilder) (component.Component, error) {
	id := params.ID
	tel := params.TelemetrySettings

	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             id,
		Transport:              "pubsub",
		ReceiverCreateSettings: params,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set up observer: %w", err)
	}

	var bucketNameFilter *regexp.Regexp
	if strings.TrimSpace(cfg.BucketNameFilter) != "" {
		bucketNameFilter, err = regexp.Compile(cfg.BucketNameFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to compile bucket_name_filter regex: %w", err)
		}
	}

	var objectKeyFilter *regexp.Regexp
	if strings.TrimSpace(cfg.ObjectKeyFilter) != "" {
		objectKeyFilter, err = regexp.Compile(cfg.ObjectKeyFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to compile object_key_filter regex: %w", err)
		}
	}

	return &logsReceiver{
		id:               id,
		cfg:              cfg,
		telemetry:        tel,
		metrics:          tb,
		next:             next,
		offsetStorage:    storageclient.NewNopStorage(),
		obsrecv:          obsrecv,
		bucketNameFilter: bucketNameFilter,
		objectKeyFilter:  objectKeyFilter,
	}, nil
}

func (r *logsReceiver) Start(_ context.Context, host component.Host) error {
	ctx := context.Background()

	// Build client options
	var opts []option.ClientOption
	if r.cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(r.cfg.CredentialsFile))
	}

	// Create low-level Pub/Sub subscriber client for synchronous Pull.
	subClient, err := subscriber.NewSubscriberClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create Pub/Sub subscriber client: %w", err)
	}
	r.subClient = subClient
	r.subscriptionPath = fmt.Sprintf("projects/%s/subscriptions/%s", r.cfg.ProjectID, r.cfg.SubscriptionID)

	// Create GCS client
	storageClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}
	r.storageClient = storageClient

	// Create offset storage
	if r.cfg.StorageID != nil {
		offsetStorage, err := storageclient.NewStorageClient(ctx, host, *r.cfg.StorageID, r.id, pipeline.SignalLogs)
		if err != nil {
			return fmt.Errorf("failed to create offset storage: %w", err)
		}
		r.offsetStorage = offsetStorage
	}

	// Initialize the worker pool now that all clients and storage are set up.
	r.workerPool.New = func() any {
		w := worker.New(
			r.telemetry,
			r.next,
			r.storageClient,
			r.obsrecv,
			r.cfg.MaxLogSize,
			r.cfg.MaxLogsEmitted,
			worker.WithTelemetryBuilder(r.metrics),
			worker.WithBucketNameFilter(r.bucketNameFilter),
			worker.WithObjectKeyFilter(r.objectKeyFilter),
			worker.WithSubscriberClient(r.subClient),
			worker.WithMaxExtension(r.cfg.MaxExtension),
		)
		w.SetOffsetStorage(r.offsetStorage)
		return w
	}

	r.startOnce.Do(func() {
		var runCtx context.Context
		runCtx, r.cancel = context.WithCancel(ctx)

		r.recent = newRecentTracker(r.cfg.DedupTTL)
		r.msgChan = make(chan *workerMessage, r.cfg.Workers)

		// Start eviction goroutine for the recent tracker.
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-runCtx.Done():
					return
				case <-ticker.C:
					r.recent.Evict()
				}
			}
		}()

		// Start fixed number of workers.
		for i := 0; i < r.cfg.Workers; i++ {
			r.workerWg.Add(1)
			go r.runWorker(runCtx)
		}

		// Start the poller (single goroutine).
		r.pollDone = make(chan struct{})
		go r.poll(runCtx, func() { close(r.pollDone) })

		r.telemetry.Logger.Info("GCS Pub/Sub event receiver started",
			zap.String("project_id", r.cfg.ProjectID),
			zap.String("subscription_id", r.cfg.SubscriptionID),
			zap.Int("workers", r.cfg.Workers))
	})

	return nil
}

// runWorker processes messages from the channel until the context is cancelled
// or the channel is closed. Analogous to awss3eventreceiver.runWorker.
func (r *logsReceiver) runWorker(ctx context.Context) {
	defer r.workerWg.Done()
	w := r.workerPool.Get().(*worker.Worker)

	r.telemetry.Logger.Debug("worker started")

	for {
		select {
		case <-ctx.Done():
			r.telemetry.Logger.Debug("worker stopping due to context cancellation")
			r.workerPool.Put(w)
			return
		case wm, ok := <-r.msgChan:
			if !ok {
				r.telemetry.Logger.Debug("worker stopping due to closed channel")
				r.workerPool.Put(w)
				return
			}
			r.telemetry.Logger.Debug("worker processing message", zap.String("message_id", wm.msg.MessageID))
			processed := w.ProcessMessage(ctx, wm.msg, r.subscriptionPath, func() {
				r.telemetry.Logger.Debug("worker finished processing message", zap.String("message_id", wm.msg.MessageID))
			})
			if processed {
				key := objectKeyFromAttrs(wm.msg.Attributes)
				r.recent.Mark(key)
			}
		}
	}
}

// poll pulls messages from Pub/Sub in a loop with a sleep when idle.
// Analogous to awss3eventreceiver.poll.
func (r *logsReceiver) poll(ctx context.Context, deferThis func()) {
	defer deferThis()
	for {
		select {
		case <-ctx.Done():
			r.telemetry.Logger.Info("context cancelled, stopping polling")
			return
		default:
		}

		n := r.pullMessages(ctx)
		if n == 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(r.cfg.PollInterval):
			}
		}
	}
}

// pullMessages issues a single Pull RPC, deduplicates the batch, and dispatches
// unique messages to the worker channel. Returns the number of messages dispatched.
func (r *logsReceiver) pullMessages(ctx context.Context) int {
	resp, err := r.subClient.Pull(ctx, &pubsubpb.PullRequest{
		Subscription: r.subscriptionPath,
		MaxMessages:  int32(r.cfg.Workers),
	})
	if err != nil {
		// Context cancellation is expected during shutdown.
		if ctx.Err() != nil {
			return 0
		}
		r.telemetry.Logger.Error("pull messages", zap.Error(err))
		r.metrics.GcseventFailures.Add(ctx, 1)
		return 0
	}
	if len(resp.ReceivedMessages) == 0 {
		return 0
	}
	r.telemetry.Logger.Debug("messages received", zap.Int("count", len(resp.ReceivedMessages)))

	// Wrap raw protobuf messages into worker.PullMessage.
	pulled := make([]*worker.PullMessage, 0, len(resp.ReceivedMessages))
	for _, rm := range resp.ReceivedMessages {
		pulled = append(pulled, &worker.PullMessage{
			AckID:      rm.AckId,
			MessageID:  rm.Message.MessageId,
			Data:       rm.Message.Data,
			Attributes: rm.Message.Attributes,
		})
	}

	// Layer 1: Batch-level deduplication by (bucket, object, generation).
	unique := r.batchDedup(ctx, pulled)

	// Layer 2: Cross-batch deduplication via the recent tracker.
	var dispatched int
	for _, msg := range unique {
		key := objectKeyFromAttrs(msg.Attributes)
		if r.recent.IsDuplicate(key) {
			r.ackMessage(ctx, msg.AckID)
			r.telemetry.Logger.Debug("skipping recently processed object",
				zap.String("bucket", key.Bucket),
				zap.String("object", key.Object),
				zap.String("generation", key.Generation))
			continue
		}

		select {
		case r.msgChan <- &workerMessage{msg: msg}:
			dispatched++
		case <-ctx.Done():
			return dispatched
		}
	}
	return dispatched
}

// batchDedup removes duplicate messages within a single Pull batch, keyed by
// (bucketId, objectId, objectGeneration). Duplicates are acked immediately.
func (r *logsReceiver) batchDedup(ctx context.Context, msgs []*worker.PullMessage) []*worker.PullMessage {
	seen := make(map[objectKey]struct{}, len(msgs))
	unique := make([]*worker.PullMessage, 0, len(msgs))
	for _, msg := range msgs {
		key := objectKeyFromAttrs(msg.Attributes)
		if _, dup := seen[key]; dup {
			r.ackMessage(ctx, msg.AckID)
			r.telemetry.Logger.Debug("acking duplicate within batch",
				zap.String("bucket", key.Bucket),
				zap.String("object", key.Object))
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, msg)
	}
	return unique
}

// ackMessage acknowledges a single message so Pub/Sub does not redeliver it.
func (r *logsReceiver) ackMessage(ctx context.Context, ackID string) {
	if err := r.subClient.Acknowledge(ctx, &pubsubpb.AcknowledgeRequest{
		Subscription: r.subscriptionPath,
		AckIds:       []string{ackID},
	}); err != nil {
		r.telemetry.Logger.Error("failed to ack message", zap.Error(err))
	}
}

func (r *logsReceiver) Shutdown(ctx context.Context) error {
	r.stopOnce.Do(func() {
		if r.cancel != nil {
			r.cancel()
		}
		if r.pollDone != nil {
			<-r.pollDone
		}
		if r.msgChan != nil {
			close(r.msgChan)
		}
		r.workerWg.Wait()
	})

	var errs []error

	if r.subClient != nil {
		if err := r.subClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Pub/Sub subscriber client: %w", err))
		}
	}

	if r.storageClient != nil {
		if err := r.storageClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close GCS client: %w", err))
		}
	}

	if err := r.offsetStorage.Close(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to close offset storage: %w", err))
	}

	return errors.Join(errs...)
}

// ---------------------------------------------------------------------------
// objectKey and recentTracker — deduplication helpers
// ---------------------------------------------------------------------------

// objectKey identifies a unique GCS object event. Using objectGeneration
// distinguishes true duplicate notifications from legitimate overwrites.
type objectKey struct {
	Bucket     string
	Object     string
	Generation string
}

func objectKeyFromAttrs(attrs map[string]string) objectKey {
	return objectKey{
		Bucket:     attrs[worker.AttrBucketID],
		Object:     attrs[worker.AttrObjectID],
		Generation: attrs[worker.AttrObjectGeneration],
	}
}

// recentTracker is a time-bounded set of recently-processed object keys.
// It catches sequential duplicate notifications that arrive in separate Pull
// batches (typically ~1 s apart due to GCS at-least-once delivery).
type recentTracker struct {
	mu   sync.Mutex
	seen map[objectKey]time.Time
	ttl  time.Duration
}

func newRecentTracker(ttl time.Duration) *recentTracker {
	return &recentTracker{
		seen: make(map[objectKey]time.Time),
		ttl:  ttl,
	}
}

// Mark records that the given key was processed just now.
func (rt *recentTracker) Mark(key objectKey) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.seen[key] = time.Now()
}

// IsDuplicate returns true if the key was processed within the TTL window.
func (rt *recentTracker) IsDuplicate(key objectKey) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if t, ok := rt.seen[key]; ok {
		if time.Since(t) < rt.ttl {
			return true
		}
		delete(rt.seen, key) // expired
	}
	return false
}

// Evict removes expired entries. Called periodically by a background goroutine.
func (rt *recentTracker) Evict() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	now := time.Now()
	for k, t := range rt.seen {
		if now.Sub(t) >= rt.ttl {
			delete(rt.seen, k)
		}
	}
}
