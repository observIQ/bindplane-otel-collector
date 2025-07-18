// Code generated by mdatagen. DO NOT EDIT.

package metadatatest

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
)

func NewSettings(tt *componenttest.Telemetry) receiver.Settings {
	set := receivertest.NewNopSettings(receivertest.NopType)
	set.ID = component.NewID(component.MustNewType("s3event"))
	set.TelemetrySettings = tt.NewTelemetrySettings()
	return set
}

func AssertEqualS3eventBatchSize(t *testing.T, tt *componenttest.Telemetry, dps []metricdata.HistogramDataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_s3event.batch_size",
		Description: "The number of logs in a batch.",
		Unit:        "{logs}",
		Data: metricdata.Histogram[int64]{
			Temporality: metricdata.CumulativeTemporality,
			DataPoints:  dps,
		},
	}
	got, err := tt.GetMetric("otelcol_s3event.batch_size")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualS3eventFailures(t *testing.T, tt *componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_s3event.failures",
		Description: "The number of failures encountered while processing S3 objects",
		Unit:        "{failures}",
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  dps,
		},
	}
	got, err := tt.GetMetric("otelcol_s3event.failures")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualS3eventObjectsHandled(t *testing.T, tt *componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_s3event.objects_handled",
		Description: "The number of S3 objects processed by the receiver",
		Unit:        "{objects}",
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  dps,
		},
	}
	got, err := tt.GetMetric("otelcol_s3event.objects_handled")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t, want, got, opts...)
}
