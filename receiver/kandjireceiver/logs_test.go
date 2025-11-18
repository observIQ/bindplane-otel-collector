package kandjireceiver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// ----------------------------------------------------------------------
// In-Memory Storage Client for Testing
// ----------------------------------------------------------------------

type inMemoryStorageClient struct {
	data map[string][]byte
}

func newInMemoryStorageClient() *inMemoryStorageClient {
	return &inMemoryStorageClient{
		data: make(map[string][]byte),
	}
}

func (m *inMemoryStorageClient) Set(ctx context.Context, key string, value []byte) error {
	m.data[key] = value
	return nil
}

func (m *inMemoryStorageClient) Get(ctx context.Context, key string) ([]byte, error) {
	val, ok := m.data[key]
	if !ok {
		return nil, nil
	}
	return val, nil
}

func (m *inMemoryStorageClient) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *inMemoryStorageClient) Batch(ctx context.Context, ops ...*storage.Operation) error {
	for _, op := range ops {
		switch op.Type {
		case storage.Set:
			m.data[op.Key] = op.Value
		case storage.Delete:
			delete(m.data, op.Key)
		}
	}
	return nil
}

func (m *inMemoryStorageClient) Close(ctx context.Context) error {
	return nil
}

// ----------------------------------------------------------------------
// Mock Client
// ----------------------------------------------------------------------

type mockKandjiClient struct {
	mock.Mock
}

func (m *mockKandjiClient) CallAPI(ctx context.Context, ep KandjiEndpoint, params map[string]any, out any) (int, error) {
	args := m.Called(ctx, ep, params, out)
	statusCode := args.Int(0)
	err := args.Error(1)

	// The response is handled by the Run() function in the mock setup
	// which copies data into the 'out' parameter before this returns

	return statusCode, err
}

func (m *mockKandjiClient) Shutdown() error {
	args := m.Called()
	return args.Error(0)
}

// ----------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------

func makeFakeAuditEvent(id string) AuditEvent {
	return AuditEvent{
		ID:              id,
		Action:          "login",
		ActorID:         "42",
		ActorType:       "user",
		TargetID:        "asset-123",
		TargetType:      "device",
		TargetComponent: "agent",
		OccurredAt:      time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func makeFakeResponse(n int) *AuditEventsResponse {
	resp := &AuditEventsResponse{}
	for i := 0; i < n; i++ {
		resp.Results = append(resp.Results, makeFakeAuditEvent("event-"+time.Now().Format("150405")+string(rune(i))))
	}
	return resp
}

// ----------------------------------------------------------------------
// Test: emitLogs
// ----------------------------------------------------------------------

func TestEmitLogsEmitsRecords(t *testing.T) {
	cfg := createMinimalConfig()
	sink := &consumertest.LogsSink{}

	receiver := newKandjiLogs(cfg, receivertest.NewNopSettings(typ), sink)

	resp := makeFakeResponse(3)

	err := receiver.emitLogs("GET /audit/events", resp)
	require.NoError(t, err)

	require.Equal(t, 3, sink.LogRecordCount())

	rl := sink.AllLogs()[0].ResourceLogs()
	require.Equal(t, 1, rl.Len())

	attrs := rl.At(0).Resource().Attributes()
	_, ok := attrs.Get("kandji.region")
	require.True(t, ok)
}

// ----------------------------------------------------------------------
// Test: emitLogs with nil consumer
// ----------------------------------------------------------------------

func TestEmitLogsConsumerNil(t *testing.T) {
	cfg := createMinimalConfig()
	receiver := newKandjiLogs(cfg, receivertest.NewNopSettings(typ), nil)

	resp := makeFakeResponse(2)

	err := receiver.emitLogs("GET /audit/events", resp)
	require.Error(t, err)
}

// ----------------------------------------------------------------------
// Test: pollEndpoint calls fetchPage + emitLogs
// ----------------------------------------------------------------------

func TestPollEndpoint(t *testing.T) {
	cfg := createMinimalConfig()
	sink := &consumertest.LogsSink{}
	l := newKandjiLogs(cfg, receivertest.NewNopSettings(typ), sink)

	mockClient := &mockKandjiClient{}
	l.client = mockClient

	resp := makeFakeResponse(2)

	mockClient.
		On("CallAPI", mock.Anything, KandjiEndpoint("GET /audit/events"), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// Copy response into the output parameter
			if out, ok := args.Get(3).(*AuditEventsResponse); ok {
				*out = *resp
			}
		}).
		Return(200, nil)
	mockClient.On("Shutdown").Return(nil)

	spec := EndpointSpec{
		ResponseType: AuditEventsResponse{},
		Params:       []ParamSpec{{Name: "limit"}, {Name: "sort_by"}},
	}

	err := l.pollEndpoint(context.Background(), "GET /audit/events", spec)
	require.NoError(t, err)
	require.Equal(t, 2, sink.LogRecordCount())
}

// ----------------------------------------------------------------------
// Test: pollAll filters endpoints + emits logs
// ----------------------------------------------------------------------

func TestPollAll(t *testing.T) {
	// Save original registry
	originalRegistry := EndpointRegistry
	defer func() {
		EndpointRegistry = originalRegistry
	}()

	cfg := createMinimalConfig()
	sink := &consumertest.LogsSink{}
	l := newKandjiLogs(cfg, receivertest.NewNopSettings(typ), sink)

	mockClient := &mockKandjiClient{}
	l.client = mockClient

	// Override registry locally
	EndpointRegistry = map[KandjiEndpoint]EndpointSpec{
		"GET /audit/events": {
			ResponseType: AuditEventsResponse{},
			Params:       []ParamSpec{{Name: "limit"}, {Name: "sort_by"}},
		},
		"GET /devices": { // Should be skipped
			ResponseType: struct{}{},
		},
	}

	resp := makeFakeResponse(1)

	mockClient.
		On("CallAPI", mock.Anything, KandjiEndpoint("GET /audit/events"), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			// Copy response into the output parameter
			if out, ok := args.Get(3).(*AuditEventsResponse); ok {
				*out = *resp
			}
		}).
		Return(200, nil)
	mockClient.On("Shutdown").Return(nil)

	err := l.pollAll(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, sink.LogRecordCount())
}

