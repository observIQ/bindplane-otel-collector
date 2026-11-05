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

package snapshotprocessor

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dynatrace/dynatrace-bindplane-otel-collector/pkg/report"
	"github.com/dynatrace/dynatrace-bindplane-otel-contrib/pkg/snapshot"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gopkg.in/yaml.v3"
)

// fakeHandler is a CustomCapabilityHandler that records sent messages
// and lets the test feed inbound messages into the processor.
type fakeHandler struct {
	inbound      chan *protobufs.CustomMessage
	mu           sync.Mutex
	sent         []sentMessage
	unregistered atomic.Bool
}

type sentMessage struct {
	messageType string
	data        []byte
}

func newFakeHandler() *fakeHandler {
	return &fakeHandler{inbound: make(chan *protobufs.CustomMessage, 8)}
}

func (h *fakeHandler) Message() <-chan *protobufs.CustomMessage { return h.inbound }

func (h *fakeHandler) SendMessage(messageType string, message []byte) (chan struct{}, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sent = append(h.sent, sentMessage{messageType: messageType, data: message})
	ch := make(chan struct{})
	close(ch)
	return ch, nil
}

func (h *fakeHandler) Unregister() { h.unregistered.Store(true) }

func (h *fakeHandler) sentMessages() []sentMessage {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]sentMessage, len(h.sent))
	copy(out, h.sent)
	return out
}

// fakeRegistry implements opampcustommessages.CustomCapabilityRegistry +
// extension.Extension so it can be plugged into a test host.
type fakeRegistry struct {
	component.StartFunc
	component.ShutdownFunc

	registeredCapability string
	handler              *fakeHandler
}

func (r *fakeRegistry) Register(capability string, _ ...opampcustommessages.CustomCapabilityRegisterOption) (opampcustommessages.CustomCapabilityHandler, error) {
	r.registeredCapability = capability
	return r.handler, nil
}

// hostWithExtensions is a component.Host that exposes a fixed set of
// extensions by ID.
type hostWithExtensions struct {
	component.Host
	exts map[component.ID]component.Component
}

func (h hostWithExtensions) GetExtensions() map[component.ID]component.Component { return h.exts }

func newHost(ext component.Component, id component.ID) hostWithExtensions {
	return hostWithExtensions{
		Host: componenttest.NewNopHost(),
		exts: map[component.ID]component.Component{id: ext},
	}
}

// Sanity: fakeRegistry satisfies the extension.Extension interface and
// the opampcustommessages.CustomCapabilityRegistry interface.
var (
	_ extension.Extension                          = (*fakeRegistry)(nil)
	_ opampcustommessages.CustomCapabilityRegistry = (*fakeRegistry)(nil)
)

func TestStart_NoOpAMP_DoesNotRegister(t *testing.T) {
	reporter := report.NewSnapshotReporter(nil)
	defer overwriteSnapshotSet(t, reporter)()

	cfg := &Config{Enabled: true} // OpAMP unset
	sp := newSnapshotProcessor(zap.NewNop(), cfg, component.MustNewIDWithName("snapshotprocessor", "x"))

	// Empty host (no extensions); start must succeed and skip
	// capability registration entirely.
	require.NoError(t, sp.start(context.Background(), componenttest.NewNopHost()))
	require.Nil(t, sp.handler, "handler should remain nil in v1-only mode")

	// Stop is also safe.
	require.NoError(t, sp.stop(context.Background()))
}

func TestStart_OpAMPMissingExtension(t *testing.T) {
	reporter := report.NewSnapshotReporter(nil)
	defer overwriteSnapshotSet(t, reporter)()

	cfg := &Config{Enabled: true, OpAMP: component.MustNewID("opamp_connection")}
	sp := newSnapshotProcessor(zap.NewNop(), cfg, component.MustNewIDWithName("snapshotprocessor", "x"))

	err := sp.start(context.Background(), componenttest.NewNopHost())
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not exist")
}

