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
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/observiq/bindplane-otel-collector/collector"
	colmocks "github.com/observiq/bindplane-otel-collector/collector/mocks"
	"github.com/observiq/bindplane-otel-collector/internal/report"
	"github.com/observiq/bindplane-otel-collector/internal/version"
	"github.com/observiq/bindplane-otel-collector/opamp"
	"github.com/observiq/bindplane-otel-collector/opamp/mocks"
	"github.com/observiq/bindplane-otel-collector/packagestate"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewClient(t *testing.T) {
	secretKey := "136bdd08-2074-40b7-ac1c-6706ac24c4f2"
	badCAFile := "bad"
	testCases := []struct {
		desc        string
		config      opamp.Config
		expectedErr error
	}{
		{
			desc: "Bad URL Scheme",
			config: opamp.Config{
				Endpoint: "http://localhost:1234",
				AgentID:  testAgentID,
			},
			expectedErr: ErrUnsupportedURL,
		},
		{
			desc: "Invalid Endpoint",
			config: opamp.Config{
				Endpoint: "\t\t\t",
				AgentID:  testAgentID,
			},
			expectedErr: errors.New("net/url: invalid control character in URL"),
		},
		{
			desc: "Bad TLS Config",
			config: opamp.Config{
				Endpoint: "ws://localhost:1234",
				AgentID:  testAgentID,
				TLS: &opamp.TLSConfig{
					CAFile: &badCAFile,
				},
			},
			expectedErr: errors.New("failed creating TLS config"),
		},
		{
			desc: "Valid Config",
			config: opamp.Config{
				Endpoint:  "ws://localhost:1234",
				AgentID:   testAgentID,
				SecretKey: &secretKey,
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			testLogger := zap.NewNop()
			mockCollector := colmocks.NewMockCollector(t)

			tmpDir := t.TempDir()

			managerPath := filepath.Join(tmpDir, "manager.yaml")
			managerFile, err := os.Create(managerPath)
			assert.NoError(t, err)

			collectorPath := filepath.Join(tmpDir, "collector.yaml")
			collectorFile, err := os.Create(collectorPath)
			assert.NoError(t, err)

			loggerPath := filepath.Join(tmpDir, "logger.yaml")
			loggerFile, err := os.Create(loggerPath)
			assert.NoError(t, err)

			// We need to close the files specifically so windows can clean up the tmp dir
			defer func() {
				err := managerFile.Close()
				assert.NoError(t, err)
				err = collectorFile.Close()
				assert.NoError(t, err)
				err = loggerFile.Close()
				assert.NoError(t, err)
			}()

			args := &NewClientArgs{
				DefaultLogger:       testLogger,
				Config:              tc.config,
				Collector:           mockCollector,
				ManagerConfigPath:   managerPath,
				CollectorConfigPath: collectorPath,
				LoggerConfigPath:    loggerPath,
			}

			actual, err := NewClient(args)

			if tc.expectedErr != nil {
				assert.ErrorContains(t, err, tc.expectedErr.Error())
				assert.Nil(t, actual)
			} else {
				assert.NoError(t, err)

				observiqClient, ok := actual.(*Client)
				require.True(t, ok)

				// Do a shallow check on all fields to assert they exist and are equal to passed in params were possible
				assert.NotNil(t, observiqClient.opampClient)
				assert.NotNil(t, observiqClient.configManager)
				assert.NotNil(t, observiqClient.packagesStateProvider)
				assert.Equal(t, testLogger.Named("opamp"), observiqClient.logger)
				assert.Equal(t, mockCollector, observiqClient.collector)
				assert.NotNil(t, observiqClient.ident)
				assert.Equal(t, report.GetManager(), observiqClient.reportManager)
				assert.Equal(t, observiqClient.currentConfig, tc.config)
				assert.False(t, observiqClient.safeGetDisconnecting())
				assert.False(t, observiqClient.safeGetUpdatingPackage())
			}

		})
	}
}

