// // Copyright observIQ, Inc.
// //
// // Licensed under the Apache License, Version 2.0 (the "License");
// // you may not use this file except in compliance with the License.
// // You may obtain a copy of the License at
// //
// //      http://www.apache.org/licenses/LICENSE-2.0
// //
// // Unless required by applicable law or agreed to in writing, software
// // distributed under the License is distributed on an "AS IS" BASIS,
// // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// // See the License for the specific language governing permissions and
// // limitations under the License.

//go:build windows

package windowseventtracereceiver

// import (
// 	"context"
// 	"testing"

// 	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw"
// 	"github.com/stretchr/testify/mock"
// 	"go.opentelemetry.io/collector/consumer"
// 	"go.opentelemetry.io/collector/pdata/plog"
// )

// // setupMockSession creates a mock session for testing
// func setupMockSession(t *testing.T) *MockSession {
// 	mockSession := &MockSession{}
// 	mockSession.On("Start", mock.Anything).Return(nil)
// 	mockSession.On("Stop", mock.Anything).Return(nil)
// 	mockSession.On("EnableProvider", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
// 	return mockSession
// }

// // setupMockConsumer creates a mock consumer for testing
// func setupMockConsumer(t *testing.T) *MockConsumer {
// 	mockConsumer := NewMockConsumer()
// 	mockConsumer.On("Start", mock.Anything).Return(nil)
// 	mockConsumer.On("Stop", mock.Anything).Return(nil)
// 	mockConsumer.On("FromSessions", mock.Anything).Return(&etw.Consumer{
// 		Events: mockConsumer.MockEvents,
// 	})
// 	return mockConsumer
// }

// // mockLogsConsumer is a mock implementation of consumer.Logs
// type mockLogsConsumer struct {
// 	mock.Mock
// 	receivedLogs []plog.Logs
// }

// // ConsumeLogs is a mock implementation of consumer.Logs.ConsumeLogs
// func (m *mockLogsConsumer) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
// 	args := m.Called(ctx, logs)
// 	m.receivedLogs = append(m.receivedLogs, logs)
// 	return args.Error(0)
// }

// // Capabilities is a mock implementation of consumer.Logs.Capabilities
// func (m *mockLogsConsumer) Capabilities() consumer.Capabilities {
// 	args := m.Called()
// 	return args.Get(0).(consumer.Capabilities)
// }
