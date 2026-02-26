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

	"cloud.google.com/go/pubsub"
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

	pubsubClient  *pubsub.Client
	storageClient *storage.Client
	sub           *pubsub.Subscription
	workerPool    sync.Pool

	receiveCancel context.CancelFunc
	receiveDone   chan struct{}

	offsetStorage storageclient.StorageClient

	bucketNameFilter *regexp.Regexp
	objectKeyFilter  *regexp.Regexp

	// inFlight tracks GCS objects currently being processed to deduplicate
	// concurrent Pub/Sub messages that reference the same object. GCS sends
	// notifications at-least-once, so two messages for the same OBJECT_FINALIZE
	// event can arrive within milliseconds of each other.
	inFlight sync.Map

	// observer for metrics about the receiver
	obsrecv *receiverhelper.ObsReport
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

	// Create Pub/Sub client
	pubsubClient, err := pubsub.NewClient(ctx, r.cfg.ProjectID, opts...)
	if err != nil {
		return fmt.Errorf("failed to create Pub/Sub client: %w", err)
	}
	r.pubsubClient = pubsubClient

	// Create GCS client
	storageClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}
	r.storageClient = storageClient

	// Get subscription
	r.sub = r.pubsubClient.Subscription(r.cfg.SubscriptionID)
	r.sub.ReceiveSettings = pubsub.ReceiveSettings{
		MaxOutstandingMessages: r.cfg.Workers,
		MaxExtension:           r.cfg.MaxExtension,
	}

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
		)
		w.SetOffsetStorage(r.offsetStorage)
		return w
	}

	r.startOnce.Do(func() {
		receiveCtx, receiveCancel := context.WithCancel(ctx)
		r.receiveCancel = receiveCancel
		r.receiveDone = make(chan struct{})

		go func() {
			defer close(r.receiveDone)
			err := r.sub.Receive(receiveCtx, r.handleMessage)
			if err != nil {
				r.telemetry.Logger.Error("Pub/Sub receive error", zap.Error(err))
			}
		}()

		r.telemetry.Logger.Info("GCS Pub/Sub event receiver started",
			zap.String("project_id", r.cfg.ProjectID),
			zap.String("subscription_id", r.cfg.SubscriptionID),
			zap.Int("workers", r.cfg.Workers))
	})

	return nil
}

func (r *logsReceiver) handleMessage(ctx context.Context, msg *pubsub.Message) {
	logger := r.telemetry.Logger.With(zap.String("message_id", msg.ID))

	// Deduplicate concurrent messages for the same GCS object. GCS delivers
	// OBJECT_FINALIZE notifications at-least-once, so two distinct Pub/Sub
	// messages can arrive for the same object within milliseconds. The second
	// message is acked immediately — the first handler will process the object.
	bucketID := msg.Attributes[worker.AttrBucketID]
	objectID := msg.Attributes[worker.AttrObjectID]
	if bucketID != "" && objectID != "" {
		inFlightKey := bucketID + "/" + objectID
		if _, alreadyInFlight := r.inFlight.LoadOrStore(inFlightKey, struct{}{}); alreadyInFlight {
			logger.Debug("skipping duplicate message: object already in flight",
				zap.String("bucket", bucketID),
				zap.String("object", objectID))
			msg.Ack()
			return
		}
		defer r.inFlight.Delete(inFlightKey)
	}

	w := r.workerPool.Get().(*worker.Worker)
	defer r.workerPool.Put(w)

	logger.Debug("processing message")
	w.ProcessMessage(ctx, msg)
}

func (r *logsReceiver) Shutdown(ctx context.Context) error {
	r.stopOnce.Do(func() {
		if r.receiveCancel != nil {
			r.receiveCancel()
		}
		if r.receiveDone != nil {
			<-r.receiveDone
		}
	})

	var errs []error

	if r.pubsubClient != nil {
		if err := r.pubsubClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Pub/Sub client: %w", err))
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
