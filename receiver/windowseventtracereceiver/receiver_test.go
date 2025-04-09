//go:build windows

package windowseventtracereceiver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw"
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
			require.NotNil(t, receiver)
			require.NoError(t, err)
		})
	}
}

func TestLogsReceiverStart(t *testing.T) {
	cfg := createTestConfig()
	sink := &consumertest.LogsSink{}
	logger := zap.NewNop()

	receiver, err := newLogsReceiver(cfg, sink, logger)
	require.NoError(t, err)

	err = receiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

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

func TestParseEventData(t *testing.T) {
	cfg := createTestConfig()
	mockConsumer := &consumertest.LogsSink{}
	logger := zap.NewNop()

	receiver, err := newLogsReceiver(cfg, mockConsumer, logger)
	require.NoError(t, err)

	// Create a test event
	event := createMockEvent()

	// Create a log record
	logs := plog.NewLogs()
	resourceLog := logs.ResourceLogs().AppendEmpty()
	scopeLog := resourceLog.ScopeLogs().AppendEmpty()
	record := scopeLog.LogRecords().AppendEmpty()

	// Parse the event
	receiver.parseEventData(event, record)

	// Verify the parsed data
	assert.Equal(t, plog.SeverityNumberInfo, record.SeverityNumber())
	assert.Equal(t, event.System.TimeCreated.SystemTime.UnixNano(), int64(record.Timestamp()))

	// Verify body fields
	body := record.Body().Map().AsRaw()
	assert.Equal(t, "TestChannel", body["channel"])
	assert.Equal(t, "TestComputer", body["computer"])
	assert.Equal(t, "5678", body["thread_id"])

	// Verify level
	levelMap, ok := body["level"].(map[string]interface{})
	require.True(t, ok, "Expected level to be a map")
	assert.Equal(t, "Information", levelMap["name"])
	assert.Equal(t, "4", levelMap["value"])

	// Verify provider
	providerMap, ok := body["provider"].(map[string]interface{})
	require.True(t, ok, "Expected provider to be a map")
	assert.Equal(t, "TestProvider", providerMap["name"])
	assert.Equal(t, "12345678-1234-1234-1234-123456789012", providerMap["guid"])
}

func createMockEvent() *etw.Event {
	return &etw.Event{
		System: etw.EventSystem{
			Provider: etw.EventProvider{
				Name: "TestProvider",
				GUID: "12345678-1234-1234-1234-123456789012",
			},
			Level: etw.EventLevel{
				Name:  "Information",
				Value: 4,
			},
			Task:     "TestTask",
			Opcode:   "TestOpcode",
			Keywords: "TestKeywords",
			TimeCreated: etw.EventTimeCreated{
				SystemTime: time.Now(),
			},
			EventID: "1234",
			Correlation: etw.EventCorrelation{
				ActivityID: "00000000-0000-0000-0000-000000000000",
			},
			Execution: etw.EventExecution{
				ProcessID: 1234,
				ThreadID:  5678,
			},
			Channel:  "TestChannel",
			Computer: "TestComputer",
		},
		EventData: map[string]any{
			"TestData": "TestValue",
		},
	}
}
