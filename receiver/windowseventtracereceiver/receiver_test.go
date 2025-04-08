//go:build windows

package windowseventtracereceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

func TestNewLogsReceiver(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg:  createTestConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := &consumertest.LogsSink{}
			logger := zap.NewNop()

			receiver, err := newLogsReceiver(tt.cfg, sink, logger)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, receiver)
		})
	}
}

func TestLogsReceiverStart(t *testing.T) {
	cfg := createTestConfig()
	sink := &consumertest.LogsSink{}
	logger := zap.NewNop()

	receiver, err := newLogsReceiver(cfg, sink, logger)
	require.NoError(t, err)

	// Create a mock session
	mockSession := setupMockSession(t)
	receiver.session = mockSession

	// Start the receiver
	err = receiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	// Ensure that session methods were called
	mockSession.AssertCalled(t, "Start", mock.Anything)

	// Clean up
	err = receiver.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		name       string
		levelName  string
		levelValue string
		want       plog.SeverityNumber
	}{
		{
			name:       "critical level",
			levelName:  "Critical",
			levelValue: "",
			want:       plog.SeverityNumberFatal,
		},
		{
			name:       "error level",
			levelName:  "Error",
			levelValue: "",
			want:       plog.SeverityNumberError,
		},
		{
			name:       "warning level",
			levelName:  "Warning",
			levelValue: "",
			want:       plog.SeverityNumberWarn,
		},
		{
			name:       "information level",
			levelName:  "Information",
			levelValue: "",
			want:       plog.SeverityNumberInfo,
		},
		{
			name:       "numeric level 1",
			levelName:  "",
			levelValue: "1",
			want:       plog.SeverityNumberFatal,
		},
		{
			name:       "numeric level 2",
			levelName:  "",
			levelValue: "2",
			want:       plog.SeverityNumberError,
		},
		{
			name:       "numeric level 3",
			levelName:  "",
			levelValue: "3",
			want:       plog.SeverityNumberWarn,
		},
		{
			name:       "numeric level 4",
			levelName:  "",
			levelValue: "4",
			want:       plog.SeverityNumberInfo,
		},
		{
			name:       "default level",
			levelName:  "Unknown",
			levelValue: "99",
			want:       plog.SeverityNumberInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSeverity(tt.levelName, tt.levelValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

// func TestParseEventData(t *testing.T) {
// 	cfg := createTestConfig()
// 	mockConsumer := &consumertest.LogsSink{}
// 	logger := zap.NewNop()

// 	receiver, err := newLogsReceiver(cfg, mockConsumer, logger)
// 	require.NoError(t, err)

// 	// Create a test event
// 	event := CreateMockEvent()

// 	// Create a log record
// 	logs := plog.NewLogs()
// 	resourceLog := logs.ResourceLogs().AppendEmpty()
// 	scopeLog := resourceLog.ScopeLogs().AppendEmpty()
// 	record := scopeLog.LogRecords().AppendEmpty()

// 	// Parse the event
// 	receiver.parseEventData(event, record)

// 	// Verify the parsed data
// 	assert.Equal(t, plog.SeverityNumberInfo, record.SeverityNumber())
// 	assert.Equal(t, event.System.TimeCreated.SystemTime.UnixNano(), int64(record.Timestamp()))

// 	// Verify body fields
// 	body := record.Body().Map().AsRaw()
// 	assert.Equal(t, "TestChannel", body["channel"])
// 	assert.Equal(t, "TestComputer", body["computer"])
// 	assert.Equal(t, "5678", body["thread_id"])

// 	// Verify level
// 	levelMap, ok := body["level"].(map[string]interface{})
// 	require.True(t, ok, "Expected level to be a map")
// 	assert.Equal(t, "Information", levelMap["name"])
// 	assert.Equal(t, "4", levelMap["value"])

// 	// Verify provider
// 	providerMap, ok := body["provider"].(map[string]interface{})
// 	require.True(t, ok, "Expected provider to be a map")
// 	assert.Equal(t, "TestProvider", providerMap["name"])
// 	assert.Equal(t, "12345678-1234-1234-1234-123456789012", providerMap["guid"])
// }
