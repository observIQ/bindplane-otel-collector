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

	"github.com/observiq/bindplane-otel-collector/internal/report"
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

	// Give the goroutine a chance to process; nothing should be sent.
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, handler.sentMessages(), "request for a different processor must not be answered")
}
