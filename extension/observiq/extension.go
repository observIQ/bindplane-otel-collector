package observiq

import (
	"context"
	"errors"
	"os"
	"runtime"
	"time"

	"github.com/jpillora/backoff"
	"github.com/observiq/observiq-collector/extension/observiq/message"
	"github.com/observiq/observiq-collector/extension/observiq/status"
	"github.com/observiq/observiq-collector/extension/observiq/websocket"
	"github.com/observiq/observiq-collector/internal/version"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Extension is the observiq extension for connecting to observiq cloud.
type Extension struct {
	config *Config
	logger *zap.Logger
	cancel context.CancelFunc
	group  *errgroup.Group
}

// createExtension creates an observiq extension from the supplied parameters.
func createExtension(ctx context.Context, params component.ExtensionCreateSettings, config config.Extension) (component.Extension, error) {
	c, ok := config.(*Config)
	if !ok {
		return nil, errors.New("invalid config type")
	}

	observiqExtension := Extension{
		config: c,
		logger: params.Logger,
	}

	return &observiqExtension, nil
}

// Start will start the observiq extension.
func (e *Extension) Start(ctx context.Context, host component.Host) error {
	pipeline := message.NewPipeline(e.config.BufferSize)
	cancelCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel

	group, groupCtx := errgroup.WithContext(cancelCtx)
	group.Go(func() error { return e.handleConnection(groupCtx, pipeline) })
	group.Go(func() error { return e.handleStatus(groupCtx, pipeline) })
	e.group = group

	return nil
}

// Shutdown will shutdown the observiq extension.
func (e *Extension) Shutdown(ctx context.Context) error {
	if e.cancel != nil {
		e.cancel()
	}

	if e.group != nil {
		_ = e.group.Wait()
	}
	return nil
}

// handleConnection will handle the connection to observiq cloud.
func (e *Extension) handleConnection(ctx context.Context, pipeline *message.Pipeline) error {
	headers := e.headers()
	backoff := backoff.Backoff{Max: e.config.MaxConnectBackoff}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff.Duration()):
			conn, err := websocket.Open(ctx, e.config.Endpoint, headers)
			if err != nil {
				e.logger.Error("Failed to open connection", zap.Error(err), zap.Float64("attempt", backoff.Attempt()))
				continue
			}

			e.logger.Info("Connected to observiq cloud")
			backoff.Reset()

			err = websocket.PumpWithTimeout(ctx, conn, pipeline, e.config.ReconnectInterval)
			switch err {
			case nil, context.DeadlineExceeded, context.Canceled:
			default:
				e.logger.Error("Unexpected connection error", zap.Error(err))
			}
		}
	}
}

// handleStatus will handle reporting status at the specified interval.
func (e *Extension) handleStatus(ctx context.Context, pipeline *message.Pipeline) error {
	ticker := time.NewTicker(e.config.StatusInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := status.Pump(pipeline)
			if err != nil {
				e.logger.Error("Status report failed", zap.Error(err))
			}
		}
	}
}

// headers returns the headers used to connect to observiq cloud.
func (e *Extension) headers() map[string][]string {
	hostname, _ := os.Hostname()
	return map[string][]string{
		"Secret-Key":       {e.config.SecretKey},
		"Agent-Id":         {e.config.AgentID},
		"Hostname":         {hostname},
		"Version":          {version.Version},
		"Operating-System": {runtime.GOOS},
		"Architecture":     {runtime.GOARCH},
	}
}
