package pebbleextension

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/observiq/bindplane-otel-collector/extension/pebbleextension/internal/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"
)

type pebbleExtension struct {
	logger  *zap.Logger
	cfg     *Config
	clients map[string]client.Client
}

// newPebbleExtension creates a new pebble storage extension
func newPebbleExtension(logger *zap.Logger, cfg *Config) (*pebbleExtension, error) {
	return &pebbleExtension{
		logger:  logger,
		cfg:     cfg,
		clients: make(map[string]client.Client),
	}, nil
}

func (p *pebbleExtension) GetClient(ctx context.Context, kind component.Kind, ent component.ID, name string) (storage.Client, error) {
	var fullName string
	if name == "" {
		fullName = fmt.Sprintf("%s_%s_%s", kindString(kind), ent.Type(), ent.Name())
	} else {
		fullName = fmt.Sprintf("%s_%s_%s_%s", kindString(kind), ent.Type(), ent.Name(), name)
	}
	fullName = strings.ReplaceAll(fullName, " ", "")

	if p.clients != nil {
		if client, ok := p.clients[fullName]; ok {
			return client, nil
		}
	}

	client, err := p.createClientForComponent(p.cfg.File.Path, fullName)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for component: %w", err)
	}
	p.clients[fullName] = client
	return client, nil
}

func (p *pebbleExtension) createClientForComponent(directory string, fullName string) (client.Client, error) {
	path := filepath.Join(directory, fullName)
	return client.NewClient(path, &client.ClientOptions{
		Sync: p.cfg.Sync,
	})
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

func (p *pebbleExtension) Start(ctx context.Context, host component.Host) error {
	// TODO: make any pre-requisites for starting the extension
	// for _, client := range p.clients {
	// 	client.Start(ctx, host)
	// }

	return nil
}

func (p *pebbleExtension) Shutdown(ctx context.Context) error {
	var errs error = nil
	for _, client := range p.clients {
		errs = errors.Join(errs, client.Close(ctx))
	}
	return errs
}
