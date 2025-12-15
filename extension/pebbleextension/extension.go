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

	"github.com/observiq/bindplane-otel-collector/extension/pebbleextension/internal/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"
)

type pebbleExtension struct {
	logger       *zap.Logger
	cfg          *Config
	clientsMutex sync.RWMutex
	clients      map[string]client.Client
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

	c, err := client.NewClient(path, options)

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

// Start is a no-op for the pebble extension as most of th work is done in the GetClient method
func (p *pebbleExtension) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown closes all the clients
func (p *pebbleExtension) Shutdown(ctx context.Context) error {
	var errs error
	p.clientsMutex.Lock()
	for _, client := range p.clients {
		errs = errors.Join(errs, client.Close(ctx))
	}
	p.clientsMutex.Unlock()
	return errs
}
