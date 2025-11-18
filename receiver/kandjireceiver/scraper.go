package kandjireceiver // import "github.com/observiq/bindplane-otel-collector/receiver/kandjireceiver"

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/receiver/kandjireceiver/internal/metadata"
)

type kClient interface {
	CallAPI(ctx context.Context, ep KandjiEndpoint, params map[string]any, out any) error
	Shutdown() error
}

type kandjiScraper struct {
	settings component.TelemetrySettings
	logger   *zap.Logger
	cfg      *Config

	client kClient
	rb     *metadata.ResourceBuilder
	mb     *metadata.MetricsBuilder
}

func newKandjiScraper(
	settings receiver.Settings,
	cfg *Config,
) *kandjiScraper {
	return &kandjiScraper{
		settings: settings.TelemetrySettings,
		logger:   settings.Logger,
		cfg:      cfg,
		rb:       metadata.NewResourceBuilder(cfg.MetricsBuilderConfig.ResourceAttributes),
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
	}
}

func (k *kandjiScraper) start(ctx context.Context, host component.Host) error {

	httpClient, err := k.cfg.ClientConfig.ToClient(ctx, host, k.settings)
	if err != nil {
		k.logger.Error("error creating HTTP client", zap.Error(err))
		return err
	}

	k.client = newKandjiClient(
		httpClient,
		k.cfg.SubDomain,
		k.cfg.Region,
		k.cfg.ApiKey,
		k.cfg.BaseHost,
	)

	k.logger.Info("Kandji scraper started",
		zap.String("subdomain", k.cfg.SubDomain),
		zap.String("region", k.cfg.Region),
		zap.String("base_host", k.cfg.BaseHost))

	return nil
}

// ----------------------------
// Scrape entrypoint
// ----------------------------

func (k *kandjiScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {

	scrapeStart := time.Now()
	md := pmetric.NewMetrics()

	k.logger.Debug("scrape cycle started",
		zap.Int("num_endpoints", len(k.cfg.EndpointParams)))

	for epStr, paramCfg := range k.cfg.EndpointParams {

		ep := KandjiEndpoint(epStr)
		spec, ok := EndpointRegistry[ep]
		if !ok {
			k.logger.Error("unknown endpoint in configuration", zap.String("endpoint", epStr))
			continue
		}

		if spec.SupportsPagination {
			if err := k.scrapePaginated(ctx, md, ep, paramCfg); err != nil {
				k.logger.Error("scrape paginated error",
					zap.String("endpoint", epStr), zap.Error(err))
				return pmetric.Metrics{}, err
			}
		} else {
			if err := k.scrapeSingle(ctx, md, ep, paramCfg); err != nil {
				k.logger.Error("scrape error",
					zap.String("endpoint", epStr), zap.Error(err))
				return pmetric.Metrics{}, err
			}
		}
	}

	k.rb.SetKandjiRegion(k.cfg.Region)
	k.rb.SetKandjiSubdomain(k.cfg.SubDomain)

	// Emit metrics with resource attributes
	result := k.mb.Emit(metadata.WithResource(k.rb.Emit()))

	k.logger.Debug("scrape cycle completed",
		zap.Duration("duration", time.Since(scrapeStart)))

	return result, nil
}

// ----------------------------
// Page executor (DRY)
// ----------------------------

func (k *kandjiScraper) fetchPage(
	ctx context.Context,
	ep KandjiEndpoint,
	params map[string]any,
) (out any, stats scrapeStats, nextCursor *string, err error) {

	spec := EndpointRegistry[ep]
	out = cloneResponseType(spec.ResponseType)

	t0 := time.Now()
	err = k.client.CallAPI(ctx, ep, params, out)
	latency := time.Since(t0)

	stats = scrapeStats{
		CallCount:        1,
		LatencyMs:        float64(latency.Milliseconds()),
		ScrapeDurationMs: float64(latency.Milliseconds()),
	}

	if err != nil {
		stats.ErrorCount = 1
		stats.Status = "error"
		stats.ErrorReason = "http_error"
		return out, stats, nil, err
	}

	stats.Status = "success"
	stats.RecordCount = getRecordCount(out)
	stats.Pages = 1

	nextCursor = extractNextCursor(out)
	return out, stats, nextCursor, nil
}

// ----------------------------
// Single-page scrape
// ----------------------------

func (k *kandjiScraper) scrapeSingle(
	ctx context.Context,
	md pmetric.Metrics,
	ep KandjiEndpoint,
	params map[string]any,
) error {

	_, stats, _, err := k.fetchPage(ctx, ep, params)
	k.recordMetrics(ep, stats)
	return err
}

func (k *kandjiScraper) scrapePaginated(
	ctx context.Context,
	md pmetric.Metrics,
	ep KandjiEndpoint,
	params map[string]any,
) error {

	var cursor *string

	for {
		pageParams := buildPageParams(params, cursor)

		_, stats, next, err := k.fetchPage(ctx, ep, pageParams)
		if err != nil {
			k.recordMetrics(ep, stats)
			return err
		}

		k.recordMetrics(ep, stats)

		if next == nil || *next == "" {
			return nil
		}

		cursor = next
	}
}

func cloneResponseType(rt any) any {
	switch rt.(type) {
	case AuditEventsResponse:
		return &AuditEventsResponse{}
	default:
		panic(fmt.Sprintf("unsupported ResponseType: %T", rt))
	}
}

func getRecordCount(out any) int64 {
	switch v := out.(type) {
	case *AuditEventsResponse:
		return int64(len(v.Results))
	default:
		return 0
	}
}

func extractNextCursor(out any) *string {
	switch v := out.(type) {
	case *AuditEventsResponse:
		return v.Next
	default:
		return nil
	}
}

func (k *kandjiScraper) recordMetrics(ep KandjiEndpoint, stats scrapeStats) {
	now := pcommon.NewTimestampFromTime(time.Now())

	statusAttr := metadata.MapAttributeStatus[stats.Status]
	reasonAttr := metadata.MapAttributeReason[stats.ErrorReason]

	k.mb.RecordKandjiAPICallsDataPoint(
		now,
		stats.CallCount,
		string(ep),
		statusAttr,
		string(stats.HTTPStatus),
	)

	if stats.ErrorCount > 0 {
		k.mb.RecordKandjiAPIErrorsDataPoint(
			now,
			stats.ErrorCount,
			string(ep),
			reasonAttr,
		)
	}

	k.mb.RecordKandjiAPILatencyDataPoint(
		now,
		stats.LatencyMs,
		string(ep),
	)

	k.mb.RecordKandjiRecordsReceivedDataPoint(
		now,
		stats.RecordCount,
		string(ep),
	)

	k.mb.RecordKandjiDataBytesDataPoint(
		now,
		stats.Bytes,
		string(ep),
	)

	k.mb.RecordKandjiPaginationPagesDataPoint(
		now,
		stats.Pages,
		string(ep),
	)

	k.mb.RecordKandjiScrapeDurationDataPoint(
		now,
		stats.ScrapeDurationMs,
	)
}

func (k *kandjiScraper) shutdown(_ context.Context) error {
	if k.client != nil {
		return k.client.Shutdown()
	}
	return nil
}

// buildPageParams creates a new param map for each page without mutating the original.
func buildPageParams(base map[string]any, cursor *string) map[string]any {
	newParams := make(map[string]any, len(base)+1)

	// copy base caller params
	for k, v := range base {
		newParams[k] = v
	}

	// add cursor only if present
	if cursor != nil && *cursor != "" {
		newParams["cursor"] = *cursor
	}

	return newParams
}
