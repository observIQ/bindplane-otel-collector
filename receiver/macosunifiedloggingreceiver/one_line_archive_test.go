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

package macosunifiedloggingreceiver

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"
	"github.com/observiq/bindplane-otel-collector/receiver/macosunifiedloggingreceiver/internal/metadata"
)

func TestParseOneLineArchive(t *testing.T) {
	p := filepath.Join("testdata", "one_line_archive.logarchive")
	glob := filepath.Join(p, "Persist", "*.tracev3")

	extFactory := macosunifiedloggingencodingextension.NewFactory()
	extCfg := extFactory.CreateDefaultConfig()

	ext, err := extFactory.Create(
		context.Background(),
		extensiontest.NewNopSettings(extFactory.Type()),
		extCfg,
	)
	require.NoError(t, err)

	host := &mockHostForTest{
		extensions: map[component.ID]component.Component{
			component.MustNewID("macosunifiedlogencoding"): ext,
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Encoding = "macosunifiedlogencoding"
	cfg.TraceV3Paths = []string{glob}
	cfg.StartAt = "beginning"
	cfg.PollInterval = 100 * time.Millisecond

	sink := new(consumertest.LogsSink)
	rcv, err := factory.CreateLogs(context.Background(), receivertest.NewNopSettings(metadata.Type), cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, rcv)

	// // Test that we can start and stop the receiver
	err = rcv.Start(context.Background(), host)
	require.NoError(t, err, "failed to start receiver")

	// Wait for the file to be processed
	// Run: "log raw-dump -a testdata/one-line.logarchive"
	// total log entries: 1
	require.Eventually(
		t,
		func() bool { return sink.LogRecordCount() > 0 },
		5*time.Second, 10*time.Millisecond,
	)

	// Verify the log content
	logs := sink.AllLogs()
	require.Len(t, logs, 1)

	byEventType := map[string]int{}

	rls := logs[0].ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		sls := rls.At(i).ScopeLogs()
		for j := 0; j < sls.Len(); j++ {
			lrs := sls.At(j).LogRecords()
			for k := 0; k < lrs.Len(); k++ {
				lr := lrs.At(k)

				// Event Types: logEvent, activityEvent, traceEvent, signpostEvent, lossEvent
				if v, ok := lr.Attributes().Get("eventType"); ok {
					switch v.AsString() {
					case "logEvent":
						byEventType["logEvent"]++
					case "activityEvent":
						byEventType["activityEvent"]++
					case "traceEvent":
						byEventType["traceEvent"]++
					case "signpostEvent":
						byEventType["signpostEvent"]++
					case "lossEvent":
						byEventType["lossEvent"]++
					}
				}
			}
		}
	}

	t.Logf("map: %#v", byEventType)

	require.Equal(t, 1, byEventType["signpostEvent"])

	// logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).EventName()
	// assert.Contains(t, logRecord.Body().AsString(), "Read traceV3 file")
	// assert.Contains(t, logRecord.Body().AsString(), testFile)

	err = rcv.Shutdown(context.Background())
	require.NoError(t, err, "failed to shutdown receiver")
}
