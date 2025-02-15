package chronicleexporter // import "github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter"

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	mp = otel.Meter("chronicleexporter")

	batchLogCount              metric.Int64Histogram
	payloadSizeBytes           metric.Int64Histogram
	requestLatencyMilliseconds metric.Int64Histogram
)

func setupMetrics() error {
	var err error

	batchLogCount, err = mp.Int64Histogram(
		"otelcol_exporter_batch_log_count",
		metric.WithDescription("The number of logs in a batch"),
		metric.WithExplicitBucketBoundaries(
			1,
			10,
			100,
			200,
			300,
			400,
			500,
			600,
			700,
			800,
			900,
			1000,
		),
	)
	if err != nil {
		return err
	}

	payloadSizeBytes, err = mp.Int64Histogram(
		"otelcol_exporter_payload_size_bytes",
		metric.WithDescription("The size of the payload in bytes"),
		// TODO(jsirianni): What is the max payload size secops accepts?
		metric.WithExplicitBucketBoundaries(
			250, // typical single log
			500,
			1000, // typical large log
			5000,
			10000,
			50000,
			100000, // 0.10mb
			150000,
			200000,
			250000, // typical single log in 1,000 batch
			300000,
			350000,
			400000,
			450000,
			500000, // 0.50mb
			550000,
			600000,
			650000,
			700000,
			750000,
			800000,
			850000,
			900000,
			950000,
			1000000, // 1mb
			2000000,
			3000000,
			4000000,
			5000000, // 5mb
		),
	)
	if err != nil {
		return err
	}

	requestLatencyMilliseconds, err = mp.Int64Histogram(
		"otelcol_exporter_request_latency_milliseconds",
		metric.WithDescription("The latency of the request in milliseconds"),
		metric.WithExplicitBucketBoundaries(
			100,
			150,
			200,
			250,
			300,
			350,
			400,
			450,
			500,
			550,
			600,
			650,
			700,
			750,
			800,
			850,
			900,
			950,
			1000,  // 1 second
			10000, // 10 seconds
			15000, // 15 seconds
			20000, // 20 seconds
			30000, // 30 seconds
			60000, // 1 minute
		),
	)

	return nil
}
