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
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/extension/badgerextension/internal/client"
)

type badgerExtension struct {
	logger *zap.Logger
	cfg    *Config

	gcContextCancel context.CancelFunc
	clientsMutex    sync.RWMutex
	clients         map[string]client.Client
}

var _ storage.Extension = (*badgerExtension)(nil)

func newBadgerExtension(logger *zap.Logger, cfg *Config) extension.Extension {
	return &badgerExtension{
		logger:       logger,
		cfg:          cfg,
		clients:      make(map[string]client.Client),
		clientsMutex: sync.RWMutex{},
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
	return &client.Options{
		SyncWrites: b.cfg.SyncWrites,
	}
}

// Start starts the badger storage extension
func (b *badgerExtension) Start(_ context.Context, _ component.Host) error {
	if b.cfg.BlobGarbageCollection != nil {
		// start background task for running blob garbage collection
		ctx, cancel := context.WithCancel(context.Background())
		b.gcContextCancel = cancel
		go b.runGC(ctx)
	}
	return nil
}

func (b *badgerExtension) runGC(ctx context.Context) {
	ticker := time.NewTicker(b.cfg.BlobGarbageCollection.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.clientsMutex.RLock()
			clients := make([]client.Client, 0, len(b.clients))
			for _, c := range b.clients {
				clients = append(clients, c)
			}
			b.clientsMutex.RUnlock()

			for _, c := range clients {
				go func(client client.Client) {
					if err := client.RunValueLogGC(b.cfg.BlobGarbageCollection.DiscardRatio); err != nil {
						b.logger.Warn("value log garbage collection failed", zap.Error(err))
					}
				}(c)
			}
		}
	}
}

// Shutdown shuts down the badger storage extension
func (b *badgerExtension) Shutdown(ctx context.Context) error {
	if b.gcContextCancel != nil {
		b.gcContextCancel()
	}

	b.clientsMutex.Lock()
	defer b.clientsMutex.Unlock()

	for _, c := range b.clients {
		if err := c.Close(ctx); err != nil {
			return fmt.Errorf("failed to close badger client: %w", err)
		}
	}
	return nil
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
