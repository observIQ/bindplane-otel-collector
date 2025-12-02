package badgerextension

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/extension/badgerextension/internal/client"
)

type badgerExtension struct {
	logger *zap.Logger
	cfg    *Config
	client client.Client

	gcContextCancel context.CancelFunc
}

var _ storage.Extension = (*badgerExtension)(nil)

func newBadgerExtension(logger *zap.Logger, cfg *Config) *badgerExtension {
	// temporary require a file
	if cfg.File == nil {
		logger.Error("file config is required")
		return nil
	}

	client, err := client.NewClient(cfg.File.Path)
	if err != nil {
		logger.Error("failed to create badger client", zap.Error(err))
		return nil
	}

	return &badgerExtension{
		logger: logger,
		cfg:    cfg,
		client: client,
	}
}

// GetClient returns a storage client for an individual compoenent
func (b *badgerExtension) GetClient(ctx context.Context, kind component.Kind, ent component.ID, name string) (storage.Client, error) {
	// TODO look into bucketing for specific components
	var fullName string
	if name == "" {
		fullName = fmt.Sprintf("%s_%s_%s", kindString(kind), ent.Type(), ent.Name())
	} else {
		fullName = fmt.Sprintf("%s_%s_%s_%s", kindString(kind), ent.Type(), ent.Name(), name)
	}
	fullName = strings.ReplaceAll(fullName, " ", "")

	return b.client, nil
}

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
			b.client.RunValueLogGC(b.cfg.BlobGarbageCollection.DiscardRatio)
		}
	}
}

func (b *badgerExtension) Shutdown(ctx context.Context) error {
	if b.gcContextCancel != nil {
		b.gcContextCancel()
	}
	err := b.client.Close(ctx)
	if err != nil {
		return fmt.Errorf("failed to close badger client: %w", err)
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
