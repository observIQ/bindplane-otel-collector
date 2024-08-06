// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestProcess_Logs(t *testing.T) {
	factory := NewFactory()
	sink := &consumertest.LogsSink{}

	pSet := processortest.NewNopSettings()
	p, err := factory.CreateLogsProcessor(context.Background(), pSet, factory.CreateDefaultConfig(), sink)
	require.NoError(t, err)

	mockOpamp := &mockOpAMPExtension{
		msgChan: make(chan *protobufs.CustomMessage, 1),
	}

	mockHost := &mockHost{
		extensions: map[component.ID]component.Component{
			component.MustNewID("opamp"): mockOpamp,
		},
	}

	require.NoError(t, p.Start(context.Background(), mockHost))
	t.Cleanup(func() {
		require.NoError(t, p.Shutdown(context.Background()))
	})

	require.Equal(t, "com.bindplane.snapshot", mockOpamp.capability)

	l, err := golden.ReadLogs(filepath.Join("testdata", "logs", "w3c-logs.yaml"))
	require.NoError(t, err)

	require.NoError(t, p.ConsumeLogs(context.Background(), l))

	require.Equal(t, 1, len(sink.AllLogs()))
	require.Equal(t, l, sink.AllLogs()[0])

	// Request buffer
	reqPayload := fmt.Sprintf(`{"processor":%q,"pipeline_type":"logs","session_id":"my-session-id"}`, pSet.ID)

	cm := &protobufs.CustomMessage{
		Capability: "com.bindplane.snapshot",
		Type:       "requestSnapshot",
		Data:       []byte(reqPayload),
	}

	mockOpamp.msgChan <- cm

	// Wait for response
	require.Eventually(t, func() bool {
		return mockOpamp.GotMessage()
	}, 5*time.Second, 100*time.Millisecond)

	by, err := os.ReadFile(filepath.Join("testdata", "snapshot", "logs-report.json"))
	require.NoError(t, err)

	var expectedMessageContents map[string]any
	err = json.Unmarshal(by, &expectedMessageContents)
	require.NoError(t, err)

	var actualMessageContents map[string]any
	err = json.Unmarshal(gunzipBytes(t, mockOpamp.sentMessage), &actualMessageContents)
	require.NoError(t, err)

	require.Equal(t, expectedMessageContents, actualMessageContents)
	require.Equal(t, "reportSnapshot", mockOpamp.sentMessageType)
}

func TestProcess_Metrics(t *testing.T) {
	factory := NewFactory()
	sink := &consumertest.MetricsSink{}

	pSet := processortest.NewNopSettings()
	p, err := factory.CreateMetricsProcessor(context.Background(), pSet, factory.CreateDefaultConfig(), sink)
	require.NoError(t, err)

	mockOpamp := &mockOpAMPExtension{
		msgChan: make(chan *protobufs.CustomMessage, 1),
	}

	mockHost := &mockHost{
		extensions: map[component.ID]component.Component{
			component.MustNewID("opamp"): mockOpamp,
		},
	}

	require.NoError(t, p.Start(context.Background(), mockHost))
	t.Cleanup(func() {
		require.NoError(t, p.Shutdown(context.Background()))
	})

	require.Equal(t, "com.bindplane.snapshot", mockOpamp.capability)

	m, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
	require.NoError(t, err)

	require.NoError(t, p.ConsumeMetrics(context.Background(), m))

	require.Equal(t, 1, len(sink.AllMetrics()))
	require.Equal(t, m, sink.AllMetrics()[0])

	// Request buffer
	reqPayload := fmt.Sprintf(`{"processor":%q,"pipeline_type":"metrics","session_id":"my-session-id"}`, pSet.ID)

	cm := &protobufs.CustomMessage{
		Capability: "com.bindplane.snapshot",
		Type:       "requestSnapshot",
		Data:       []byte(reqPayload),
	}

	mockOpamp.msgChan <- cm

	// Wait for response
	require.Eventually(t, func() bool {
		return mockOpamp.GotMessage()
	}, 5*time.Second, 100*time.Millisecond)

	by, err := os.ReadFile(filepath.Join("testdata", "snapshot", "metrics-report.json"))
	require.NoError(t, err)

	var expectedMessageContents map[string]any
	err = json.Unmarshal(by, &expectedMessageContents)
	require.NoError(t, err)

	var actualMessageContents map[string]any
	err = json.Unmarshal(gunzipBytes(t, mockOpamp.sentMessage), &actualMessageContents)
	require.NoError(t, err)

	require.Equal(t, expectedMessageContents, actualMessageContents)
	require.Equal(t, "reportSnapshot", mockOpamp.sentMessageType)
}

