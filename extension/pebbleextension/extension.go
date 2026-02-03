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

package pebbleextension

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/extension/pebbleextension/internal/client"
)

type pebbleExtension struct {
	logger             *zap.Logger
	cfg                *Config
	clientsMutex       sync.RWMutex
	clients            map[string]client.Client
	compactionCancel   context.CancelFunc
	compactionDoneChan chan struct{}
}

// newPebbleExtension creates a new pebble storage extension
func newPebbleExtension(logger *zap.Logger, cfg *Config) (*pebbleExtension, error) {
	return &pebbleExtension{
		logger:       logger,
		cfg:          cfg,
		clientsMutex: sync.RWMutex{},
		clients:      make(map[string]client.Client),
	}, nil
}

// GetClient creates a new client for the specified component
func (p *pebbleExtension) GetClient(_ context.Context, kind component.Kind, ent component.ID, name string) (storage.Client, error) {
	var fullName string
	if name == "" {
		fullName = fmt.Sprintf("%s_%s_%s", kindString(kind), ent.Type(), ent.Name())
	} else {
		fullName = fmt.Sprintf("%s_%s_%s_%s", kindString(kind), ent.Type(), ent.Name(), name)
	}
	fullName = strings.ReplaceAll(fullName, " ", "")

	if p.cfg.Directory.PathPrefix != "" {
		fullName = fmt.Sprintf("%s_%s", p.cfg.Directory.PathPrefix, fullName)
	}

	if p.clients != nil {
		p.clientsMutex.RLock()
		client, ok := p.clients[fullName]
		p.clientsMutex.RUnlock()
		if ok {
			return client, nil
		}
	}

	client, err := p.createClientForComponent(p.cfg.Directory.Path, fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for component: %w", err)
	}
	p.clientsMutex.Lock()
	p.clients[fullName] = client
	p.clientsMutex.Unlock()
	return client, nil
}

// createClientForComponent creates a new client for the specified component
func (p *pebbleExtension) createClientForComponent(directory string, fullName string) (client.Client, error) {
	path := filepath.Join(directory, fullName)
	options := &client.Options{
		Sync: p.cfg.Sync,
	}
	if p.cfg.Cache != nil {
		options.CacheSize = p.cfg.Cache.Size
	}

	c, err := client.NewClient(path, p.logger.Named("client").Named(fullName), options)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for component: %w", err)
	}
	return c, nil
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

// Start initializes the pebble extension and starts a background task for compaction if configured
func (p *pebbleExtension) Start(_ context.Context, _ component.Host) error {
	p.compactionDoneChan = make(chan struct{})

	if p.cfg.Compaction != nil && p.cfg.Compaction.Interval > 0 {
		ctx, cancel := context.WithCancel(context.Background())
		p.compactionCancel = cancel
		go p.startCompaction(ctx)
	}

	return nil
}

// runCompaction runs the compaction process in a loop
func (p *pebbleExtension) startCompaction(ctx context.Context) {
	ticker := time.NewTicker(p.cfg.Compaction.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.compactionDoneChan:
			return
		case <-ticker.C:
			p.runCompaction()
		}
	}
}

func (p *pebbleExtension) runCompaction() {
	p.clientsMutex.RLock()
	clients := make([]client.Client, 0, len(p.clients))
	for _, c := range p.clients {
		clients = append(clients, c)
	}
	p.clientsMutex.RUnlock()

	// small optimization to avoid creating a wait group and iterating over clients if there are no clients
	if len(clients) == 0 {
		return
	}

	wg := sync.WaitGroup{}
	groupSize := p.cfg.Compaction.Concurrency

	for i := 0; i < len(clients); i += groupSize {
		end := i + groupSize
		if end > len(clients) {
			end = len(clients)
		}
		batch := clients[i:end]
		for _, c := range batch {
			wg.Add(1)
			go func(c client.Client) {
				defer wg.Done()
				if err := c.Compact(true); err != nil {
					p.logger.Warn("compaction failed", zap.Error(err))
				}
			}(c)
		}
		wg.Wait()
	}
}

// Shutdown closes all the clients and stops background compaction
func (p *pebbleExtension) Shutdown(ctx context.Context) error {
	if p.compactionCancel != nil {
		p.compactionCancel()
	}

	if p.compactionDoneChan != nil {
		close(p.compactionDoneChan)
	}

	var errs error
	p.clientsMutex.Lock()
	for _, client := range p.clients {
		errs = errors.Join(errs, client.Close(ctx))
	}
	p.clientsMutex.Unlock()
	return errs
}
