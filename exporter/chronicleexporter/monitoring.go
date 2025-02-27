package chronicleexporter // import "github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter"

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	mp = otel.Meter("chronicleexporter")

	// Common metrics (gRPC and HTTP)
	batchLogCount              metric.Int64Histogram
	requestLatencyMilliseconds metric.Int64Histogram

	// HTTP exporter metrics
	payloadSizeBytes metric.Int64Histogram
)

func setupMetrics() error {
	var err error

	batchLogCount, err = mp.Int64Histogram(
		"otelcol_exporter_batch_size",
		metric.WithDescription("The number of logs in a batch"),
		metric.WithExplicitBucketBoundaries(
			1,
			100,
			250,
			500,
			750,
			1000,
		),
	)
	if err != nil {
		return err
	}

	payloadSizeBytes, err = mp.Int64Histogram(
		"otelcol_exporter_payload_size",
		metric.WithDescription("The size of the payload in bytes"),
		metric.WithExplicitBucketBoundaries(
			100,
			500,
			1000,    // typical large log
			100000,  // 0.10mb
			500000,  // 0.50mb
			1000000, // 1mb
			2000000,
			3000000,
			4000000,
			5000000, // 5mb
		),
		metric.WithUnit("B"),
	)
	if err != nil {
		return err
	}

	requestLatencyMilliseconds, err = mp.Int64Histogram(
		"otelcol_exporter_request_latency",
		metric.WithDescription("The latency of the request in milliseconds"),
		metric.WithExplicitBucketBoundaries(
			100,
			200,
			300,
			400,
			500,
			600,
			700,
			800,
			900,
			1000, // 1 second
			2000,
			5000,
			10000, // 10 seconds
			15000,
			20000,
			30000,
			60000, // 1 minute
		),
		metric.WithUnit("ms"),
	)

	return nil
}
