package kandjireceiver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

// -------------------------------------------------------------------
// Constants
// -------------------------------------------------------------------

const (
	logStorageKeyPrefix = "kandji_cursor_"
)

// -------------------------------------------------------------------
// Logs Receiver
// -------------------------------------------------------------------

type kandjiLogsReceiver struct {
	settings      component.TelemetrySettings
	logger        *zap.Logger
	consumer      consumer.Logs
	cfg           *Config
	client        kClient
	storageClient storage.Client
	id            component.ID

	wg     *sync.WaitGroup
	cancel context.CancelFunc

	mu      sync.Mutex
	cursors map[KandjiEndpoint]*string // checkpointed per-endpoint
}

// -------------------------------------------------------------------
// Constructor
// -------------------------------------------------------------------

func newKandjiLogs(
	cfg *Config,
	settings receiver.Settings,
	consumer consumer.Logs,
) *kandjiLogsReceiver {
	return &kandjiLogsReceiver{
		settings: settings.TelemetrySettings,
		logger:   settings.Logger,
		consumer: consumer,
		cfg:      cfg,
		id:       settings.ID,
		wg:       &sync.WaitGroup{},
		cursors:  map[KandjiEndpoint]*string{},
	}
}

// -------------------------------------------------------------------
// Start + Shutdown
// -------------------------------------------------------------------

func (l *kandjiLogsReceiver) Start(ctx context.Context, host component.Host) error {

	httpClient, err := l.cfg.ClientConfig.ToClient(ctx, host, l.settings)
	if err != nil {
		l.logger.Error("failed to create HTTP client", zap.Error(err))
		return err
	}

	l.client = newKandjiClient(
		httpClient,
		l.cfg.SubDomain,
		l.cfg.Region,
		l.cfg.ApiKey,
		l.cfg.BaseHost,
		l.logger,
	)

	// Storage checkpoint (StorageID should be on your Config, like m365)
	stc, err := adapter.GetStorageClient(ctx, host, l.cfg.StorageID, l.id)
	if err != nil {
		return fmt.Errorf("failed to get storage client: %w", err)
	}
	l.storageClient = stc

	l.loadCheckpoint(ctx)

	// Poll loop
	cancelCtx, cancel := context.WithCancel(ctx)
	l.cancel = cancel

	l.logger.Info("kandji logs receiver started")
	return l.startPolling(cancelCtx)
}

func (l *kandjiLogsReceiver) Shutdown(ctx context.Context) error {
	l.logger.Info("shutting down kandji logs receiver")

	if l.cancel != nil {
		l.cancel()
	}
	l.wg.Wait()

	_ = l.checkpoint(ctx)
	return l.storageClient.Close(ctx)
}

// -------------------------------------------------------------------
// Poll Loop
// -------------------------------------------------------------------

func (l *kandjiLogsReceiver) startPolling(ctx context.Context) error {
	// Assuming you've added something like:
	//   Logs struct { PollInterval time.Duration `mapstructure:"poll_interval"` }
	t := time.NewTicker(l.cfg.Logs.PollInterval)
	l.wg.Add(1)

	// Do an initial poll immediately, then continue with ticker
	l.logger.Info("performing initial poll")
	if err := l.pollAll(ctx); err != nil {
		l.logger.Error("initial polling error", zap.Error(err))
	}

	go func() {
		defer l.wg.Done()
		defer t.Stop()

		for {
			select {
			case <-t.C:
				l.logger.Info("ticker fired, calling pollAll")
				if err := l.pollAll(ctx); err != nil {
					l.logger.Error("polling error", zap.Error(err))
				}
			case <-ctx.Done():
				l.logger.Info("polling context cancelled")
				return
			}
		}
	}()

	return nil
}

// -------------------------------------------------------------------
// Poll ALL log endpoints from registry
// -------------------------------------------------------------------

func (l *kandjiLogsReceiver) pollAll(ctx context.Context) error {
	l.logger.Info("pollAll started",
		zap.Int("configured_endpoints", len(l.cfg.EndpointParams)),
	)

	if l.consumer == nil {
		l.logger.Error("consumer is nil - logs cannot be emitted")
		return fmt.Errorf("consumer is nil")
	}

	// Poll only endpoints configured in endpoint_params
	// Each endpoint gets its configured query parameters
	foundCount := 0
	for epStr, params := range l.cfg.EndpointParams {
		ep := KandjiEndpoint(epStr)
		spec, ok := EndpointRegistry[ep]
		if !ok {
			l.logger.Warn("endpoint in endpoint_params not found in registry",
				zap.String("endpoint", epStr),
			)
			continue
		}

		// Only poll endpoints that return AuditEventsResponse
		_, isValueType := spec.ResponseType.(AuditEventsResponse)
		_, isPtrType := spec.ResponseType.(*AuditEventsResponse)
		if !isValueType && !isPtrType {
			l.logger.Debug("skipping endpoint - not AuditEventsResponse",
				zap.String("endpoint", epStr),
			)
			continue
		}

		foundCount++
		l.logger.Info("polling log endpoint",
			zap.String("endpoint", string(ep)),
			zap.Any("params", params),
		)

		if err := l.pollEndpoint(ctx, ep, spec, params); err != nil {
			l.logger.Error("failed polling kandji logs",
				zap.String("endpoint", string(ep)),
				zap.Error(err),
			)
		}
	}

	l.logger.Info("pollAll completed",
		zap.Int("found_log_endpoints", foundCount),
	)

	return l.checkpoint(ctx)
}

