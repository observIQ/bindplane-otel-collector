// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter"

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// marshaler marshals into data bytes based on configuration
//
//go:generate mockery --name marshaler --output ./internal/mocks --with-expecter --filename mock_marshaler.go --structname MockMarshaler
type marshaler interface {
	// MarshalTraces returns the marshaled json traces data
	MarshalTraces(td ptrace.Traces) ([]byte, error)

	// MarshalLogs returns the marshaled json logs data
	MarshalLogs(ld plog.Logs) ([]byte, error)

	// MarshalMetrics returns the marshaled json metrics data
	MarshalMetrics(md pmetric.Metrics) ([]byte, error)

	// Format returns the file format of the data this marshaler returns
	Format() string
}

func newMarshaler() marshaler {
	return &baseMarshaller{
		logsMarshaler:    &plog.JSONMarshaler{},
		tracesMarshaler:  &ptrace.JSONMarshaler{},
		metricsMarshaler: &pmetric.JSONMarshaler{},
	}
}

// baseMarshaller is the base marshaller that marshals otlp structures into JSON
type baseMarshaller struct {
	logsMarshaler    plog.Marshaler
	tracesMarshaler  ptrace.Marshaler
	metricsMarshaler pmetric.Marshaler
}

func (b *baseMarshaller) MarshalTraces(td ptrace.Traces) ([]byte, error) {
	return b.tracesMarshaler.MarshalTraces(td)
}

func (b *baseMarshaller) MarshalLogs(ld plog.Logs) ([]byte, error) {
	return b.logsMarshaler.MarshalLogs(ld)
}

func (b *baseMarshaller) MarshalMetrics(md pmetric.Metrics) ([]byte, error) {
	return b.metricsMarshaler.MarshalMetrics(md)
}

func (b *baseMarshaller) Format() string {
	return "json"
}