// ----------------------------------------------------------------------
// Test: pollEndpoint handles API error
// ----------------------------------------------------------------------

func TestPollEndpointError(t *testing.T) {
	cfg := createMinimalConfig()
	sink := &consumertest.LogsSink{}
	l := newKandjiLogs(cfg, receivertest.NewNopSettings(typ), sink)

	mockClient := &mockKandjiClient{}
	l.client = mockClient

	mockClient.
		On("CallAPI", mock.Anything, KandjiEndpoint("GET /audit/events"), mock.Anything, mock.Anything).
		Return(500, errors.New("boom"))
	mockClient.On("Shutdown").Return(nil)

	spec := EndpointSpec{
		ResponseType: AuditEventsResponse{},
		Params:       []ParamSpec{{Name: "limit"}},
	}

	err := l.pollEndpoint(context.Background(), "GET /audit/events", spec)
	require.Error(t, err)
	require.Equal(t, 0, sink.LogRecordCount())
}

// ----------------------------------------------------------------------
// Test: Checkpoint Save + Load
// ----------------------------------------------------------------------

func TestCheckpointSaveLoad(t *testing.T) {
	// Save original registry
	originalRegistry := EndpointRegistry
	defer func() {
		EndpointRegistry = originalRegistry
	}()

	// Ensure the endpoint is in the registry for loadCheckpoint to find it
	EndpointRegistry = map[KandjiEndpoint]EndpointSpec{
		"GET /audit/events": {
			ResponseType: AuditEventsResponse{},
		},
	}

	cfg := createMinimalConfig()
	sink := &consumertest.LogsSink{}
	l := newKandjiLogs(cfg, receivertest.NewNopSettings(typ), sink)

	// Use in-memory storage client that actually stores data
	mem := newInMemoryStorageClient()
	l.storageClient = mem

	ep := KandjiEndpoint("GET /audit/events")
	// Use a cursor value that will pass normalizeCursor validation
	// normalizeCursor calls sanitizeCursor which just trims and returns the value
	// if it doesn't match URL patterns, so "abc123" should work fine
	cursor := "abc123"
	l.cursors[ep] = &cursor

	err := l.checkpoint(context.Background())
	require.NoError(t, err)

	// Verify it was saved
	key := logStorageKeyPrefix + string(ep)
	saved, err := mem.Get(context.Background(), key)
	require.NoError(t, err)
	require.NotNil(t, saved)

	// Clear then reload
	l.cursors = map[KandjiEndpoint]*string{}
	l.loadCheckpoint(context.Background())

	require.NotNil(t, l.cursors[ep], "cursor should be loaded from storage")
	require.Equal(t, "abc123", *l.cursors[ep])
}

// ----------------------------------------------------------------------
// Minimal config helper
// ----------------------------------------------------------------------

func createMinimalConfig() *Config {
	storageID := component.MustNewID("kandji")
	return &Config{
		SubDomain:    "test",
		Region:       "us",
		ApiKey:       "123",
		BaseHost:     "api.kandji.io",
		StorageID:    &storageID,
		Logs:         LogsConfig{PollInterval: 10 * time.Millisecond},
		ClientConfig: confighttp.ClientConfig{
			// For tests, the ClientConfig is unused, mock client is injected
		},
	}
}
