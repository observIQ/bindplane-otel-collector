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

package badgerextension

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/extension/badgerextension/internal"
	"github.com/observiq/bindplane-otel-collector/extension/badgerextension/internal/client"
	"github.com/observiq/bindplane-otel-collector/extension/badgerextension/internal/metadata"
)

type badgerExtension struct {
	logger    *zap.Logger
	cfg       *Config
	component component.ID

	gcContextCancel   context.CancelFunc
	otelAttrs         metric.MeasurementOption
	clientsMutex      sync.RWMutex
	clients           map[string]client.Client
	doneChan          chan struct{}
	telemetrySettings component.TelemetrySettings

	mb *metadata.TelemetryBuilder
}

var _ storage.Extension = (*badgerExtension)(nil)

func newBadgerExtension(logger *zap.Logger, cfg *Config, telemetrySettings component.TelemetrySettings, component component.ID) extension.Extension {
	return &badgerExtension{
		logger:    logger,
		cfg:       cfg,
		component: component,
		otelAttrs: metric.WithAttributeSet(attribute.NewSet(
			attribute.String(internal.ExtensionAttribute, component.String()),
		)),
		clients:           make(map[string]client.Client),
		clientsMutex:      sync.RWMutex{},
		telemetrySettings: telemetrySettings,
	}
}

// GetClient returns a storage client for an individual component
func (b *badgerExtension) GetClient(_ context.Context, kind component.Kind, ent component.ID, name string) (storage.Client, error) {
	var fullName string
	if name == "" {
		fullName = fmt.Sprintf("%s_%s_%s", kindString(kind), ent.Type(), ent.Name())
	} else {
		fullName = fmt.Sprintf("%s_%s_%s_%s", kindString(kind), ent.Type(), ent.Name(), name)
	}
	fullName = strings.ReplaceAll(fullName, " ", "")

	if b.cfg.StoragePrefix != "" {
		fullName = fmt.Sprintf("%s_%s", b.cfg.StoragePrefix, fullName)
	}

	if b.clients != nil {
		b.clientsMutex.RLock()
		client, ok := b.clients[fullName]
		b.clientsMutex.RUnlock()
		if ok {
			return client, nil
		}
	}

	client, err := b.createClientForComponent(b.cfg.Directory.Path, fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client for %s: %w", fullName, err)
	}
	b.clientsMutex.Lock()
	b.clients[fullName] = client
	b.clientsMutex.Unlock()
	return client, nil
}

func (b *badgerExtension) createClientForComponent(directory string, fullName string) (client.Client, error) {
	fullPath := filepath.Join(directory, fullName)
	return client.NewClient(fullPath, b.clientOptions())
}

// clientOptions returns the options for the badger storage client to be used during client creation
func (b *badgerExtension) clientOptions() *client.Options {
	opts := &client.Options{
		SyncWrites: b.cfg.SyncWrites,
	}
	return opts
}

// Start starts the badger storage extension
func (b *badgerExtension) Start(_ context.Context, _ component.Host) error {
	b.doneChan = make(chan struct{})

	if b.cfg.Telemetry != nil && b.cfg.Telemetry.Enabled {
		go b.monitor(context.Background())
	}

	if b.cfg.BlobGarbageCollection != nil && b.cfg.BlobGarbageCollection.Interval > 0 {
		// start background task for running blob garbage collection
		ctx, cancel := context.WithCancel(context.Background())
		b.gcContextCancel = cancel
		go b.runGC(ctx)
	}

	return nil
}

// runGC runs the garbage collection process in a loop
func (b *badgerExtension) runGC(ctx context.Context) {
	ticker := time.NewTicker(b.cfg.BlobGarbageCollection.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-b.doneChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			beginning := b.startGCOperation()

			b.clientsMutex.RLock()
			clients := make([]client.Client, 0, len(b.clients))
			for _, c := range b.clients {
				clients = append(clients, c)
			}
			b.clientsMutex.RUnlock()

			// small optimization to avoid creating a wait group and iterating over clients if there are no clients
			if len(clients) == 0 {
				continue
			}

			wg := sync.WaitGroup{}
			wg.Add(len(clients))

			for _, c := range clients {
				go func(client client.Client) {
					defer wg.Done()
					if err := client.RunValueLogGC(b.cfg.BlobGarbageCollection.DiscardRatio); err != nil {
						b.logger.Warn("value log garbage collection failed", zap.Error(err))
					}
				}(c)
			}
			wg.Wait()
			b.finishGCOperation(ctx, beginning)
		}
	}
}

