// Copyright  observIQ, Inc.
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

// Package report contains reporters for collecting specific information about the collector-
package report

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/report/snapshot"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// snapShotKind is the kind for the snapshot reporter
var snapShotKind = "snapshot"

// snapshotConfig specifies what snapshots to collect
type snapshotConfig struct {
	// Endpoint is where to send the snapshots
	Endpoint *endpointConfig `yaml:"endpoint"`

	// Processor is the full ComponentID of the snapshot processor
	Processor string `yaml:"processor"`

	// PipelineType will be "logs", "metrics", or "traces"
	PipelineType string `yaml:"pipeline_type"`

	// SearchQuery is an optional query string that will filter telemetry
	// such that only telemetry containing the string is reported.
	SearchQuery *string `yaml:"search_query"`

	// MinimumTimestamp is the minimum timestamp used to filter telemetry such that only telemetry
	// with a timestamp higher than specified will be reported.
	MinimumTimestamp *time.Time `yaml:"minimum_timestamp"`

	// MaximumPayloadSizeBytes is the maximum size of the payload in bytes that should be sent to the endpoint.
	MaximumPayloadSizeBytes int `json:"maximum_payload_size" yaml:"maximum_payload_size,omitempty" mapstructure:"maximum_payload_size"`
}

// endpointConfig is the configuration of a specific endpoint and full headers to include
type endpointConfig struct {
	URL     string      `yaml:"url"`
	Headers http.Header `yaml:"headers"`
}

var _ Reporter = (*SnapshotReporter)(nil)
var _ snapshot.Snapshotter = (*SnapshotReporter)(nil)

// SnapshotReporter tracks and reports snapshots
type SnapshotReporter struct {
	client Client

	// idealPayloadSize is the desired number of items to be in a payload when a snapshot is reported
	idealPayloadSize int

	// Buffers
	logBuffers    map[string]*snapshot.LogBuffer
	metricBuffers map[string]*snapshot.MetricBuffer
	traceBuffers  map[string]*snapshot.TraceBuffer

	// Buffer Locks
	logLock    sync.Mutex
	metricLock sync.Mutex
	traceLock  sync.Mutex
}

// NewSnapshotReporter creates a new SnapshotReporter with the associated client
func NewSnapshotReporter(client Client) *SnapshotReporter {
	return &SnapshotReporter{
		client:           client,
		idealPayloadSize: 100,
		logBuffers:       make(map[string]*snapshot.LogBuffer),
		metricBuffers:    make(map[string]*snapshot.MetricBuffer),
		traceBuffers:     make(map[string]*snapshot.TraceBuffer),
	}
}

// Kind returns kind of the reporter
func (s *SnapshotReporter) Kind() string {
	return snapShotKind
}

// Report applies the new configuration and reports snapshots specified in it
func (s *SnapshotReporter) Report(cfg any) error {
	ssCfg, ok := cfg.(*snapshotConfig)
	if !ok {
		return errors.New("invalid config type")
	}

	// Gather payload
	payload, err := s.prepRequestPayload(ssCfg.Processor, ssCfg.PipelineType, ssCfg.SearchQuery, ssCfg.MinimumTimestamp, ssCfg.MaximumPayloadSizeBytes)
	if err != nil {
		return fmt.Errorf("prep request payload: %w", err)
	}

	compressedPayload, err := compress(payload)
	if err != nil {
		return fmt.Errorf("compress payload: %w", err)
	}

	// Prep request
	req, err := http.NewRequest(http.MethodPost, ssCfg.Endpoint.URL, bytes.NewReader(compressedPayload))
	if err != nil {
		return fmt.Errorf("failed to construct snapshot request: %w", err)
	}
	// Add content headers
	req.Header.Add("Content-Type", "application/protobuf")
	req.Header.Add("Content-Encoding", "gzip")

	// Add Component-ID header
	req.Header.Add("Component-ID", ssCfg.Processor)

	// Add headers from config
	for k, values := range ssCfg.Endpoint.Headers {
		for _, value := range values {
			req.Header.Add(k, value)
		}
	}

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("snapshot request failed: %w", err)
	}

	if resp.StatusCode > 299 {
		return fmt.Errorf("non-200 response for snapshot report: %d", resp.StatusCode)
	}

	return nil
}

// Reset clears all buffers
func (s *SnapshotReporter) Reset() {
	s.logLock.Lock()
	s.metricLock.Lock()
	s.traceLock.Lock()
	defer s.logLock.Unlock()
	defer s.metricLock.Unlock()
	defer s.traceLock.Unlock()

	s.logBuffers = make(map[string]*snapshot.LogBuffer)
	s.metricBuffers = make(map[string]*snapshot.MetricBuffer)
	s.traceBuffers = make(map[string]*snapshot.TraceBuffer)
}

// SaveLogs saves off logs in a snapshot to be reported later
func (s *SnapshotReporter) SaveLogs(componentID string, ld plog.Logs) {
	s.logLock.Lock()
	buffer, ok := s.logBuffers[componentID]
	if !ok {
		buffer = snapshot.NewLogBuffer(s.idealPayloadSize)
		s.logBuffers[componentID] = buffer
	}
	s.logLock.Unlock()

	buffer.Add(ld)
}

// SaveTraces saves off traces in a snapshot to be reported later
func (s *SnapshotReporter) SaveTraces(componentID string, td ptrace.Traces) {
	s.traceLock.Lock()
	buffer, ok := s.traceBuffers[componentID]
	if !ok {
		buffer = snapshot.NewTraceBuffer(s.idealPayloadSize)
		s.traceBuffers[componentID] = buffer
	}
	s.traceLock.Unlock()

	buffer.Add(td)
}

// SaveMetrics saves off metrics in a snapshot to be reported later
func (s *SnapshotReporter) SaveMetrics(componentID string, md pmetric.Metrics) {
	s.metricLock.Lock()
	buffer, ok := s.metricBuffers[componentID]
	if !ok {
		buffer = snapshot.NewMetricBuffer(s.idealPayloadSize)
		s.metricBuffers[componentID] = buffer
	}
	s.metricLock.Unlock()

	buffer.Add(md)
}

// prepRequestPayload based on the pipelineType will return a marshaled proto of the OTLP data types for the componentID
func (s *SnapshotReporter) prepRequestPayload(componentID, pipelineType string, searchQuery *string, minimumTimestamp *time.Time, maximumPayloadSize int) (payload []byte, err error) {
	switch pipelineType {
	case "logs":
		s.logLock.Lock()
		buffer, ok := s.logBuffers[componentID]
		s.logLock.Unlock()
		if !ok {
			return []byte{}, nil
		}

		payload, err = buffer.ConstructPayload(&plog.ProtoMarshaler{}, searchQuery, minimumTimestamp, maximumPayloadSize)
	case "metrics":
		s.metricLock.Lock()
		buffer, ok := s.metricBuffers[componentID]
		s.metricLock.Unlock()
		if !ok {
			return []byte{}, nil
		}

		payload, err = buffer.ConstructPayload(&pmetric.ProtoMarshaler{}, searchQuery, minimumTimestamp, maximumPayloadSize)
	case "traces":
		s.traceLock.Lock()
		buffer, ok := s.traceBuffers[componentID]
		s.traceLock.Unlock()
		if !ok {
			return []byte{}, nil
		}

		payload, err = buffer.ConstructPayload(&ptrace.ProtoMarshaler{}, searchQuery, minimumTimestamp, maximumPayloadSize)
	}

	return
}

// compress gzip compresses the data
func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