func TestClientConnect(t *testing.T) {
	secretKeyContents := "136bdd08-2074-40b7-ac1c-6706ac24c4f2"
	allHash := []byte("allHash")
	hash := []byte("hash")
	newHash := []byte("newHash")
	newVersion := "99.99.99"

	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "SetAgentDescription fails",
			testFunc: func(*testing.T) {
				statuses := map[string]*protobufs.PackageStatus{
					packagestate.CollectorPackageName: {
						Name:                 packagestate.CollectorPackageName,
						AgentHasVersion:      version.Version(),
						AgentHasHash:         hash,
						ServerOfferedVersion: newVersion,
						ServerOfferedHash:    newHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installing,
					},
				}
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					Packages:                      statuses,
				}
				expectedErr := errors.New("oops")

				mockOpAmpClient := new(mocks.MockOpAMPClient)
				mockOpAmpClient.On("SetAgentDescription", mock.Anything).Return(expectedErr)
				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockStateProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)
					assert.Equal(t, "Failed setting agent description: oops", status.Packages[packagestate.CollectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, status.Packages[packagestate.CollectorPackageName].Status)
				})

				c := &Client{
					opampClient:   mockOpAmpClient,
					logger:        zap.NewNop(),
					ident:         &identity{},
					configManager: nil,
					collector:     nil,
					currentConfig: opamp.Config{
						Endpoint:  "ws://localhost:1234",
						SecretKey: &secretKeyContents,
					},
					packagesStateProvider: mockStateProvider,
				}

				err := c.Connect(context.Background())
				assert.ErrorIs(t, err, expectedErr)
			},
		},
		{
			desc: "TLS fails",
			testFunc: func(*testing.T) {
				statuses := map[string]*protobufs.PackageStatus{
					packagestate.CollectorPackageName: {
						Name:                 packagestate.CollectorPackageName,
						AgentHasVersion:      version.Version(),
						AgentHasHash:         hash,
						ServerOfferedVersion: newVersion,
						ServerOfferedHash:    newHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installing,
					},
				}
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					Packages:                      statuses,
				}

				mockOpAmpClient := new(mocks.MockOpAMPClient)
				mockOpAmpClient.On("SetAgentDescription", mock.Anything).Return(nil)
				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockStateProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)
					assert.Contains(t, status.Packages[packagestate.CollectorPackageName].ErrorMessage, "Failed creating TLS config: failed to read CA file: open bad-ca.cert:")
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, status.Packages[packagestate.CollectorPackageName].Status)
				})
				badCAFile := "bad-ca.cert"

				c := &Client{
					opampClient:   mockOpAmpClient,
					logger:        zap.NewNop(),
					ident:         &identity{},
					configManager: nil,
					collector:     nil,
					currentConfig: opamp.Config{
						Endpoint:  "ws://localhost:1234",
						SecretKey: &secretKeyContents,
						TLS: &opamp.TLSConfig{
							CAFile: &badCAFile,
						},
					},
					packagesStateProvider: mockStateProvider,
				}

				err := c.Connect(context.Background())
				assert.Error(t, err)
			},
		},
		{
			desc: "Collector fails to start",
			testFunc: func(*testing.T) {
				statuses := map[string]*protobufs.PackageStatus{
					packagestate.CollectorPackageName: {
						Name:                 packagestate.CollectorPackageName,
						AgentHasVersion:      version.Version(),
						AgentHasHash:         hash,
						ServerOfferedVersion: newVersion,
						ServerOfferedHash:    newHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installing,
					},
				}
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					Packages:                      statuses,
				}

				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetAgentDescription", mock.Anything).Return(nil)
				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockStateProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)
					assert.Equal(t, "Collector failed to start: oops", status.Packages[packagestate.CollectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, status.Packages[packagestate.CollectorPackageName].Status)
				})

				expectedErr := errors.New("oops")

				mockCollector := colmocks.NewMockCollector(t)
				mockCollector.On("Run", mock.Anything).Return(expectedErr)

				c := &Client{
					opampClient:   mockOpAmpClient,
					logger:        zap.NewNop(),
					ident:         &identity{agentID: testAgentID},
					configManager: nil,
					collector:     mockCollector,
					currentConfig: opamp.Config{
						Endpoint:  "ws://localhost:1234",
						SecretKey: &secretKeyContents,
					},
					packagesStateProvider: mockStateProvider,
				}

				err := c.Connect(context.Background())
				assert.ErrorIs(t, err, expectedErr)
			},
		},
		{
			desc: "Start fails",
			testFunc: func(*testing.T) {
				statuses := map[string]*protobufs.PackageStatus{
					packagestate.CollectorPackageName: {
						Name:                 packagestate.CollectorPackageName,
						AgentHasVersion:      version.Version(),
						AgentHasHash:         hash,
						ServerOfferedVersion: newVersion,
						ServerOfferedHash:    newHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installing,
					},
				}
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					Packages:                      statuses,
				}
				expectedErr := errors.New("oops")

				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetAgentDescription", mock.Anything).Return(nil)
				mockOpAmpClient.On("Start", mock.Anything, mock.Anything).Return(expectedErr)
				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockStateProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)
					assert.Equal(t, "OpAMP client failed to start: oops", status.Packages[packagestate.CollectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, status.Packages[packagestate.CollectorPackageName].Status)
				})

				statusChannel := make(chan *collector.Status)
				mockCollector := colmocks.NewMockCollector(t)
				mockCollector.On("Run", mock.Anything).Return(nil)
				mockCollector.On("Status").Return((<-chan *collector.Status)(statusChannel))

				c := &Client{
					opampClient:   mockOpAmpClient,
					logger:        zap.NewNop(),
					ident:         &identity{agentID: testAgentID},
					configManager: nil,
					collector:     mockCollector,
					currentConfig: opamp.Config{
						Endpoint:  "ws://localhost:1234",
						SecretKey: &secretKeyContents,
					},
					packagesStateProvider: mockStateProvider,
				}

				err := c.Connect(context.Background())
				assert.ErrorIs(t, err, expectedErr)

				// Cleanup
				c.collectorMntrCancel()
				assert.Eventually(t, func() bool {
					c.collectorMntrWg.Wait()
					return true
				}, 2*time.Second, 100*time.Millisecond)
			},
		},
		{
			desc: "Connect successful",
			testFunc: func(*testing.T) {
				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetAgentDescription", mock.Anything).Return(nil)

				statusChannel := make(chan *collector.Status)
				mockCollector := colmocks.NewMockCollector(t)
				mockCollector.On("Run", mock.Anything).Return(nil)
				mockCollector.On("Status").Return((<-chan *collector.Status)(statusChannel))

				mockPackagesStateProvider := mocks.NewMockPackagesStateProvider(t)

				c := &Client{
					opampClient: mockOpAmpClient,
					logger:      zap.NewNop(),
					ident: &identity{
						agentID:  testAgentID,
						hostname: "my.localnet",
					},
					configManager: nil,
					collector:     mockCollector,
					currentConfig: opamp.Config{
						Endpoint:  "ws://localhost:1234",
						SecretKey: &secretKeyContents,
					},
					packagesStateProvider: mockPackagesStateProvider,
				}

				expectedSettings := types.StartSettings{
					OpAMPServerURL: c.currentConfig.Endpoint,
					Header: http.Header{
						"Authorization":               []string{fmt.Sprintf("Secret-Key %s", c.currentConfig.GetSecretKey())},
						"User-Agent":                  []string{fmt.Sprintf("observiq-otel-collector/%s", version.Version())},
						"OpAMP-Version":               []string{opamp.Version()},
						"Agent-ID":                    []string{c.ident.agentID.String()},
						"Agent-Version":               []string{version.Version()},
						"Agent-Hostname":              []string{c.ident.hostname},
						"X-Bindplane-Agent-Id-Format": []string{"ULID"},
					},
					TLSConfig:   nil,
					InstanceUid: c.ident.agentID.OpAMPInstanceUID(),
					Callbacks: types.Callbacks{
						OnConnect:          c.onConnectHandler,
						OnConnectFailed:    c.onConnectFailedHandler,
						OnError:            c.onErrorHandler,
						OnMessage:          c.onMessageFuncHandler,
						GetEffectiveConfig: c.onGetEffectiveConfigHandler,
					},
					PackagesStateProvider: c.packagesStateProvider,
				}
				mockOpAmpClient.On("Start", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					settings := args.Get(1).(types.StartSettings)
					assert.Equal(t, expectedSettings.OpAMPServerURL, settings.OpAMPServerURL)
					assert.Equal(t, expectedSettings.Header, settings.Header)
					assert.Equal(t, expectedSettings.TLSConfig, settings.TLSConfig)
					assert.Equal(t, expectedSettings.InstanceUid, settings.InstanceUid)
					assert.Equal(t, expectedSettings.PackagesStateProvider, settings.PackagesStateProvider)
					// assert is unable to compare function pointers
				})

				err := c.Connect(context.Background())
				assert.NoError(t, err)

				// Cleanup
				c.collectorMntrCancel()
				assert.Eventually(t, func() bool {
					c.collectorMntrWg.Wait()
					return true
				}, 2*time.Second, 100*time.Millisecond)
			},
		},
		{
			desc: "Problem connecting & not installing",
			testFunc: func(*testing.T) {
				statuses := map[string]*protobufs.PackageStatus{
					packagestate.CollectorPackageName: {
						Name:                 packagestate.CollectorPackageName,
						AgentHasVersion:      version.Version(),
						ServerOfferedVersion: version.Version(),
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
					},
				}
				packageStatuses := &protobufs.PackageStatuses{
					Packages: statuses,
				}

				expectedErr := errors.New("oops")

				mockOpAmpClient := new(mocks.MockOpAMPClient)
				mockOpAmpClient.On("SetAgentDescription", mock.Anything).Return(expectedErr)
				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(packageStatuses, nil)

				c := &Client{
					opampClient:   mockOpAmpClient,
					logger:        zap.NewNop(),
					ident:         &identity{},
					configManager: nil,
					collector:     nil,
					currentConfig: opamp.Config{
						Endpoint:  "ws://localhost:1234",
						SecretKey: &secretKeyContents,
					},
					packagesStateProvider: mockStateProvider,
				}

				err := c.Connect(context.Background())
				assert.ErrorIs(t, err, expectedErr)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestClientDisconnect(t *testing.T) {
	ctx := context.Background()
	mockOpAmpClient := new(mocks.MockOpAMPClient)
	mockOpAmpClient.On("Stop", ctx).Return(nil)
	mockCollector := colmocks.NewMockCollector(t)
	statusChan := make(chan *collector.Status)
	mockCollector.On("Status").Return((<-chan *collector.Status)(statusChan))
	mockCollector.On("Stop", ctx).Return()

	c := &Client{
		opampClient:   mockOpAmpClient,
		collector:     mockCollector,
		reportManager: report.GetManager(),
		logger:        zap.NewNop(),
	}

	// Start collector monitoring to ensure this is shut down properly
	c.startCollectorMonitoring(ctx)

	var err error
	waitFunc := func() bool {
		err = c.Disconnect(ctx)
		return true
	}

	require.Eventually(t, waitFunc, time.Second*2, time.Millisecond*100)
	assert.NoError(t, err)
	assert.True(t, c.safeGetDisconnecting())
	mockOpAmpClient.AssertExpectations(t)
}

func TestClient_onConnectHandler(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "LastReportedStatus error",
			testFunc: func(*testing.T) {
				expectedErr := errors.New("oops")

				mockOpAmpClient := new(mocks.MockOpAMPClient)
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil)
				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(nil, expectedErr)

				c := &Client{
					opampClient:           mockOpAmpClient,
					logger:                zap.NewNop(),
					packagesStateProvider: mockStateProvider,
				}

				c.onConnectHandler(context.Background())
			},
		},
		{
			desc: "LastReportedStatus no main package info",
			testFunc: func(*testing.T) {
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: []byte("allHash"),
					Packages:                      make(map[string]*protobufs.PackageStatus),
				}

				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(packageStatuses, nil)

				c := &Client{
					logger:                zap.NewNop(),
					packagesStateProvider: mockStateProvider,
				}

				c.onConnectHandler(context.Background())
			},
		},
		{
			desc: "Good LastReportedStatus but not installing",
			testFunc: func(*testing.T) {
				allHash := []byte("allHash")
				hash := []byte("hash")
				newHash := []byte("newHash")
				newVersion := "99.99.99"
				errorMessage := "problem"
				statuses := map[string]*protobufs.PackageStatus{
					packagestate.CollectorPackageName: {
						Name:                 packagestate.CollectorPackageName,
						AgentHasVersion:      version.Version(),
						AgentHasHash:         hash,
						ServerOfferedVersion: newVersion,
						ServerOfferedHash:    newHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed,
						ErrorMessage:         errorMessage,
					},
				}
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					Packages:                      statuses,
				}

				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(packageStatuses, nil)

				c := &Client{
					logger:                zap.NewNop(),
					packagesStateProvider: mockStateProvider,
				}

				c.onConnectHandler(context.Background())
			},
		},
		{
			desc: "Installing with version mismatch",
			testFunc: func(*testing.T) {
				allHash := []byte("allHash")
				hash := []byte("hash")
				newHash := []byte("newHash")
				newVersion := "99.99.99"
				statuses := map[string]*protobufs.PackageStatus{
					packagestate.CollectorPackageName: {
						Name:                 packagestate.CollectorPackageName,
						AgentHasVersion:      version.Version(),
						AgentHasHash:         hash,
						ServerOfferedVersion: newVersion,
						ServerOfferedHash:    newHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installing,
					},
				}
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					Packages:                      statuses,
				}

				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockStateProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, allHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagestate.CollectorPackageName, status.Packages[packagestate.CollectorPackageName].Name)
					assert.Equal(t, version.Version(), status.Packages[packagestate.CollectorPackageName].AgentHasVersion)
					assert.Equal(t, hash, status.Packages[packagestate.CollectorPackageName].AgentHasHash)
					assert.Equal(t, newVersion, status.Packages[packagestate.CollectorPackageName].ServerOfferedVersion)
					assert.Equal(t, newHash, status.Packages[packagestate.CollectorPackageName].ServerOfferedHash)
					assert.Equal(t, "Failed because of collector version mismatch: expected 99.99.99, actual latest", status.Packages[packagestate.CollectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, status.Packages[packagestate.CollectorPackageName].Status)
				})

				c := &Client{
					logger:                zap.NewNop(),
					packagesStateProvider: mockStateProvider,
				}

				c.onConnectHandler(context.Background())
			},
		},
		{
			desc: "Installing with new version match",
			testFunc: func(*testing.T) {
				allHash := []byte("allHash")
				hash := []byte("hash")
				newHash := []byte("newHash")
				oldVersion := "99.99.99"
				newVersion := version.Version()
				statuses := map[string]*protobufs.PackageStatus{
					packagestate.CollectorPackageName: {
						Name:                 packagestate.CollectorPackageName,
						AgentHasVersion:      oldVersion,
						AgentHasHash:         hash,
						ServerOfferedVersion: newVersion,
						ServerOfferedHash:    newHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installing,
					},
				}
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					Packages:                      statuses,
				}

				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockStateProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil)
				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, allHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagestate.CollectorPackageName, status.Packages[packagestate.CollectorPackageName].Name)
					assert.Equal(t, newVersion, status.Packages[packagestate.CollectorPackageName].AgentHasVersion)
					assert.Equal(t, newHash, status.Packages[packagestate.CollectorPackageName].AgentHasHash)
					assert.Equal(t, newVersion, status.Packages[packagestate.CollectorPackageName].ServerOfferedVersion)
					assert.Equal(t, newHash, status.Packages[packagestate.CollectorPackageName].ServerOfferedHash)
					assert.Equal(t, "", status.Packages[packagestate.CollectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_Installed, status.Packages[packagestate.CollectorPackageName].Status)
				})

				c := &Client{
					logger:                zap.NewNop(),
					opampClient:           mockOpAmpClient,
					packagesStateProvider: mockStateProvider,
				}

				c.onConnectHandler(context.Background())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestClient_onConnectFailedHandler(t *testing.T) {
	expectedErr := errors.New("oops")

	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "LastReportedStatus error",
			testFunc: func(*testing.T) {
				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(nil, expectedErr)

				c := &Client{
					logger:                zap.NewNop(),
					packagesStateProvider: mockStateProvider,
				}

				c.onConnectFailedHandler(context.Background(), expectedErr)
			},
		},
		{
			desc: "LastReportedStatus no main package info",
			testFunc: func(*testing.T) {
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: []byte("allHash"),
					Packages:                      make(map[string]*protobufs.PackageStatus),
				}

				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(packageStatuses, nil)

				c := &Client{
					logger:                zap.NewNop(),
					packagesStateProvider: mockStateProvider,
				}

				c.onConnectFailedHandler(context.Background(), expectedErr)
			},
		},
		{
			desc: "Disconnect do not change package status",
			testFunc: func(*testing.T) {
				mockStateProvider := new(mocks.MockPackagesStateProvider)

				c := &Client{
					logger:                zap.NewNop(),
					packagesStateProvider: mockStateProvider,
					disconnecting:         true,
				}

				c.onConnectFailedHandler(context.Background(), expectedErr)
			},
		},
		{
			desc: "Good LastReportedStatus but not installing",
			testFunc: func(*testing.T) {
				allHash := []byte("allHash")
				hash := []byte("hash")
				newHash := []byte("newHash")
				newVersion := "99.99.99"
				errorMessage := "problem"
				statuses := map[string]*protobufs.PackageStatus{
					packagestate.CollectorPackageName: {
						Name:                 packagestate.CollectorPackageName,
						AgentHasVersion:      version.Version(),
						AgentHasHash:         hash,
						ServerOfferedVersion: newVersion,
						ServerOfferedHash:    newHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed,
						ErrorMessage:         errorMessage,
					},
				}
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					Packages:                      statuses,
				}

				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(packageStatuses, nil)

				c := &Client{
					logger:                zap.NewNop(),
					packagesStateProvider: mockStateProvider,
				}

				c.onConnectFailedHandler(context.Background(), expectedErr)
			},
		},
		{
			desc: "Good LastReportedStatus and installing",
			testFunc: func(*testing.T) {
				allHash := []byte("allHash")
				hash := []byte("hash")
				newHash := []byte("newHash")
				newVersion := "99.99.99"
				statuses := map[string]*protobufs.PackageStatus{
					packagestate.CollectorPackageName: {
						Name:                 packagestate.CollectorPackageName,
						AgentHasVersion:      version.Version(),
						AgentHasHash:         hash,
						ServerOfferedVersion: newVersion,
						ServerOfferedHash:    newHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installing,
					},
				}
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					Packages:                      statuses,
				}

				mockOpAmpClient := new(mocks.MockOpAMPClient)
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil)
				mockStateProvider := new(mocks.MockPackagesStateProvider)
				mockStateProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockStateProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, allHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagestate.CollectorPackageName, status.Packages[packagestate.CollectorPackageName].Name)
					assert.Equal(t, version.Version(), status.Packages[packagestate.CollectorPackageName].AgentHasVersion)
					assert.Equal(t, hash, status.Packages[packagestate.CollectorPackageName].AgentHasHash)
					assert.Equal(t, newVersion, status.Packages[packagestate.CollectorPackageName].ServerOfferedVersion)
					assert.Equal(t, newHash, status.Packages[packagestate.CollectorPackageName].ServerOfferedHash)
					assert.Equal(t, fmt.Sprintf("Failed to connect to OpAMP Server: %s", expectedErr), status.Packages[packagestate.CollectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, status.Packages[packagestate.CollectorPackageName].Status)
				})

				c := &Client{
					opampClient:           mockOpAmpClient,
					logger:                zap.NewNop(),
					packagesStateProvider: mockStateProvider,
				}

				c.onConnectFailedHandler(context.Background(), expectedErr)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestClient_onGetEffectiveConfigHandler(t *testing.T) {
	mockManager := mocks.NewMockConfigManager(t)

	c := &Client{
		logger:        zap.NewNop(),
		configManager: mockManager,
	}

	mockManager.On("ComposeEffectiveConfig").Return(&protobufs.EffectiveConfig{}, nil)

	c.onGetEffectiveConfigHandler(context.Background())
	mockManager.AssertExpectations(t)
}

