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
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.uber.org/zap"
)

var (
	logger        = zap.NewNop()
	config        = &Config{}
	settings      = component.TelemetrySettings{}
	id            = component.ID{}
	ssapireceiver = newSSAPIReceiver(logger, config, settings, id)
)

func TestPolling(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.JobPollInterval = 1 * time.Second
	ssapireceiver.config = cfg

	client := &mockLogsClient{}
	ssapireceiver.client = client

	file := filepath.Join("testdata", "logs", "testPollJobStatus", "input-done.xml")
	client.On("GetJobStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(client.loadTestStatusResponse(t, file), nil)
	cancelCtx, cancel := context.WithCancel(context.Background())
	ssapireceiver.cancel = cancel
	ssapireceiver.checkpointRecord = &EventRecord{}

	err := ssapireceiver.pollSearchCompletion(cancelCtx, "123456")
	require.NoError(t, err)
	client.AssertNumberOfCalls(t, "GetJobStatus", 1)

	// Test polling for a job that is still running
	file = filepath.Join("testdata", "logs", "testPollJobStatus", "input-queued.xml")
	client.On("GetJobStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(client.loadTestStatusResponse(t, file), nil)
	err = ssapireceiver.pollSearchCompletion(cancelCtx, "123456")
	require.NoError(t, err)
	client.AssertNumberOfCalls(t, "GetJobStatus", 2)

	err = ssapireceiver.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestIsSearchCompleted(t *testing.T) {
	jobResponse := SearchJobStatusResponse{
		Content: SearchJobContent{
			Dict: Dict{
				Keys: []Key{
					{
						Name:  "dispatchState",
						Value: "DONE",
					},
				},
			},
		},
	}

	emptyResponse := SearchJobStatusResponse{}

	done := ssapireceiver.isSearchCompleted(jobResponse)
	require.True(t, done)

	jobResponse.Content.Dict.Keys[0].Value = "RUNNING"
	done = ssapireceiver.isSearchCompleted(jobResponse)
	require.False(t, done)

	done = ssapireceiver.isSearchCompleted(emptyResponse)
	require.False(t, done)
}

func TestInitCheckpoint(t *testing.T) {
	mockStorage := &mockStorage{}
	searches := []Search{
		{
			Query: "index=otel",
		},
		{
			Query: "index=otel2",
		},
		{
			Query: "index=otel3",
		},
		{
			Query: "index=otel4",
		},
		{
			Query: "index=otel5",
		},
	}
	ssapireceiver.config.Searches = searches
	ssapireceiver.storageClient = mockStorage
	err := ssapireceiver.initCheckpoint(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, ssapireceiver.checkpointRecord.Offset)

	mockStorage.Value = []byte(`{"offset":5,"search":"index=otel3"}`)
	err = ssapireceiver.initCheckpoint(context.Background())
	require.NoError(t, err)
	require.Equal(t, 5, ssapireceiver.checkpointRecord.Offset)
	require.Equal(t, "index=otel3", ssapireceiver.checkpointRecord.Search)
}

func TestCheckpoint(t *testing.T) {
	mockStorage := &mockStorage{}
	ssapireceiver.storageClient = mockStorage
	mockStorage.On("Set", mock.Anything, eventStorageKey, mock.Anything).Return(nil)
	ssapireceiver.checkpointRecord = &EventRecord{
		Offset: 0,
		Search: "",
	}
	err := ssapireceiver.checkpoint(context.Background())
	require.NoError(t, err)
	mockStorage.AssertCalled(t, "Set", mock.Anything, eventStorageKey, []byte(`{"offset":0,"search":""}`))

	ssapireceiver.checkpointRecord = &EventRecord{
		Offset: 5,
		Search: "index=otel3",
	}

	err = ssapireceiver.checkpoint(context.Background())
	require.NoError(t, err)
	mockStorage.AssertCalled(t, "Set", mock.Anything, eventStorageKey, []byte(`{"offset":5,"search":"index=otel3"}`))
}

func TestLoadCheckpoint(t *testing.T) {
	mockStorage := &mockStorage{}
	ssapireceiver.storageClient = mockStorage
	mockStorage.Value = []byte(`{"offset":5,"search":"index=otel3"}`)
	err := ssapireceiver.loadCheckpoint(context.Background())
	require.NoError(t, err)
	require.Equal(t, 5, ssapireceiver.checkpointRecord.Offset)
	require.Equal(t, "index=otel3", ssapireceiver.checkpointRecord.Search)

	mockStorage.Value = []byte(`{"offset":10,"search":"index=otel4"}`)
	err = ssapireceiver.loadCheckpoint(context.Background())
	require.NoError(t, err)
	require.Equal(t, 10, ssapireceiver.checkpointRecord.Offset)
	require.Equal(t, "index=otel4", ssapireceiver.checkpointRecord.Search)

	mockStorage.Value = []byte(`{}`)
	err = ssapireceiver.loadCheckpoint(context.Background())
	require.NoError(t, err)
}

type mockLogsClient struct {
	mock.Mock
}

func (m *mockLogsClient) loadTestStatusResponse(t *testing.T, file string) SearchJobStatusResponse {
	logBytes, err := os.ReadFile(file)
	require.NoError(t, err)
	var resp SearchJobStatusResponse
	err = xml.Unmarshal(logBytes, &resp)
	require.NoError(t, err)
	return resp
}

func (m *mockLogsClient) GetJobStatus(searchID string) (SearchJobStatusResponse, error) {
	args := m.Called(searchID)
	return args.Get(0).(SearchJobStatusResponse), args.Error(1)
}

func (m *mockLogsClient) CreateSearchJob(searchQuery string) (CreateJobResponse, error) {
	args := m.Called(searchQuery)
	return args.Get(0).(CreateJobResponse), args.Error(1)
}

func (m *mockLogsClient) GetSearchResults(searchID string, offset int, batchSize int) (SearchResults, error) {
	args := m.Called(searchID, offset, batchSize)
	return args.Get(0).(SearchResults), args.Error(1)
}

type mockStorage struct {
	mock.Mock
	Key   string
	Value []byte
}

func (m *mockStorage) Get(_ context.Context, _ string) ([]byte, error) {
	return []byte(m.Value), nil
}

func (m *mockStorage) Set(ctx context.Context, key string, value []byte) error {
	args := m.Called(ctx, key, value)
	m.Key = key
	m.Value = value
	return args.Error(0)
}

func (m *mockStorage) Batch(_ context.Context, _ ...storage.Operation) error {
	return nil
}

func (m *mockStorage) Close(_ context.Context) error {
	return nil
}

func (m *mockStorage) Delete(_ context.Context, _ string) error {
	return nil
}