func (b *badgerExtension) monitor(ctx context.Context) {
	mb, err := metadata.NewTelemetryBuilder(b.telemetrySettings)
	if err != nil {
		b.logger.Error("failed to create telemetry builder", zap.Error(err))
		return
	}
	b.mb = mb
	defer mb.Shutdown()

	ticker := time.NewTicker(b.cfg.Telemetry.UpdateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-b.doneChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.clientsMutex.RLock()
			clientCount := len(b.clients)
			// make a shallow copy of the clients under lock so we can process them outside of the lock
			clients := make([]client.Client, 0, clientCount)
			for _, c := range b.clients {
				clients = append(clients, c)
			}
			b.clientsMutex.RUnlock()

			b.mb.ExtensionStorageClientCount.Record(ctx, int64(clientCount))
			// process clients outside of lock to avoid blocking other operations
			for _, c := range clients {
				diskUsage, err := c.GetDiskUsage()
				if err != nil {
					b.logger.Error("failed to get disk usage", zap.Error(err))
					continue
				}

				b.mb.ExtensionStorageClientDiskUsageBytes.Record(ctx, diskUsage.LSMUsage,
					b.otelAttrs,
					metric.WithAttributes(attribute.String(internal.StorageTypeAttribute, internal.StorageTypeLSM)))
				b.mb.ExtensionStorageClientDiskUsageBytes.Record(ctx, diskUsage.ValueLogUsage,
					b.otelAttrs,
					metric.WithAttributes(attribute.String(internal.StorageTypeAttribute, internal.StorageTypeValueLog)))

				opCounts := c.GetOperationCounts()
				b.mb.ExtensionStorageClientOperationsCount.Add(ctx, opCounts.Get,
					b.otelAttrs,
					metric.WithAttributes(attribute.String(internal.OperationTypeAttribute, internal.OperationTypeGet)))
				b.mb.ExtensionStorageClientOperationsCount.Add(ctx, opCounts.Set,
					b.otelAttrs,
					metric.WithAttributes(attribute.String(internal.OperationTypeAttribute, internal.OperationTypeSet)))
				b.mb.ExtensionStorageClientOperationsCount.Add(ctx, opCounts.Delete,
					b.otelAttrs,
					metric.WithAttributes(attribute.String(internal.OperationTypeAttribute, internal.OperationTypeDelete)))
			}
		}
	}
}

// startGCOperation records the beginning of a garbage collection operation
func (b *badgerExtension) startGCOperation() *time.Time {
	if b.cfg.Telemetry == nil || !b.cfg.Telemetry.Enabled {
		return nil
	}
	now := time.Now()
	return &now
}

// finishGCOperation records the duration of a garbage collection operation
func (b *badgerExtension) finishGCOperation(ctx context.Context, beginning *time.Time) {
	if beginning == nil {
		return
	}
	duration := time.Since(*beginning)
	b.mb.ExtensionStorageClientGcDurationMilliseconds.Record(ctx, duration.Milliseconds())
}

// Shutdown shuts down the badger storage extension
func (b *badgerExtension) Shutdown(ctx context.Context) error {
	if b.gcContextCancel != nil {
		b.gcContextCancel()
	}
	if b.doneChan != nil {
		close(b.doneChan)
	}

	b.clientsMutex.Lock()
	defer b.clientsMutex.Unlock()

	var shutdownErrors error
	for _, c := range b.clients {
		if err := c.Close(ctx); err != nil {
			shutdownErrors = errors.Join(shutdownErrors, fmt.Errorf("failed to close badger client: %w", err))
		}
	}
	return shutdownErrors
}

func kindString(k component.Kind) string {
	switch k {
	case component.KindReceiver:
		return "receiver"
	case component.KindProcessor:
		return "processor"
	case component.KindExporter:
		return "exporter"
	case component.KindExtension:
		return "extension"
	case component.KindConnector:
		return "connector"
	default:
		return "other" // not expected
	}
}