// -------------------------------------------------------------------
// Poll a single endpoint
// -------------------------------------------------------------------

func (l *kandjiLogsReceiver) pollEndpoint(
	ctx context.Context,
	ep KandjiEndpoint,
	spec EndpointSpec,
	endpointParams map[string]any,
) error {

	// Start with configured endpoint_params from config
	// These are the query parameters for the HTTP request
	// Note: ValidateParams will sanitize string values, but we do basic validation here too
	params := make(map[string]any)
	for k, v := range endpointParams {
		// Basic sanitization: trim string values
		if strVal, ok := v.(string); ok {
			params[k] = strings.TrimSpace(strVal)
		} else {
			params[k] = v
		}
	}

	// Apply defaults for common params if not provided
	if _, hasLimit := params["limit"]; !hasLimit {
		for _, p := range spec.Params {
			if p.Name == "limit" {
				params["limit"] = 500 // default reasonable max page size
				break
			}
		}
	}
	if _, hasSortBy := params["sort_by"]; !hasSortBy {
		for _, p := range spec.Params {
			if p.Name == "sort_by" {
				params["sort_by"] = "-occurred_at" // default newest first
				break
			}
		}
	}

	// Insert cursor for continuation
	l.mu.Lock()
	if cur, ok := l.cursors[ep]; ok && cur != nil {
		if normalized := normalizeCursor(cur); normalized != nil {
			l.cursors[ep] = normalized
			params["cursor"] = *normalized
		} else {
			delete(l.cursors, ep)
		}
	}
	l.mu.Unlock()

	page := 1
	for {
		l.logger.Info("kandji logs request",
			zap.String("endpoint", string(ep)),
			zap.Int("page", page),
			zap.Any("params", params),
		)

		resp, stats, next, err := l.fetchPage(ctx, ep, params)

		if err != nil {
			l.logger.Error("fetchPage failed", zap.Error(err))
			return err
		}

		l.logger.Info("calling emitLogs",
			zap.String("endpoint", string(ep)),
			zap.Int("num_records", len(resp.Results)),
		)

		if err := l.emitLogs(ep, resp); err != nil {
			l.logger.Error("log consumption failed", zap.Error(err))
		} else {
			l.logger.Info("emitLogs completed successfully",
				zap.String("endpoint", string(ep)),
				zap.Int("num_records", len(resp.Results)),
			)
		}

		nextCursor := normalizeCursor(next)
		if nextCursor == nil {
			break
		}

		l.logger.Info("kandji logs response",
			zap.String("endpoint", string(ep)),
			zap.Int("page", page),
			zap.Int64("records", int64(len(resp.Results))),
			zap.Float64("latency_ms", stats.LatencyMs),
			zap.Int64("bytes", stats.Bytes),
			zap.Stringp("next_cursor", nextCursor),
		)

		l.mu.Lock()
		l.cursors[ep] = nextCursor
		l.mu.Unlock()

		params["cursor"] = *nextCursor
		page++
	}

	return nil
}

// -------------------------------------------------------------------
// Fetch a single page for ANY log endpoint
// -------------------------------------------------------------------

func (l *kandjiLogsReceiver) fetchPage(
	ctx context.Context,
	ep KandjiEndpoint,
	params map[string]any,
) (*AuditEventsResponse, scrapeStats, *string, error) {

	out := &AuditEventsResponse{}

	t0 := time.Now()
	statusCode, err := l.client.CallAPI(ctx, ep, params, out)
	latency := time.Since(t0).Milliseconds()

	stats := scrapeStats{
		CallCount:  1,
		LatencyMs:  float64(latency),
		Pages:      1,
		HTTPStatus: statusCode,
	}

	if err != nil {
		stats.ErrorCount = 1
		stats.Status = "error"
		return out, stats, nil, err
	}

	stats.Status = "success"
	stats.RecordCount = int64(len(out.Results))

	b, _ := json.Marshal(out.Results)
	stats.Bytes = int64(len(b))

	return out, stats, out.Next, nil
}

// -------------------------------------------------------------------
// Emit logs
// -------------------------------------------------------------------