func TestProcess_Traces(t *testing.T) {
	factory := NewFactory()
	sink := &consumertest.TracesSink{}

	pSet := processortest.NewNopSettings()
	p, err := factory.CreateTracesProcessor(context.Background(), pSet, factory.CreateDefaultConfig(), sink)
	require.NoError(t, err)

	mockOpamp := &mockOpAMPExtension{
		msgChan: make(chan *protobufs.CustomMessage, 1),
	}

	mockHost := &mockHost{
		extensions: map[component.ID]component.Component{
			component.MustNewID("opamp"): mockOpamp,
		},
	}

	require.NoError(t, p.Start(context.Background(), mockHost))
	t.Cleanup(func() {
		require.NoError(t, p.Shutdown(context.Background()))
	})

	require.Equal(t, "com.bindplane.snapshot", mockOpamp.capability)

	tr, err := golden.ReadTraces(filepath.Join("testdata", "traces", "bindplane-traces.yaml"))
	require.NoError(t, err)

	require.NoError(t, p.ConsumeTraces(context.Background(), tr))

	require.Equal(t, 1, len(sink.AllTraces()))
	require.Equal(t, tr, sink.AllTraces()[0])

	// Request buffer
	reqPayload := fmt.Sprintf(`{"processor":%q,"pipeline_type":"traces","session_id":"my-session-id"}`, pSet.ID)

	cm := &protobufs.CustomMessage{
		Capability: "com.bindplane.snapshot",
		Type:       "requestSnapshot",
		Data:       []byte(reqPayload),
	}

	mockOpamp.msgChan <- cm

	// Wait for response
	require.Eventually(t, func() bool {
		return mockOpamp.GotMessage()
	}, 5*time.Second, 100*time.Millisecond)

	by, err := os.ReadFile(filepath.Join("testdata", "snapshot", "traces-report.json"))
	require.NoError(t, err)

	var expectedMessageContents map[string]any
	err = json.Unmarshal(by, &expectedMessageContents)
	require.NoError(t, err)

	var actualMessageContents map[string]any
	err = json.Unmarshal(gunzipBytes(t, mockOpamp.sentMessage), &actualMessageContents)
	require.NoError(t, err)

	require.Equal(t, expectedMessageContents, actualMessageContents)
	require.Equal(t, "reportSnapshot", mockOpamp.sentMessageType)
}

// mockHost for component.Host
type mockHost struct {
	extensions map[component.ID]component.Component
}

func (nh *mockHost) GetFactory(component.Kind, component.Type) component.Factory {
	return nil
}

func (nh *mockHost) GetExtensions() map[component.ID]component.Component {
	return nh.extensions
}

type mockOpAMPExtension struct {
	msgChan chan *protobufs.CustomMessage

	capability string

	gotMessageMux   sync.Mutex
	gotMessage      bool
	sentMessageType string
	sentMessage     []byte
}

// Start implements component.Component::Start
func (m *mockOpAMPExtension) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown implements component.Component::Shutdown
func (m *mockOpAMPExtension) Shutdown(_ context.Context) error { return nil }

func (m *mockOpAMPExtension) Register(capability string, _ ...opampcustommessages.CustomCapabilityRegisterOption) (handler opampcustommessages.CustomCapabilityHandler, err error) {
	m.capability = capability
	return m, nil
}

func (m *mockOpAMPExtension) Message() <-chan *protobufs.CustomMessage {
	return m.msgChan
}

func (m *mockOpAMPExtension) SendMessage(messageType string, message []byte) (messageSendingChannel chan struct{}, err error) {
	m.gotMessageMux.Lock()
	defer m.gotMessageMux.Unlock()

	if m.gotMessage {
		return
	}
	m.gotMessage = true

	m.sentMessageType = messageType
	m.sentMessage = message
	return
}

func (m *mockOpAMPExtension) GotMessage() bool {
	m.gotMessageMux.Lock()
	defer m.gotMessageMux.Unlock()

	return m.gotMessage
}

func (m *mockOpAMPExtension) Unregister() {}

func gunzipBytes(t *testing.T, b []byte) []byte {
	t.Helper()

	r, err := gzip.NewReader(bytes.NewBuffer(b))
	require.NoError(t, err)
	bOut, err := io.ReadAll(r)
	require.NoError(t, err)

	return bOut
}
