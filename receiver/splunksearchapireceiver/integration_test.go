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

package splunksearchapireceiver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// Test the case where some data is exported, but a subsequent call for paginated data fails
func TestSplunkResultsPaginationFailure(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Searches = []Search{
		{
			Query:          "search index=otel",
			EarliestTime:   "2024-11-14T00:00:00.000Z",
			LatestTime:     "2024-11-14T23:59:59.000Z",
			EventBatchSize: 5,
		},
	}
	var callCount int
	server := newMockSplunkServerPagination(&callCount)
	defer server.Close()
	settings := componenttest.NewNopTelemetrySettings()
	ssapir := newSSAPIReceiver(zap.NewNop(), cfg, settings, component.NewID(typeStr))
	ssapir.client, _ = newDefaultSplunkSearchAPIClient(context.Background(), settings, *cfg, componenttest.NewNopHost())
	ssapir.client.(*defaultSplunkSearchAPIClient).client = server.Client()
	ssapir.client.(*defaultSplunkSearchAPIClient).endpoint = server.URL
	ssapir.logsConsumer = &consumertest.LogsSink{}

	ssapir.storageClient = storage.NewNopClient()

	ssapir.initCheckpoint(context.Background())
	ssapir.runQueries(context.Background())
	require.Equal(t, 3, ssapir.checkpointRecord.Offset)
	require.Equal(t, 1, callCount)
}

func newMockSplunkServerPagination(callCount *int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/services/search/jobs" {
			rw.Header().Set("Content-Type", "application/xml")
			rw.WriteHeader(201)
			rw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
			<response>
				<sid>123456</sid>
			</response>
			`))
		} else if req.URL.String() == "/services/search/v2/jobs/123456" {
			rw.Header().Set("Content-Type", "application/xml")
			rw.WriteHeader(200)
			rw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
			<response>
				<content>
					<type>DISPATCH</type>
					<dict>
						<key name="dispatchState">DONE</key>
					</dict>
				</content>
			</response>`))
		} else if req.URL.String() == "/services/search/v2/jobs/123456/results?output_mode=json&offset=0&count=5" && req.URL.Query().Get("offset") == "0" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(200)
			rw.Write(splunkEventsResultsP1)
			*callCount++
		} else if req.URL.String() == "/services/search/v2/jobs/123456/results?output_mode=json&offset=5&count=5" && req.URL.Query().Get("offset") == "5" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(400)
			rw.Write([]byte("error, bad request"))
		}
	}))
}

// Test the case where the GCP exporter returns an error
func TestExporterFailure(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Searches = []Search{
		{
			Query:          "search index=otel",
			EarliestTime:   "2024-11-14T00:00:00.000Z",
			LatestTime:     "2024-11-14T23:59:59.000Z",
			EventBatchSize: 3,
		},
	}
	server := newMockSplunkServer()
	defer server.Close()
	settings := componenttest.NewNopTelemetrySettings()
	ssapir := newSSAPIReceiver(zap.NewNop(), cfg, settings, component.NewID(typeStr))
	logsConsumer := &mockLogsConsumerExporterErr{}
	logsConsumer.On("ConsumeLogs", mock.Anything, mock.Anything).Return(nil)

	ssapir.logsConsumer = logsConsumer
	ssapir.client, _ = newDefaultSplunkSearchAPIClient(context.Background(), settings, *cfg, componenttest.NewNopHost())
	ssapir.client.(*defaultSplunkSearchAPIClient).client = server.Client()
	ssapir.client.(*defaultSplunkSearchAPIClient).endpoint = server.URL

	ssapir.initCheckpoint(context.Background())
	ssapir.runQueries(context.Background())
	require.Equal(t, 5, ssapir.checkpointRecord.Offset)
	require.Equal(t, "search index=otel", ssapir.checkpointRecord.Search)

	// simulate data failing
	// the checkpoint should not be updated, and an error should be returned
	ssapir.checkpointRecord.Offset = 0
	offset = 0
	logsConsumerErr := &mockLogsConsumerExporterErr{}
	logsConsumerErr.On("ConsumeLogs", mock.Anything, mock.Anything).Return(errors.New("error exporting logs"))

	ssapir.logsConsumer = logsConsumerErr
	ssapir.initCheckpoint(context.Background())
	ssapir.runQueries(context.Background())
	require.Equal(t, 0, ssapir.checkpointRecord.Offset)
	require.Equal(t, "search index=otel", ssapir.checkpointRecord.Search)
}

func newMockSplunkServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/services/search/jobs" {
			rw.Header().Set("Content-Type", "application/xml")
			rw.WriteHeader(201)
			rw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
			<response>
				<sid>123456</sid>
			</response>
			`))
		} else if req.URL.String() == "/services/search/v2/jobs/123456" {
			rw.Header().Set("Content-Type", "application/xml")
			rw.WriteHeader(200)
			rw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
			<response>
				<content>
					<type>DISPATCH</type>
					<dict>
						<key name="dispatchState">DONE</key>
					</dict>
				</content>
			</response>`))
		} else if req.URL.String() == "/services/search/v2/jobs/123456/results?output_mode=json&offset=0&count=3" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(200)
			rw.Write(splunkEventsResultsP1)
		} else if req.URL.String() == "/services/search/v2/jobs/123456/results?output_mode=json&offset=3&count=3" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(200)
			rw.Write(splunkEventsResultsP2)
		}
	}))
}

var splunkEventsResultsP1 = []byte(`{
	"init_offset": 0,
	"results": [
		{
			"_raw": "Hello, world!",
			"_time": "2024-11-14T13:02:31.000-05:00"
		},
		{
			"_raw": "Goodbye, world!",
			"_time": "2024-11-14T13:02:30.000-05:00"
		},
		{
			"_raw": "lorem ipsum",
			"_time": "2024-11-14T13:02:29.000-05:00"
		}
	]
}`)

var splunkEventsResultsP2 = []byte(`{
	"init_offset": 3,
	"results": [
		{
			"_raw": "dolor sit amet",
			"_time": "2024-11-14T13:02:28.000-05:00"
		},
		{
			"_raw": "consectetur adipiscing elit",
			"_time": "2024-11-14T13:02:27.000-05:00"
		}
	]
}`)

type mockLogsConsumerExporterErr struct {
	mock.Mock
}

func (m *mockLogsConsumerExporterErr) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	args := m.Called(ctx, logs)
	return args.Error(0)
}

func (m *mockLogsConsumerExporterErr) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