func TestStart_OpAMPRegistersAndShutdownUnregisters(t *testing.T) {
	reporter := report.NewSnapshotReporter(nil)
	defer overwriteSnapshotSet(t, reporter)()

	handler := newFakeHandler()
	registry := &fakeRegistry{handler: handler}
	extID := component.MustNewID("opamp_connection")
	host := newHost(registry, extID)

	cfg := &Config{Enabled: true, OpAMP: extID}
	sp := newSnapshotProcessor(zap.NewNop(), cfg, component.MustNewIDWithName("snapshotprocessor", "x"))

	require.NoError(t, sp.start(context.Background(), host))
	require.Equal(t, snapshotCapability, registry.registeredCapability)
	require.Same(t, handler, sp.handler)

	require.NoError(t, sp.stop(context.Background()))
	require.True(t, handler.unregistered.Load(), "handler should be unregistered on stop")
}

func TestHandleSnapshotRequest_LogsRoundTrip(t *testing.T) {
	reporter := report.NewSnapshotReporter(nil)
	defer overwriteSnapshotSet(t, reporter)()

	handler := newFakeHandler()
	registry := &fakeRegistry{handler: handler}
	extID := component.MustNewID("opamp_connection")
	host := newHost(registry, extID)

	processorID := component.MustNewIDWithName("snapshotprocessor", "x")
	cfg := &Config{Enabled: true, OpAMP: extID}
	sp := newSnapshotProcessor(zap.NewNop(), cfg, processorID)
	require.NoError(t, sp.start(context.Background(), host))
	defer func() { require.NoError(t, sp.stop(context.Background())) }()

	// Push a logs payload via the v1 path (simulating real telemetry flow).
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty().Body().SetStr("hello")
	_, err := sp.processLogs(context.Background(), logs)
	require.NoError(t, err)

	// Send a v2 snapshot request via the inbound channel.
	req := snapshotRequest{
		Processor:    processorID,
		PipelineType: "logs",
		SessionID:    "s1",
	}
	reqBody, err := yaml.Marshal(req)
	require.NoError(t, err)
	handler.inbound <- &protobufs.CustomMessage{Type: snapshotRequestType, Data: reqBody}

	// Wait for the response to be sent.
	require.Eventually(t, func() bool {
		return len(handler.sentMessages()) >= 1
	}, time.Second, 10*time.Millisecond)

	sent := handler.sentMessages()[0]
	require.Equal(t, snapshotReportType, sent.messageType)

	// Decompress + JSON-decode the response.
	gz, err := gzip.NewReader(bytes.NewReader(sent.data))
	require.NoError(t, err)
	rawJSON, err := io.ReadAll(gz)
	require.NoError(t, err)

	var got snapshotReport
	require.NoError(t, json.Unmarshal(rawJSON, &got))
	assert.Equal(t, "s1", got.SessionID)
	assert.Equal(t, "logs", got.TelemetryType)
	require.NotEmpty(t, got.TelemetryPayload, "expected non-empty telemetry payload")
}

