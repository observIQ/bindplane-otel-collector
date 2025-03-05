// Copyright  Bindplane, Inc.
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

package observiq

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/observiq/bindplane-otel-collector/collector"
	colmocks "github.com/observiq/bindplane-otel-collector/collector/mocks"
	"github.com/observiq/bindplane-otel-collector/opamp"
	"github.com/observiq/bindplane-otel-collector/opamp/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func Test_managerReload(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Invalid new config contents",
			testFunc: func(*testing.T) {
				client := &Client{
					logger: zap.NewNop(),
				}
				reloadFunc := managerReload(client, ManagerConfigName)

				badContents := []byte(`\t\t\t`)

				changed, err := reloadFunc(badContents)
				assert.ErrorContains(t, err, "failed to validate config")
				assert.False(t, changed)
			},
		},
		{
			desc: "No Changes to updatable fields",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()

				managerFilePath := filepath.Join(tmpDir, ManagerConfigName)
				client := &Client{
					logger: zap.NewNop(),
					currentConfig: opamp.Config{
						Endpoint: "ws://localhost:1234",
						AgentID:  testAgentID,
					},
				}
				reloadFunc := managerReload(client, managerFilePath)

				newContents, err := yaml.Marshal(client.currentConfig)
				assert.NoError(t, err)

				// Write new updates to file to ensure there's no changes
				err = os.WriteFile(managerFilePath, newContents, 0600)
				assert.NoError(t, err)

				changed, err := reloadFunc(newContents)
				assert.NoError(t, err)
				assert.False(t, changed)
			},
		},
		{
			desc: "Changes to updatable fields, successful update",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()

				managerFilePath := filepath.Join(tmpDir, ManagerConfigName)

				currConfig := &opamp.Config{
					Endpoint: "ws://localhost:1234",
					AgentID:  testAgentID,
				}

				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetAgentDescription", mock.Anything).Return(nil)

				client := &Client{
					logger:             zap.NewNop(),
					opampClient:        mockOpAmpClient,
					ident:              newIdentity(zap.NewNop(), *currConfig, "0.0.0"),
					currentConfig:      *currConfig,
					measurementsSender: newMeasurementsSender(zap.NewNop(), nil, mockOpAmpClient, 0, nil),
				}
				reloadFunc := managerReload(client, managerFilePath)

				currContents, err := yaml.Marshal(currConfig)
				assert.NoError(t, err)

				// Write new updates to file to ensure there's no changes
				err = os.WriteFile(managerFilePath, currContents, 0600)
				assert.NoError(t, err)

				// Create a new config data
				agentName := "name"
				newConfig := &opamp.Config{
					Endpoint:  "ws://localhost:1234",
					AgentID:   testAgentID,
					AgentName: &agentName,
				}

				newContents, err := yaml.Marshal(newConfig)
				assert.NoError(t, err)

				changed, err := reloadFunc(newContents)
				assert.NoError(t, err)
				assert.True(t, changed)

				// Verify client identity was updated
				assert.Equal(t, newConfig.AgentName, client.ident.agentName)
				assert.Equal(t, newConfig.AgentName, client.currentConfig.AgentName)

				// Verify new file was written
				data, err := os.ReadFile(managerFilePath)
				assert.NoError(t, err)
				assert.Equal(t, newContents, data)
			},
		},
		{
			desc: "Changes to updatable fields, failure occurs, rollback happens",
			testFunc: func(*testing.T) {
				tmpDir := t.TempDir()

				managerFilePath := filepath.Join(tmpDir, ManagerConfigName)

				currConfig := &opamp.Config{
					Endpoint: "ws://localhost:1234",
					AgentID:  testAgentID,
				}

				expectedErr := errors.New("oops")
				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetAgentDescription", mock.Anything).Return(expectedErr)

				client := &Client{
					logger:        zap.NewNop(),
					opampClient:   mockOpAmpClient,
					ident:         newIdentity(zap.NewNop(), *currConfig, "0.0.0"),
					currentConfig: *currConfig,
				}
				reloadFunc := managerReload(client, managerFilePath)

				currContents, err := yaml.Marshal(currConfig)
				assert.NoError(t, err)

				// Write new updates to file to ensure there's no changes
				err = os.WriteFile(managerFilePath, currContents, 0600)
				assert.NoError(t, err)

				// Create new config data
				agentName := "name"
				newConfig := &opamp.Config{
					Endpoint:  "ws://localhost:1234",
					AgentID:   testAgentID,
					AgentName: &agentName,
				}

				newContents, err := yaml.Marshal(newConfig)
				assert.NoError(t, err)

				changed, err := reloadFunc(newContents)
				assert.ErrorContains(t, err, "failed to set agent description")
				assert.False(t, changed)

				// Verify client identity was rolledback
				assert.Equal(t, currConfig.AgentName, client.ident.agentName)

				// Verify config rollback
				assert.Equal(t, client.currentConfig, *currConfig)

				// Verify config rolledback
				data, err := os.ReadFile(managerFilePath)
				assert.NoError(t, err)
				assert.Equal(t, currContents, data)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func Test_collectorReload(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Collector failed to restart, rollback required",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()

				collectorFilePath := filepath.Join(tmpDir, CollectorConfigName)

				expectedErr := errors.New("oops")
				statusChannel := make(chan *collector.Status)
				mockCollector := colmocks.NewMockCollector(t)
				mockCollector.On("Status").Return((<-chan *collector.Status)(statusChannel))
				mockCollector.On("Restart", mock.Anything).Return(expectedErr).Once()
				mockCollector.On("Restart", mock.Anything).Return(nil).Once()

				currContents := []byte("current: config")

				// Write Config file so we can verify it remained the same
				err := os.WriteFile(collectorFilePath, currContents, 0600)
				assert.NoError(t, err)

				client := &Client{
					logger:    zap.NewNop(),
					collector: mockCollector,
				}

				// Setup Context to mock out already running collector monitor
				client.collectorMntrCtx, client.collectorMntrCancel = context.WithCancel(context.Background())

				reloadFunc := collectorReload(client, collectorFilePath)

				changed, err := reloadFunc([]byte("valid: config"))
				assert.ErrorIs(t, err, expectedErr)
				assert.False(t, changed)

				// Verify config rolledback
				data, err := os.ReadFile(collectorFilePath)
				assert.NoError(t, err)
				assert.Equal(t, currContents, data)

				// Cleanup
				assert.Eventually(t, func() bool {
					client.stopCollectorMonitoring()
					return true
				}, 2*time.Second, 100*time.Millisecond)
			},
		},
		{
			desc: "Successful update",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()

				collectorFilePath := filepath.Join(tmpDir, CollectorConfigName)

				mockCollector := colmocks.NewMockCollector(t)
				statusChannel := make(chan *collector.Status)
				mockCollector.On("Status").Return((<-chan *collector.Status)(statusChannel))
				mockCollector.On("Restart", mock.Anything).Return(nil)

				currContents := []byte("current: config")

				// Write Config file so we can verify it remained the same
				err := os.WriteFile(collectorFilePath, currContents, 0600)
				assert.NoError(t, err)

				client := &Client{
					collector: mockCollector,
					logger:    zap.NewNop(),
				}

				// Setup Context to mock out already running collector monitor
				client.collectorMntrCtx, client.collectorMntrCancel = context.WithCancel(context.Background())

				reloadFunc := collectorReload(client, collectorFilePath)

				newContents := []byte("valid: config")
				changed, err := reloadFunc(newContents)
				assert.NoError(t, err)
				assert.True(t, changed)

				// Verify new config set
				data, err := os.ReadFile(collectorFilePath)
				assert.NoError(t, err)
				assert.Equal(t, newContents, data)

				// Cleanup
				assert.Eventually(t, func() bool {
					client.stopCollectorMonitoring()
					return true
				}, 2*time.Second, 100*time.Millisecond)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

// Test_loggerReload tests general cases since there are a lot of failure points with parsing the logging config
// We verify a success case and a case where the collector fails to accept the config
func Test_loggerReload(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Successful update",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()

				loggerFilePath := filepath.Join(tmpDir, LoggingConfigName)

				currContents := []byte("current: config")

				// Write Config file so we can verify it remained the same
				err := os.WriteFile(loggerFilePath, currContents, 0600)
				assert.NoError(t, err)

				mockCol := colmocks.NewMockCollector(t)
				mockCol.On("GetLoggingOpts").Return([]zap.Option{})
				mockCol.On("SetLoggingOpts", mock.Anything)
				mockCol.On("Restart", mock.Anything).Return(nil)

				client := &Client{
					logger:    zap.NewNop(),
					collector: mockCol,
				}

				reloadFunc := loggerReload(client, loggerFilePath)

				newContents := []byte("output: stdout\nlevel: debug")
				changed, err := reloadFunc(newContents)
				assert.NoError(t, err)
				assert.True(t, changed)

				// Verify config updated
				data, err := os.ReadFile(loggerFilePath)
				assert.NoError(t, err)
				assert.Equal(t, newContents, data)
				// Verify logger was set
				assert.NotNil(t, client.logger)
			},
		},
		{
			desc: "Collector fails to restart, rollback",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()

				loggerFilePath := filepath.Join(tmpDir, LoggingConfigName)

				currContents := []byte("current: config")

				// Write Config file so we can verify it remained the same
				err := os.WriteFile(loggerFilePath, currContents, 0600)
				assert.NoError(t, err)

				expectedErr := errors.New("oops")

				mockCol := colmocks.NewMockCollector(t)
				mockCol.On("GetLoggingOpts").Return([]zap.Option{})
				mockCol.On("SetLoggingOpts", mock.Anything)
				mockCol.On("Restart", mock.Anything).Return(expectedErr).Once()
				mockCol.On("Restart", mock.Anything).Return(nil).Once()

				currLogger := zap.NewNop()
				client := &Client{
					collector: mockCol,
					logger:    currLogger,
				}

				reloadFunc := loggerReload(client, loggerFilePath)

				newContents := []byte("output: stdout\nlevel: debug")
				changed, err := reloadFunc(newContents)
				assert.ErrorIs(t, err, expectedErr)
				assert.False(t, changed)

				// Verify config updated
				data, err := os.ReadFile(loggerFilePath)
				assert.NoError(t, err)
				assert.Equal(t, currContents, data)
				// Verify logger was set
				assert.Equal(t, currLogger, client.logger)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
