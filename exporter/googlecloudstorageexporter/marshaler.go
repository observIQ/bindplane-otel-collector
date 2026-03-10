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
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// marshaler marshals into data bytes based on configuration
//
//go:generate mockery --name marshaler --output ./internal/mocks --with-expecter --filename mock_marshaler.go --structname MockMarshaler
type marshaler interface {
	// MarshalTraces returns a reader for the marshaled json traces data
	MarshalTraces(td ptrace.Traces) (io.Reader, error)

	// MarshalLogs returns a reader for the marshaled json logs data
	MarshalLogs(ld plog.Logs) (io.Reader, error)

	// MarshalMetrics returns a reader for the marshaled json metrics data
	MarshalMetrics(md pmetric.Metrics) (io.Reader, error)

	// Format returns the file format of the data this marshaler returns
	Format() string
}

// newMarshaler creates a new marshaler based on compression type
func newMarshaler(compression compressionType) marshaler {
	base := &baseMarshaler{
		logsMarshaler:    &plog.JSONMarshaler{},
		tracesMarshaler:  &ptrace.JSONMarshaler{},
		metricsMarshaler: &pmetric.JSONMarshaler{},
	}

	switch compression {
	case gzipCompression:
		return &gzipMarshaler{
			base: base,
		}
	default:
		return base
	}
}

// baseMarshaler is the base marshaller that marshals otlp structures into JSON
type baseMarshaler struct {
	logsMarshaler    plog.Marshaler
	tracesMarshaler  ptrace.Marshaler
	metricsMarshaler pmetric.Marshaler
}

// MarshalTraces returns a reader for the marshaled json traces data
func (b *baseMarshaler) MarshalTraces(td ptrace.Traces) (io.Reader, error) {
	data, err := b.tracesMarshaler.MarshalTraces(td)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// MarshalLogs returns a reader for the marshaled json logs data
func (b *baseMarshaler) MarshalLogs(ld plog.Logs) (io.Reader, error) {
	data, err := b.logsMarshaler.MarshalLogs(ld)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// MarshalMetrics returns a reader for the marshaled json metrics data
func (b *baseMarshaler) MarshalMetrics(md pmetric.Metrics) (io.Reader, error) {
	data, err := b.metricsMarshaler.MarshalMetrics(md)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// Format returns the file format of the data this marshaler returns
func (b *baseMarshaler) Format() string {
	return "json"
}

// gzipMarshaler gzip compresses marshalled data
type gzipMarshaler struct {
	base *baseMarshaler
}

// MarshalTraces returns a reader for the gzip-compressed marshaled json traces data
func (g *gzipMarshaler) MarshalTraces(td ptrace.Traces) (io.Reader, error) {
	data, err := g.base.tracesMarshaler.MarshalTraces(td)
	if err != nil {
		return nil, fmt.Errorf("marshal traces: %w", err)
	}
	return g.compressReader(data), nil
}

// MarshalLogs returns a reader for the gzip-compressed marshaled json logs data
func (g *gzipMarshaler) MarshalLogs(ld plog.Logs) (io.Reader, error) {
	data, err := g.base.logsMarshaler.MarshalLogs(ld)
	if err != nil {
		return nil, fmt.Errorf("marshal logs: %w", err)
	}
	return g.compressReader(data), nil
}

// MarshalMetrics returns a reader for the gzip-compressed marshaled json metrics data
func (g *gzipMarshaler) MarshalMetrics(md pmetric.Metrics) (io.Reader, error) {
	data, err := g.base.metricsMarshaler.MarshalMetrics(md)
	if err != nil {
		return nil, fmt.Errorf("marshal metrics: %w", err)
	}
	return g.compressReader(data), nil
}

// Format returns the file format of the data this marshaler returns
func (g *gzipMarshaler) Format() string {
	return "json.gz"
}

// compressReader returns a reader that streams gzip-compressed data.
// Compression runs in a goroutine, allowing the consumer to read
// compressed bytes as they are produced.
func (g *gzipMarshaler) compressReader(data []byte) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		gw := gzip.NewWriter(pw)
		if _, err := gw.Write(data); err != nil {
			pw.CloseWithError(err)
			return
		}
		if err := gw.Close(); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.Close()
	}()
	return pr
}