func TestClient_onRemoteConfigHandler(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Config Changes return error, no change",
			testFunc: func(*testing.T) {
				expectedErr := errors.New("oops")
				expectedChanged := false
				mockManager := mocks.NewMockConfigManager(t)
				mockManager.On("ApplyConfigChanges", mock.Anything).Return(expectedChanged, expectedErr)

				remoteConfig := &protobufs.AgentRemoteConfig{
					ConfigHash: []byte("hash"),
				}

				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetRemoteConfigStatus", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.RemoteConfigStatus)

					assert.NotNil(t, status)
					assert.Equal(t, remoteConfig.GetConfigHash(), status.GetLastRemoteConfigHash())
					assert.Equal(t, protobufs.RemoteConfigStatuses_RemoteConfigStatuses_FAILED, status.GetStatus())
					assert.Contains(t, status.GetErrorMessage(), expectedErr.Error())

				})

				c := &Client{
					configManager: mockManager,
					logger:        zap.NewNop(),
					opampClient:   mockOpAmpClient,
				}

				err := c.onRemoteConfigHandler(context.Background(), remoteConfig)
				assert.NoError(t, err)
			},
		},
		{
			desc: "Config Changes occur",
			testFunc: func(*testing.T) {
				mockManager := mocks.NewMockConfigManager(t)
				mockManager.On("ApplyConfigChanges", mock.Anything).Return(true, nil)

				remoteConfig := &protobufs.AgentRemoteConfig{
					ConfigHash: []byte("hash"),
				}

				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("UpdateEffectiveConfig", mock.Anything).Return(nil)
				mockOpAmpClient.On("SetRemoteConfigStatus", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.RemoteConfigStatus)

					assert.NotNil(t, status)
					assert.Equal(t, remoteConfig.GetConfigHash(), status.GetLastRemoteConfigHash())
					assert.Equal(t, protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED, status.GetStatus())
					assert.Equal(t, "", status.GetErrorMessage())

				})

				c := &Client{
					configManager: mockManager,
					logger:        zap.NewNop(),
					opampClient:   mockOpAmpClient,
				}

				err := c.onRemoteConfigHandler(context.Background(), remoteConfig)
				assert.NoError(t, err)
			},
		},
		{
			desc: "No Config Changes occur",
			testFunc: func(*testing.T) {
				mockManager := mocks.NewMockConfigManager(t)
				mockManager.On("ApplyConfigChanges", mock.Anything).Return(false, nil)

				remoteConfig := &protobufs.AgentRemoteConfig{
					ConfigHash: []byte("hash"),
				}

				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetRemoteConfigStatus", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.RemoteConfigStatus)

					assert.NotNil(t, status)
					assert.Equal(t, remoteConfig.GetConfigHash(), status.GetLastRemoteConfigHash())
					assert.Equal(t, protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED, status.GetStatus())
					assert.Equal(t, "", status.GetErrorMessage())

				})

				c := &Client{
					configManager: mockManager,
					logger:        zap.NewNop(),
					opampClient:   mockOpAmpClient,
				}

				err := c.onRemoteConfigHandler(context.Background(), remoteConfig)
				assert.NoError(t, err)
			},
		},
		{
			desc: "SetRemoteConfigStatus errors",
			testFunc: func(*testing.T) {
				expectedErr := errors.New("oops")

				mockManager := mocks.NewMockConfigManager(t)
				mockManager.On("ApplyConfigChanges", mock.Anything).Return(false, nil)

				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetRemoteConfigStatus", mock.Anything).Return(expectedErr)

				c := &Client{
					configManager: mockManager,
					logger:        zap.NewNop(),
					opampClient:   mockOpAmpClient,
				}

				err := c.onRemoteConfigHandler(context.Background(), &protobufs.AgentRemoteConfig{})
				assert.ErrorIs(t, err, expectedErr)
			},
		},
		{
			desc: "UpdateEffectiveConfig errors",
			testFunc: func(*testing.T) {
				expectedErr := errors.New("oops")

				mockManager := mocks.NewMockConfigManager(t)
				mockManager.On("ApplyConfigChanges", mock.Anything).Return(true, nil)

				remoteConfig := &protobufs.AgentRemoteConfig{
					ConfigHash: []byte("hash"),
				}

				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("UpdateEffectiveConfig", mock.Anything).Return(expectedErr)
				mockOpAmpClient.On("SetRemoteConfigStatus", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.RemoteConfigStatus)

					assert.NotNil(t, status)
					assert.Equal(t, remoteConfig.GetConfigHash(), status.GetLastRemoteConfigHash())
					assert.Equal(t, protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED, status.GetStatus())
					assert.Equal(t, "", status.GetErrorMessage())

				})

				c := &Client{
					configManager: mockManager,
					logger:        zap.NewNop(),
					opampClient:   mockOpAmpClient,
				}

				err := c.onRemoteConfigHandler(context.Background(), remoteConfig)
				assert.ErrorIs(t, err, expectedErr)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestClient_onPackagesAvailableHandler(t *testing.T) {
	collectorPackageName := packagestate.CollectorPackageName
	allHash := []byte("totalhash0")
	newAllHash := []byte("totalhash1")
	packageHash := []byte("hash0")
	newPackageHash := []byte("hash1")
	newVersion := "999.999.999"
	expectedErr := errors.New("oops")

	packages := map[string]*protobufs.PackageAvailable{
		collectorPackageName: {
			Version: version.Version(),
			Hash:    packageHash,
			File:    &protobufs.DownloadableFile{},
		},
	}
	packagesAvailable := &protobufs.PackagesAvailable{
		AllPackagesHash: newAllHash,
		Packages:        packages,
	}

	statuses := map[string]*protobufs.PackageStatus{
		collectorPackageName: {
			Name:                 collectorPackageName,
			AgentHasVersion:      version.Version(),
			AgentHasHash:         packageHash,
			ServerOfferedVersion: version.Version(),
			ServerOfferedHash:    packageHash,
			Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
		},
	}
	packageStatuses := &protobufs.PackageStatuses{
		ServerProvidedAllPackagesHash: allHash,
		Packages:                      statuses,
	}

	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Same PackagesAvailable version but bad Last PackagesStatuses",
			testFunc: func(t *testing.T) {
				mockProvider := mocks.NewMockPackagesStateProvider(t)
				mockProvider.On("LastReportedStatuses").Return(nil, expectedErr)
				mockProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil)
				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, packagesAvailable.AllPackagesHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].ServerOfferedVersion)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].ServerOfferedHash)
					assert.Equal(t, "", status.Packages[collectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_Installed, status.Packages[collectorPackageName].Status)
					assert.Equal(t, collectorPackageName, status.Packages[collectorPackageName].Name)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].AgentHasHash)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].AgentHasVersion)
				})

				c := &Client{
					packagesStateProvider: mockProvider,
					opampClient:           mockOpAmpClient,
					logger:                zap.NewNop(),
				}

				err := c.onPackagesAvailableHandler(packagesAvailable)
				assert.NoError(t, err)
			},
		},
		{
			desc: "Same PackagesAvailable version",
			testFunc: func(t *testing.T) {
				mockProvider := mocks.NewMockPackagesStateProvider(t)
				mockProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil)
				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, packagesAvailable.AllPackagesHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].ServerOfferedVersion)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].ServerOfferedHash)
					assert.Equal(t, "", status.Packages[collectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_Installed, status.Packages[collectorPackageName].Status)
					assert.Equal(t, collectorPackageName, status.Packages[collectorPackageName].Name)
					assert.Equal(t, packageStatuses.Packages[collectorPackageName].AgentHasHash, status.Packages[collectorPackageName].AgentHasHash)
					assert.Equal(t, packageStatuses.Packages[collectorPackageName].AgentHasVersion, status.Packages[collectorPackageName].AgentHasVersion)
				})

				c := &Client{
					packagesStateProvider: mockProvider,
					opampClient:           mockOpAmpClient,
					logger:                zap.NewNop(),
				}

				err := c.onPackagesAvailableHandler(packagesAvailable)
				assert.NoError(t, err)
			},
		},
		{
			desc: "Same PackagesAvailable version and non supported package",
			testFunc: func(t *testing.T) {
				badPackageName := "no-support-package"
				packagesNotSupported := map[string]*protobufs.PackageAvailable{
					collectorPackageName: {
						Version: version.Version(),
						Hash:    packageHash,
						File:    &protobufs.DownloadableFile{},
					},
					badPackageName: {
						Version: newVersion,
						Hash:    packageHash,
					},
				}
				packagesAvailableNotSupported := &protobufs.PackagesAvailable{
					AllPackagesHash: newAllHash,
					Packages:        packagesNotSupported,
				}

				mockProvider := mocks.NewMockPackagesStateProvider(t)
				mockProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil)
				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, packagesAvailableNotSupported.AllPackagesHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 2, len(status.Packages))
					assert.Equal(t, packagesAvailableNotSupported.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].ServerOfferedVersion)
					assert.Equal(t, packagesAvailableNotSupported.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].ServerOfferedHash)
					assert.Equal(t, "", status.Packages[collectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_Installed, status.Packages[collectorPackageName].Status)
					assert.Equal(t, collectorPackageName, status.Packages[collectorPackageName].Name)
					assert.Equal(t, packagesAvailableNotSupported.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].AgentHasHash)
					assert.Equal(t, packagesAvailableNotSupported.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].AgentHasVersion)
					assert.Equal(t, packagesAvailableNotSupported.Packages[badPackageName].Version, status.Packages[badPackageName].ServerOfferedVersion)
					assert.Equal(t, packagesAvailableNotSupported.Packages[badPackageName].Hash, status.Packages[badPackageName].ServerOfferedHash)
					assert.Equal(t, "Package not supported", status.Packages[badPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, status.Packages[badPackageName].Status)
					assert.Equal(t, badPackageName, status.Packages[badPackageName].Name)
					assert.Nil(t, status.Packages[badPackageName].AgentHasHash)
					assert.Equal(t, "", status.Packages[badPackageName].AgentHasVersion)
				})

				c := &Client{
					packagesStateProvider: mockProvider,
					opampClient:           mockOpAmpClient,
					logger:                zap.NewNop(),
				}

				err := c.onPackagesAvailableHandler(packagesAvailableNotSupported)
				assert.NoError(t, err)
			},
		},
		{
			desc: "Same PackagesAvailable version but Last PackageStatuses version mismatch",
			testFunc: func(t *testing.T) {
				statusesDiffHash := map[string]*protobufs.PackageStatus{
					collectorPackageName: {
						Name:                 collectorPackageName,
						AgentHasVersion:      newVersion,
						AgentHasHash:         newPackageHash,
						ServerOfferedVersion: newVersion,
						ServerOfferedHash:    newPackageHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
					},
				}
				packageStatusesDiffHash := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: newAllHash,
					Packages:                      statusesDiffHash,
				}

				mockProvider := mocks.NewMockPackagesStateProvider(t)
				mockProvider.On("LastReportedStatuses").Return(packageStatusesDiffHash, nil)
				mockProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil)
				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, packagesAvailable.AllPackagesHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].ServerOfferedVersion)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].ServerOfferedHash)
					assert.Equal(t, "", status.Packages[collectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_Installed, status.Packages[collectorPackageName].Status)
					assert.Equal(t, collectorPackageName, status.Packages[collectorPackageName].Name)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].AgentHasHash)
					assert.NotEqual(t, statusesDiffHash[collectorPackageName].AgentHasHash, status.Packages[collectorPackageName].AgentHasHash)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].AgentHasVersion)
					assert.NotEqual(t, statusesDiffHash[collectorPackageName].AgentHasVersion, status.Packages[collectorPackageName].AgentHasVersion)
				})

				c := &Client{
					packagesStateProvider: mockProvider,
					opampClient:           mockOpAmpClient,
					logger:                zap.NewNop(),
				}

				err := c.onPackagesAvailableHandler(packagesAvailable)
				assert.NoError(t, err)
			},
		},
		// The version of this test where the update goes well can't exist because
		// it would kill the collector. StartAndMonitorUpdater will always return an error
		// if it does return.
		{
			desc: "New PackagesAvailable version with good file but bad update",
			testFunc: func(t *testing.T) {
				packagesNew := map[string]*protobufs.PackageAvailable{
					collectorPackageName: {
						Version: newVersion,
						Hash:    newPackageHash,
						File:    &protobufs.DownloadableFile{},
					},
				}
				packagesAvailableNew := &protobufs.PackagesAvailable{
					AllPackagesHash: newAllHash,
					Packages:        packagesNew,
				}
				savedStatuses := map[string]*protobufs.PackageStatus{
					collectorPackageName: {
						Name:                 collectorPackageName,
						AgentHasVersion:      version.Version(),
						AgentHasHash:         packageHash,
						ServerOfferedVersion: newVersion,
						ServerOfferedHash:    newPackageHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installing,
					},
				}
				savedPackageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: newAllHash,
					Packages:                      savedStatuses,
				}
				wg := sync.WaitGroup{}
				wg.Add(2)
				mockUpdaterManager := mocks.NewMockUpdaterManager(t)
				mockUpdaterManager.On("StartAndMonitorUpdater").Return(expectedErr)
				mockProvider := mocks.NewMockPackagesStateProvider(t)
				mockProvider.On("LastReportedStatuses").Return(packageStatuses, nil).Once()
				mockProvider.On("LastReportedStatuses").Return(savedPackageStatuses, nil)
				mockProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil)
				mockFileManager := mocks.NewMockDownloadableFileManager(t)
				mockFileManager.On("FetchAndExtractArchive", mock.Anything).Return(nil).Run(func(_ mock.Arguments) {
					wg.Done()
				})
				mockFileManager.On("CleanupArtifacts").Return().Times(1)

				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil).Once().Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, packagesAvailableNew.AllPackagesHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagesAvailableNew.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].ServerOfferedVersion)
					assert.Equal(t, packagesAvailableNew.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].ServerOfferedHash)
					assert.Equal(t, "", status.Packages[collectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_Installing, status.Packages[collectorPackageName].Status)
					assert.Equal(t, collectorPackageName, status.Packages[collectorPackageName].Name)
					assert.Equal(t, packageStatuses.Packages[collectorPackageName].AgentHasHash, status.Packages[collectorPackageName].AgentHasHash)
					assert.Equal(t, packageStatuses.Packages[collectorPackageName].AgentHasVersion, status.Packages[collectorPackageName].AgentHasVersion)
				})
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, packagesAvailableNew.AllPackagesHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagesAvailableNew.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].ServerOfferedVersion)
					assert.Equal(t, packagesAvailableNew.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].ServerOfferedHash)
					assert.Equal(t, "Failed to run the latest Updater: oops", status.Packages[collectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, status.Packages[collectorPackageName].Status)
					assert.Equal(t, collectorPackageName, status.Packages[collectorPackageName].Name)
					assert.Equal(t, packageStatuses.Packages[collectorPackageName].AgentHasHash, status.Packages[collectorPackageName].AgentHasHash)
					assert.Equal(t, packageStatuses.Packages[collectorPackageName].AgentHasVersion, status.Packages[collectorPackageName].AgentHasVersion)
					wg.Done()
				})

				c := &Client{
					packagesStateProvider:   mockProvider,
					downloadableFileManager: mockFileManager,
					opampClient:             mockOpAmpClient,
					logger:                  zap.NewNop(),
					updaterManager:          mockUpdaterManager,
				}

				err := c.onPackagesAvailableHandler(packagesAvailableNew)
				assert.NoError(t, err)
				wg.Wait()
				assert.False(t, c.safeGetUpdatingPackage())
			},
		},
		{
			desc: "New PackagesAvailable version while already installing",
			testFunc: func(t *testing.T) {
				packagesNew := map[string]*protobufs.PackageAvailable{
					collectorPackageName: {
						Version: newVersion,
						Hash:    newPackageHash,
						File:    &protobufs.DownloadableFile{},
					},
				}
				packagesAvailableNew := &protobufs.PackagesAvailable{
					AllPackagesHash: newAllHash,
					Packages:        packagesNew,
				}

				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "Already installing new packages", status.ErrorMessage)
					assert.Equal(t, packagesAvailableNew.AllPackagesHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 0, len(status.Packages))
				})

				c := &Client{
					opampClient:     mockOpAmpClient,
					logger:          zap.NewNop(),
					updatingPackage: true,
				}

				err := c.onPackagesAvailableHandler(packagesAvailableNew)
				assert.ErrorContains(t, err, "failed because already installing packages")
			},
		},
		{
			desc: "New PackagesAvailable version with no DownloadableFile",
			testFunc: func(t *testing.T) {
				packagesNoFile := map[string]*protobufs.PackageAvailable{
					collectorPackageName: {
						Version: newVersion,
						Hash:    newPackageHash,
					},
				}
				packagesAvailableNoFile := &protobufs.PackagesAvailable{
					AllPackagesHash: newAllHash,
					Packages:        packagesNoFile,
				}

				mockProvider := mocks.NewMockPackagesStateProvider(t)
				mockProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil)
				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, packagesAvailableNoFile.AllPackagesHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagesAvailableNoFile.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].ServerOfferedVersion)
					assert.Equal(t, packagesAvailableNoFile.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].ServerOfferedHash)
					assert.Equal(t, "Packaged update failed because no downloadable file detected", status.Packages[collectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, status.Packages[collectorPackageName].Status)
					assert.Equal(t, collectorPackageName, status.Packages[collectorPackageName].Name)
					assert.Equal(t, packageStatuses.Packages[collectorPackageName].AgentHasHash, status.Packages[collectorPackageName].AgentHasHash)
					assert.Equal(t, packageStatuses.Packages[collectorPackageName].AgentHasVersion, status.Packages[collectorPackageName].AgentHasVersion)
				})

				c := &Client{
					packagesStateProvider: mockProvider,
					opampClient:           mockOpAmpClient,
					logger:                zap.NewNop(),
				}

				err := c.onPackagesAvailableHandler(packagesAvailableNoFile)
				assert.NoError(t, err)
			},
		},
		{
			desc: "New PackagesAvailable version with bad DownloadableFile",
			testFunc: func(t *testing.T) {
				packagesNew := map[string]*protobufs.PackageAvailable{
					collectorPackageName: {
						Version: newVersion,
						Hash:    newPackageHash,
						File:    &protobufs.DownloadableFile{},
					},
				}
				packagesAvailableNew := &protobufs.PackagesAvailable{
					AllPackagesHash: newAllHash,
					Packages:        packagesNew,
				}

				mockFileManager := mocks.NewMockDownloadableFileManager(t)
				mockFileManager.On("FetchAndExtractArchive", mock.Anything).Return(expectedErr)
				mockFileManager.On("CleanupArtifacts").Return().Times(1)
				mockProvider := mocks.NewMockPackagesStateProvider(t)
				mockProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil)
				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				// This is for the initial status that is sent in the main function.
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil).Once().Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, packagesAvailableNew.AllPackagesHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagesAvailableNew.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].ServerOfferedVersion)
					assert.Equal(t, packagesAvailableNew.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].ServerOfferedHash)
					assert.Equal(t, "", status.Packages[collectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_Installing, status.Packages[collectorPackageName].Status)
					assert.Equal(t, collectorPackageName, status.Packages[collectorPackageName].Name)
					assert.Equal(t, packageStatuses.Packages[collectorPackageName].AgentHasHash, status.Packages[collectorPackageName].AgentHasHash)
					assert.Equal(t, packageStatuses.Packages[collectorPackageName].AgentHasVersion, status.Packages[collectorPackageName].AgentHasVersion)
				})
				// This will be called within the goroutine that is spun up from the main function.
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)
					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, packagesAvailableNew.AllPackagesHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagesAvailableNew.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].ServerOfferedVersion)
					assert.Equal(t, packagesAvailableNew.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].ServerOfferedHash)
					assert.Equal(t, "Failed to download and verify downloadable file: oops", status.Packages[collectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, status.Packages[collectorPackageName].Status)
					assert.Equal(t, collectorPackageName, status.Packages[collectorPackageName].Name)
					assert.Equal(t, packageStatuses.Packages[collectorPackageName].AgentHasHash, status.Packages[collectorPackageName].AgentHasHash)
					assert.Equal(t, packageStatuses.Packages[collectorPackageName].AgentHasVersion, status.Packages[collectorPackageName].AgentHasVersion)
				})

				c := &Client{
					packagesStateProvider:   mockProvider,
					downloadableFileManager: mockFileManager,
					opampClient:             mockOpAmpClient,
					logger:                  zap.NewNop(),
				}

				err := c.onPackagesAvailableHandler(packagesAvailableNew)
				assert.NoError(t, err)
				assert.Eventually(t, func() bool { return c.safeGetUpdatingPackage() == false }, 10*time.Second, 10*time.Millisecond)
			},
		},
		{
			desc: "Same PackagesAvailable version but bad set last PackageStatuses",
			testFunc: func(t *testing.T) {
				mockProvider := mocks.NewMockPackagesStateProvider(t)
				mockProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockProvider.On("SetLastReportedStatuses", mock.Anything).Return(expectedErr).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, packagesAvailable.AllPackagesHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].ServerOfferedVersion)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].ServerOfferedHash)
					assert.Equal(t, "", status.Packages[collectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_Installed, status.Packages[collectorPackageName].Status)
					assert.Equal(t, collectorPackageName, status.Packages[collectorPackageName].Name)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].AgentHasHash)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].AgentHasVersion)
				})
				mockOpAmpClient := mocks.NewMockOpAMPClient(t)

				c := &Client{
					packagesStateProvider: mockProvider,
					opampClient:           mockOpAmpClient,
					logger:                zap.NewNop(),
				}

				err := c.onPackagesAvailableHandler(packagesAvailable)
				assert.ErrorIs(t, err, expectedErr)
			},
		},
		{
			desc: "Same PackagesAvailable version but bad SEND PackageStatuses",
			testFunc: func(t *testing.T) {
				mockProvider := mocks.NewMockPackagesStateProvider(t)
				mockProvider.On("LastReportedStatuses").Return(packageStatuses, nil)
				mockProvider.On("SetLastReportedStatuses", mock.Anything).Return(nil)
				mockOpAmpClient := mocks.NewMockOpAMPClient(t)
				mockOpAmpClient.On("SetPackageStatuses", mock.Anything).Return(expectedErr).Run(func(args mock.Arguments) {
					status := args.Get(0).(*protobufs.PackageStatuses)

					assert.NotNil(t, status)
					assert.Equal(t, "", status.ErrorMessage)
					assert.Equal(t, packagesAvailable.AllPackagesHash, status.ServerProvidedAllPackagesHash)
					assert.Equal(t, 1, len(status.Packages))
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].ServerOfferedVersion)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].ServerOfferedHash)
					assert.Equal(t, "", status.Packages[collectorPackageName].ErrorMessage)
					assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_Installed, status.Packages[collectorPackageName].Status)
					assert.Equal(t, collectorPackageName, status.Packages[collectorPackageName].Name)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Hash, status.Packages[collectorPackageName].AgentHasHash)
					assert.Equal(t, packagesAvailable.Packages[collectorPackageName].Version, status.Packages[collectorPackageName].AgentHasVersion)
				})

				c := &Client{
					packagesStateProvider: mockProvider,
					opampClient:           mockOpAmpClient,
					logger:                zap.NewNop(),
				}

				err := c.onPackagesAvailableHandler(packagesAvailable)
				assert.ErrorIs(t, err, expectedErr)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