func (l *kandjiLogsReceiver) emitLogs(
	ep KandjiEndpoint,
	resp *AuditEventsResponse,
) error {

	l.logger.Info("emitLogs called",
		zap.String("endpoint", string(ep)),
		zap.Int("num_results", len(resp.Results)),
		zap.Bool("consumer_is_nil", l.consumer == nil),
	)

	if l.consumer == nil {
		l.logger.Error("consumer is nil in emitLogs - cannot emit logs")
		return fmt.Errorf("consumer is nil")
	}

	if len(resp.Results) == 0 {
		l.logger.Info("emitLogs: no results to emit")
		return nil
	}

	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()

	// Set scope name and version (required by many exporters)
	sl.Scope().SetName("kandji")
	sl.Scope().SetVersion("1.0.0")

	scope := sl.LogRecords()

	attrs := rl.Resource().Attributes()
	attrs.PutStr("kandji.region", l.cfg.Region)
	attrs.PutStr("kandji.subdomain", l.cfg.SubDomain)
	attrs.PutStr("kandji.endpoint", string(ep))
	attrs.PutStr("kandji.log_type", "audit_events")

	for _, ev := range resp.Results {
		lr := scope.AppendEmpty()

		// full event JSON as body (easy for downstream search)
		body, err := json.Marshal(ev)
		if err != nil {
			l.logger.Warn("failed to marshal event to JSON", zap.Error(err))
			continue
		}

		// Set body - using the correct pdata API
		bodyVal := lr.Body()
		bodyVal.SetStr(string(body))

		ts, err := time.Parse(time.RFC3339Nano, ev.OccurredAt)
		if err != nil {
			// fall back to "now" if parse fails
			l.logger.Warn("failed to parse occurred_at, using current time",
				zap.String("occurred_at", ev.OccurredAt),
				zap.Error(err),
			)
			now := time.Now().UTC()
			lr.SetTimestamp(pcommon.NewTimestampFromTime(now))
			lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(now))
		} else {
			lr.SetTimestamp(pcommon.NewTimestampFromTime(ts))
			lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now().UTC()))
		}

		m := lr.Attributes()
		m.PutStr("id", ev.ID)
		m.PutStr("action", ev.Action)
		m.PutStr("actor.id", ev.ActorID)
		m.PutStr("actor.type", ev.ActorType)
		m.PutStr("target.id", ev.TargetID)
		m.PutStr("target.type", ev.TargetType)
		m.PutStr("target.component", ev.TargetComponent)
	}

	totalRecords := scope.Len()
	l.logger.Info("calling consumer.ConsumeLogs",
		zap.String("endpoint", string(ep)),
		zap.Int("total_log_records", totalRecords),
		zap.Bool("consumer_is_nil", l.consumer == nil),
		zap.Int("resource_logs_count", logs.ResourceLogs().Len()),
	)

	if totalRecords == 0 {
		l.logger.Warn("no log records to emit")
		return nil
	}

	err := l.consumer.ConsumeLogs(context.Background(), logs)
	if err != nil {
		l.logger.Error("consumer.ConsumeLogs returned error", zap.Error(err))
	} else {
		l.logger.Info("consumer.ConsumeLogs completed successfully",
			zap.String("endpoint", string(ep)),
			zap.Int("total_log_records", scope.Len()),
		)
	}

	return err
}

// -------------------------------------------------------------------
// Checkpoint
// -------------------------------------------------------------------

func (l *kandjiLogsReceiver) checkpoint(ctx context.Context) error {
	for ep, cur := range l.cursors {
		cur = normalizeCursor(cur)
		if cur == nil {
			continue
		}

		data, _ := json.Marshal(map[string]string{
			"cursor": *cur,
		})

		key := logStorageKeyPrefix + string(ep)
		if err := l.storageClient.Set(ctx, key, data); err != nil {
			l.logger.Error("failed to write cursor checkpoint",
				zap.String("endpoint", string(ep)),
				zap.Error(err),
			)
		} else {
			l.logger.Debug("cursor checkpoint saved",
				zap.String("endpoint", string(ep)),
				zap.String("cursor", *cur),
			)
		}
	}
	return nil
}

func (l *kandjiLogsReceiver) loadCheckpoint(ctx context.Context) {
	for ep := range EndpointRegistry {
		key := logStorageKeyPrefix + string(ep)

		b, err := l.storageClient.Get(ctx, key)
		if err != nil || b == nil {
			continue
		}

		var v map[string]string
		if json.Unmarshal(b, &v) == nil {
			if cur, ok := v["cursor"]; ok {
				c := cur
				if normalized := normalizeCursor(&c); normalized != nil {
					l.cursors[ep] = normalized
					l.logger.Debug("cursor checkpoint loaded",
						zap.String("endpoint", string(ep)),
						zap.String("cursor", *normalized),
					)
				}
			}
		}
	}
}
