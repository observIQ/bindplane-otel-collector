package kandjireceiver

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

const (
	logStorageKey = "kandji_last_log_cursor"
)

// ------------------------------------------------------------
// Internal types
// ------------------------------------------------------------

// Shared through metrics scraper
type logStats struct {
	TotalRequests int64
	TotalErrors   int64
	TotalRecords  int64
	TotalBytes    int64
	TotalPages    int64
	LastLatencyMs float64
	mu            sync.Mutex
}

func (ls *logStats) add(s scrapeStats) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	ls.TotalRequests += s.CallCount
	ls.TotalErrors += s.ErrorCount
	ls.TotalRecords += s.RecordCount
	ls.TotalBytes += s.Bytes
	ls.TotalPages += s.Pages
	ls.LastLatencyMs = s.LatencyMs
}

// ------------------------------------------------------------
// Logs Receiver struct
// ------------------------------------------------------------

type kandjiLogsReceiver struct {
	settings      component.TelemetrySettings
	logger        *zap.Logger
	consumer      consumer.Logs
	cfg           *Config
	client        kClient
	storageClient storage.Client
	id            component.ID

	wg        *sync.WaitGroup
	cancel    context.CancelFunc
	mu        sync.Mutex
	cursor    *string // Kandji pagination cursor
	stats     *logStats
	startedAt time.Time
}

// ------------------------------------------------------------
// Constructor
// ------------------------------------------------------------

func newKandjiLogs(cfg *Config, settings component.Settings, consumer consumer.Logs) *kandjiLogsReceiver {
	return &kandjiLogsReceiver{
		settings: settings.TelemetrySettings,
		logger:   settings.Logger,
		consumer: consumer,
		cfg:      cfg,
		id:       settings.ID,
		wg:       &sync.WaitGroup{},
		stats:    &logStats{},
	}
}

// ------------------------------------------------------------
// Start + Shutdown
// ------------------------------------------------------------

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
	)

	// Storage checkpoint
	stc, err := adapter.GetStorageClient(ctx, host, l.cfg.StorageID, l.id)
	if err != nil {
		return fmt.Errorf("failed to get storage client: %w", err)
	}
	l.storageClient = stc
	l.loadCheckpoint(ctx)

	// Start polling loop
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

// ------------------------------------------------------------
// Poll Loop
// ------------------------------------------------------------

func (l *kandjiLogsReceiver) startPolling(ctx context.Context) error {
	t := time.NewTicker(l.cfg.LogPollInterval())
	l.wg.Add(1)

	go func() {
		defer l.wg.Done()
		defer t.Stop()

		for {
			select {
			case <-t.C:
				if err := l.poll(ctx); err != nil {
					l.logger.Error("polling error", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// ------------------------------------------------------------
// Poll Kandji Audit Logs
// ------------------------------------------------------------

func (l *kandjiLogsReceiver) poll(ctx context.Context) error {
	l.logger.Debug("polling kandji audit logs")

	params := map[string]any{
		"limit":   500,
		"sort_by": "-occurred_at",
	}

	if l.cursor != nil {
		params["cursor"] = *l.cursor
	}

	for {
		out, stats, next, err := l.fetchAuditPage(ctx, params)
		l.stats.add(stats)

		if err != nil {
			return err
		}

		if err := l.emitLogs(out); err != nil {
			l.logger.Error("consume logs failed", zap.Error(err))
		}

		if next == nil || *next == "" {
			break
		}

		nextVal := *next
		l.cursor = &nextVal
		params["cursor"] = nextVal
	}

	return l.checkpoint(ctx)
}

// ------------------------------------------------------------
// Fetch a single page of audit logs
// ------------------------------------------------------------

func (l *kandjiLogsReceiver) fetchAuditPage(
	ctx context.Context,
	params map[string]any,
) (*AuditEventsResponse, scrapeStats, *string, error) {

	ep := EPAuditEventsList
	spec := EndpointRegistry[ep]

	out := &AuditEventsResponse{}

	t0 := time.Now()
	err := l.client.CallAPI(ctx, ep, params, out)
	latency := time.Since(t0).Milliseconds()

	stats := scrapeStats{
		CallCount: 1,
		LatencyMs: float64(latency),
		Pages:     1,
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

// ------------------------------------------------------------
// Convert API response → OTel logs
// ------------------------------------------------------------

func (l *kandjiLogsReceiver) emitLogs(resp *AuditEventsResponse) error {
	if len(resp.Results) == 0 {
		return nil
	}

	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	scope := sl.LogRecords()

	attrs := rl.Resource().Attributes()
	attrs.PutStr("kandji.region", l.cfg.Region)
	attrs.PutStr("kandji.subdomain", l.cfg.SubDomain)
	attrs.PutStr("kandji.audit.type", "events")

	for _, ev := range resp.Results {
		lr := scope.AppendEmpty()

		body, _ := json.Marshal(ev)
		lr.Body().SetStr(string(body))

		ts, _ := time.Parse(time.RFC3339Nano, ev.OccurredAt)
		lr.SetTimestamp(pcommon.NewTimestampFromTime(ts))

		m := lr.Attributes()
		m.PutStr("id", ev.ID)
		m.PutStr("action", ev.Action)
		m.PutStr("actor.id", ev.ActorID)
		m.PutStr("actor.type", ev.ActorType)
		m.PutStr("target.id", ev.TargetID)
		m.PutStr("target.type", ev.TargetType)
		m.PutStr("target.component", ev.TargetComponent)
	}

	return l.consumer.ConsumeLogs(context.Background(), logs)
}

// ------------------------------------------------------------
// Checkpointing
// ------------------------------------------------------------

func (l *kandjiLogsReceiver) checkpoint(ctx context.Context) error {
	if l.cursor == nil {
		return nil
	}

	b, _ := json.Marshal(map[string]string{
		"cursor": *l.cursor,
	})
	return l.storageClient.Set(ctx, logStorageKey, b)
}

func (l *kandjiLogsReceiver) loadCheckpoint(ctx context.Context) {
	b, err := l.storageClient.Get(ctx, logStorageKey)
	if err != nil || b == nil {
		l.cursor = nil
		return
	}

	var v map[string]string
	if json.Unmarshal(b, &v) == nil {
		if cur, ok := v["cursor"]; ok {
			l.cursor = &cur
		}
	}
}

// ------------------------------------------------------------
// Exposed to scraper.go to publish metrics
// ------------------------------------------------------------

func (l *kandjiLogsReceiver) Stats() logStats {
	l.stats.mu.Lock()
	defer l.stats.mu.Unlock()
	return *l.stats
}