func TestHandleSnapshotRequest_PayloadIsValidJSON(t *testing.T) {
	cases := []struct {
		pipelineType string
		save         func(t *testing.T, sp *snapshotProcessor)
		unmarshal    func(payload []byte) error
	}{
		{
			pipelineType: "logs",
			save: func(t *testing.T, sp *snapshotProcessor) {
				logs := plog.NewLogs()
				logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("hello")
				_, err := sp.processLogs(context.Background(), logs)
				require.NoError(t, err)
			},
			unmarshal: func(payload []byte) error {
				_, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(payload)
				return err
			},
		},
		{
			pipelineType: "metrics",
			save: func(t *testing.T, sp *snapshotProcessor) {
				metrics := pmetric.NewMetrics()
				m := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				m.SetName("m")
				m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(1)
				_, err := sp.processMetrics(context.Background(), metrics)
				require.NoError(t, err)
			},
			unmarshal: func(payload []byte) error {
				_, err := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics(payload)
				return err
			},
		},
		{
			pipelineType: "traces",
			save: func(t *testing.T, sp *snapshotProcessor) {
				traces := ptrace.NewTraces()
				traces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("s")
				_, err := sp.processTraces(context.Background(), traces)
				require.NoError(t, err)
			},
			unmarshal: func(payload []byte) error {
				_, err := (&ptrace.JSONUnmarshaler{}).UnmarshalTraces(payload)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.pipelineType, func(t *testing.T) {
			reporter := report.NewSnapshotReporter(nil)
			defer overwriteSnapshotSet(t, reporter)()

			handler := newFakeHandler()
			registry := &fakeRegistry{handler: handler}
			extID := component.MustNewID("opamp_connection")
			host := newHost(registry, extID)

			processorID := component.MustNewIDWithName("snapshotprocessor", "x")
			cfg := &Config{Enabled: true, OpAMP: extID}
			sp := newSnapshotProcessor(zap.NewNop(), cfg, processorID)
			require.NoError(t, sp.start(context.Background(), host))
			defer func() { require.NoError(t, sp.stop(context.Background())) }()

			request := func(sessionID string, wantCount int) snapshotReport {
				req := snapshotRequest{
					Processor:    processorID,
					PipelineType: tc.pipelineType,
					SessionID:    sessionID,
				}
				body, err := yaml.Marshal(req)
				require.NoError(t, err)
				handler.inbound <- &protobufs.CustomMessage{Type: snapshotRequestType, Data: body}

				require.Eventually(t, func() bool {
					return len(handler.sentMessages()) >= wantCount
				}, time.Second, 10*time.Millisecond)

				gz, err := gzip.NewReader(bytes.NewReader(handler.sentMessages()[wantCount-1].data))
				require.NoError(t, err)
				rawJSON, err := io.ReadAll(gz)
				require.NoError(t, err)

				var got snapshotReport
				require.NoError(t, json.Unmarshal(rawJSON, &got))
				return got
			}

			// Empty state: no buffer exists for this processor yet.
			empty := request("empty", 1)
			require.True(t, json.Valid(empty.TelemetryPayload))
			require.NoError(t, tc.unmarshal(empty.TelemetryPayload))

			tc.save(t, sp)

			full := request("full", 2)
			require.True(t, json.Valid(full.TelemetryPayload))
			require.NoError(t, tc.unmarshal(full.TelemetryPayload))
			require.NotEqual(t, "{}", string(full.TelemetryPayload))
		})
	}
}

func TestHandleSnapshotRequest_WrongProcessorIDIsIgnored(t *testing.T) {
	reporter := report.NewSnapshotReporter(nil)
	defer overwriteSnapshotSet(t, reporter)()

	handler := newFakeHandler()
	registry := &fakeRegistry{handler: handler}
	extID := component.MustNewID("opamp_connection")
	host := newHost(registry, extID)

	cfg := &Config{Enabled: true, OpAMP: extID}
	sp := newSnapshotProcessor(zap.NewNop(), cfg, component.MustNewIDWithName("snapshotprocessor", "x"))
	require.NoError(t, sp.start(context.Background(), host))
	defer func() { require.NoError(t, sp.stop(context.Background())) }()

	// Address the request to a different processor instance.
	req := snapshotRequest{
		Processor:    component.MustNewIDWithName("snapshotprocessor", "other"),
		PipelineType: "logs",
		SessionID:    "skipme",
	}
	body, err := yaml.Marshal(req)
	require.NoError(t, err)
	handler.inbound <- &protobufs.CustomMessage{Type: snapshotRequestType, Data: body}

	// A request for a different processor must never be answered; confirm nothing is
	// sent across a short window instead of sleeping once.
	require.Never(t, func() bool {
		return len(handler.sentMessages()) > 0
	}, 50*time.Millisecond, 10*time.Millisecond, "request for a different processor must not be answered")
}

// newStartedProcessor wires a processor to a fake OpAMP handler so a test can
// push requests in and read the gzipped reports back out.
func newStartedProcessor(t *testing.T) (*snapshotProcessor, *fakeHandler, component.ID) {
	t.Helper()
	reporter := report.NewSnapshotReporter(nil)
	t.Cleanup(overwriteSnapshotSet(t, reporter))

	handler := newFakeHandler()
	extID := component.MustNewID("opamp_connection")
	host := newHost(&fakeRegistry{handler: handler}, extID)

	processorID := component.MustNewIDWithName("snapshotprocessor", "x")
	sp := newSnapshotProcessor(zap.NewNop(), &Config{Enabled: true, OpAMP: extID}, processorID)
	require.NoError(t, sp.start(context.Background(), host))
	t.Cleanup(func() { require.NoError(t, sp.stop(context.Background())) })
	return sp, handler, processorID
}

// requestRawReport sends a snapshot request and returns the decompressed JSON
// of the wantCount-th report the handler sent.
func requestRawReport(t *testing.T, handler *fakeHandler, processorID component.ID, pipelineType, sessionID string, wantCount int) []byte {
	t.Helper()
	body, err := yaml.Marshal(snapshotRequest{Processor: processorID, PipelineType: pipelineType, SessionID: sessionID})
	require.NoError(t, err)
	handler.inbound <- &protobufs.CustomMessage{Type: snapshotRequestType, Data: body}

	// Generous: an on-demand request legitimately waits a second for a fill.
	require.Eventually(t, func() bool {
		return len(handler.sentMessages()) >= wantCount
	}, 5*time.Second, 10*time.Millisecond)

	gz, err := gzip.NewReader(bytes.NewReader(handler.sentMessages()[wantCount-1].data))
	require.NoError(t, err)
	rawJSON, err := io.ReadAll(gz)
	require.NoError(t, err)
	return rawJSON
}

func TestHandleSnapshotRequest_ReportCarriesMetadata(t *testing.T) {
	cases := []struct {
		pipelineType string
		save         func(t *testing.T, sp *snapshotProcessor)
		count        func(t *testing.T, payload []byte) int
	}{
		{
			pipelineType: "logs",
			save: func(t *testing.T, sp *snapshotProcessor) {
				logs := plog.NewLogs()
				lrs := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
				for i := 0; i < 3; i++ {
					lrs.AppendEmpty().Body().SetStr("hello")
				}
				_, err := sp.processLogs(context.Background(), logs)
				require.NoError(t, err)
			},
			count: func(t *testing.T, payload []byte) int {
				ld, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(payload)
				require.NoError(t, err)
				return ld.LogRecordCount()
			},
		},
		{
			pipelineType: "metrics",
			save: func(t *testing.T, sp *snapshotProcessor) {
				metrics := pmetric.NewMetrics()
				m := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				m.SetName("m")
				dps := m.SetEmptyGauge().DataPoints()
				for i := 0; i < 3; i++ {
					dps.AppendEmpty().SetIntValue(int64(i))
				}
				_, err := sp.processMetrics(context.Background(), metrics)
				require.NoError(t, err)
			},
			count: func(t *testing.T, payload []byte) int {
				md, err := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics(payload)
				require.NoError(t, err)
				return md.DataPointCount()
			},
		},
		{
			pipelineType: "traces",
			save: func(t *testing.T, sp *snapshotProcessor) {
				traces := ptrace.NewTraces()
				spans := traces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans()
				for i := 0; i < 3; i++ {
					spans.AppendEmpty().SetName("s")
				}
				_, err := sp.processTraces(context.Background(), traces)
				require.NoError(t, err)
			},
			count: func(t *testing.T, payload []byte) int {
				td, err := (&ptrace.JSONUnmarshaler{}).UnmarshalTraces(payload)
				require.NoError(t, err)
				return td.SpanCount()
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.pipelineType, func(t *testing.T) {
			sp, handler, processorID := newStartedProcessor(t)
			tc.save(t, sp)

			var got snapshotReport
			require.NoError(t, json.Unmarshal(requestRawReport(t, handler, processorID, tc.pipelineType, "s1", 1), &got))
			require.NotNil(t, got.Metadata)
			assert.Equal(t, "continuous", got.Metadata.Mode)
			assert.True(t, got.Metadata.Filled)
			assert.Zero(t, got.Metadata.FillWaitMs)
			assert.Equal(t, 3, got.Metadata.Records)
			assert.Equal(t, tc.count(t, got.TelemetryPayload), got.Metadata.Records)
		})
	}
}

func TestHandleSnapshotRequest_NoBufferReportHasNoMetadata(t *testing.T) {
	_, handler, processorID := newStartedProcessor(t)

	// Nothing was ever saved, so there is no buffer and nothing to describe.
	var got map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(requestRawReport(t, handler, processorID, "logs", "s1", 1), &got))
	assert.NotContains(t, got, "metadata")
	assert.JSONEq(t, "{}", string(got["telemetry_payload"]))
}

func TestHandleSnapshotRequest_ReportDecodesWithoutMetadataField(t *testing.T) {
	sp, handler, processorID := newStartedProcessor(t)
	logs := plog.NewLogs()
	logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("hello")
	_, err := sp.processLogs(context.Background(), logs)
	require.NoError(t, err)

	// A server built before metadata existed decodes into this shape and
	// must keep working.
	var legacy struct {
		SessionID        string          `json:"session_id"`
		TelemetryType    string          `json:"telemetry_type"`
		TelemetryPayload json.RawMessage `json:"telemetry_payload"`
	}
	raw := requestRawReport(t, handler, processorID, "logs", "s1", 1)
	require.Contains(t, string(raw), `"metadata"`)
	require.NoError(t, json.Unmarshal(raw, &legacy))
	assert.Equal(t, "s1", legacy.SessionID)
	assert.Equal(t, "logs", legacy.TelemetryType)
	require.NotEmpty(t, legacy.TelemetryPayload)
}

// TestHandleSnapshotRequest_PayloadCannotFit covers the failure path: a
// payload that cannot be made to fit even at the lowest retention is logged
// with the request's stats, and no report is sent.
func TestHandleSnapshotRequest_PayloadCannotFit(t *testing.T) {
	reporter := report.NewSnapshotReporter(nil)
	t.Cleanup(overwriteSnapshotSet(t, reporter))
	handler := newFakeHandler()
	extID := component.MustNewID("opamp_connection")
	host := newHost(&fakeRegistry{handler: handler}, extID)
	processorID := component.MustNewIDWithName("snapshotprocessor", "x")
	core, logs := observer.New(zap.ErrorLevel)
	sp := newSnapshotProcessor(zap.New(core), &Config{Enabled: true, OpAMP: extID}, processorID)
	require.NoError(t, sp.start(context.Background(), host))
	t.Cleanup(func() { require.NoError(t, sp.stop(context.Background())) })

	ld := plog.NewLogs()
	lrs := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
	for i := 0; i < 3; i++ {
		lrs.AppendEmpty().Body().SetStr("hello")
	}
	_, err := sp.processLogs(context.Background(), ld)
	require.NoError(t, err)

	// One byte cannot hold even an empty gzip stream, so every retention fails.
	body, err := yaml.Marshal(snapshotRequest{Processor: processorID, PipelineType: "logs", SessionID: "too-small", MaximumPayloadSizeBytes: 1})
	require.NoError(t, err)
	handler.inbound <- &protobufs.CustomMessage{Type: snapshotRequestType, Data: body}

	const msg = "construct logs payload"
	require.Eventually(t, func() bool {
		return logs.FilterMessage(msg).Len() == 1
	}, 5*time.Second, 10*time.Millisecond)
	fields := logs.FilterMessage(msg).All()[0].ContextMap()
	assert.Equal(t, "continuous", fields["mode"])
	assert.Equal(t, true, fields["filled"])
	assert.Contains(t, fields, "fill_wait_ms")
	assert.Contains(t, fields, "duration_ms")
	assert.Contains(t, fields["error"], "too large")
	assert.Empty(t, handler.sentMessages(), "no report is sent for a request that failed")
}

// TestReportMetadata checks the metadata for a request that waited the whole
// fill wait and was served the last known records, and that sub-millisecond
// timings are kept rather than truncated.
func TestReportMetadata(t *testing.T) {
	m := newReportMetadata(snapshot.RequestStats{
		Mode:     snapshot.ModeOnDemand,
		Filled:   false,
		FillWait: 1002 * time.Millisecond,
		Duration: 1002*time.Millisecond + 312*time.Microsecond,
		Records:  100,
	})
	assert.Equal(t, "on_demand", m.Mode)
	assert.False(t, m.Filled)
	assert.InDelta(t, 1002, m.FillWaitMs, 1e-9)
	assert.InDelta(t, 1002.312, m.DurationMs, 1e-9)
	assert.Equal(t, 100, m.Records)

	m = newReportMetadata(snapshot.RequestStats{Mode: snapshot.ModeContinuous, Filled: true, Duration: 312 * time.Microsecond, Records: 7})
	assert.Zero(t, m.FillWaitMs)
	assert.InDelta(t, 0.312, m.DurationMs, 1e-9)
}
