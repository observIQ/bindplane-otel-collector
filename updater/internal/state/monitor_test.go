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

package state

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/observiq/bindplane-otel-collector/packagestate/mocks"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
)

func TestCollectorMonitorSetState(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Package not in current status",
			testFunc: func(*testing.T) {
				mockStateManger := mocks.NewMockStateManager(t)

				collectorMonitor := &CollectorMonitor{
					stateManager: mockStateManger,
					currentStatus: &protobufs.PackageStatuses{
						Packages: make(map[string]*protobufs.PackageStatus),
					},
				}

				err := collectorMonitor.SetState("my_package", protobufs.PackageStatusEnum_PackageStatusEnum_Installed, nil)
				assert.Error(t, err)
			},
		},
		{
			desc: "Sets Status no error",
			testFunc: func(*testing.T) {
				pgkName := "my_package"
				expectedStatus := &protobufs.PackageStatuses{
					Packages: map[string]*protobufs.PackageStatus{
						pgkName: {
							Name:                 pgkName,
							AgentHasVersion:      "1.0",
							AgentHasHash:         []byte("hash1"),
							ServerOfferedVersion: "1.2",
							ServerOfferedHash:    []byte("hash2"),
							Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
						},
					},
				}

				mockStateManger := mocks.NewMockStateManager(t)
				mockStateManger.On("SaveStatuses", expectedStatus).Return(nil)

				collectorMonitor := &CollectorMonitor{
					stateManager: mockStateManger,
					currentStatus: &protobufs.PackageStatuses{
						Packages: map[string]*protobufs.PackageStatus{
							pgkName: {
								Name:                 pgkName,
								AgentHasVersion:      "1.0",
								AgentHasHash:         []byte("hash1"),
								ServerOfferedVersion: "1.2",
								ServerOfferedHash:    []byte("hash2"),
								Status:               protobufs.PackageStatusEnum_PackageStatusEnum_InstallPending,
							},
						},
					},
				}

				err := collectorMonitor.SetState("my_package", protobufs.PackageStatusEnum_PackageStatusEnum_Installed, nil)
				assert.NoError(t, err)
				assert.Equal(t, expectedStatus, collectorMonitor.currentStatus)
			},
		},
		{
			desc: "Sets Status w/error",
			testFunc: func(*testing.T) {
				pgkName := "my_package"
				statusErr := errors.New("some error")

				expectedStatus := &protobufs.PackageStatuses{
					Packages: map[string]*protobufs.PackageStatus{
						pgkName: {
							Name:                 pgkName,
							AgentHasVersion:      "1.0",
							AgentHasHash:         []byte("hash1"),
							ServerOfferedVersion: "1.2",
							ServerOfferedHash:    []byte("hash2"),
							Status:               protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed,
							ErrorMessage:         statusErr.Error(),
						},
					},
				}

				mockStateManger := mocks.NewMockStateManager(t)
				mockStateManger.On("SaveStatuses", expectedStatus).Return(nil)

				collectorMonitor := &CollectorMonitor{
					stateManager: mockStateManger,
					currentStatus: &protobufs.PackageStatuses{
						Packages: map[string]*protobufs.PackageStatus{
							pgkName: {
								Name:                 pgkName,
								AgentHasVersion:      "1.0",
								AgentHasHash:         []byte("hash1"),
								ServerOfferedVersion: "1.2",
								ServerOfferedHash:    []byte("hash2"),
								Status:               protobufs.PackageStatusEnum_PackageStatusEnum_InstallPending,
							},
						},
					},
				}

				err := collectorMonitor.SetState("my_package", protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed, statusErr)
				assert.NoError(t, err)
				assert.Equal(t, expectedStatus, collectorMonitor.currentStatus)
			},
		},
		{
			desc: "StateManager fails to save",
			testFunc: func(*testing.T) {
				pgkName := "my_package"
				expectedErr := errors.New("bad")
				expectedStatus := &protobufs.PackageStatuses{
					Packages: map[string]*protobufs.PackageStatus{
						pgkName: {
							Name:                 pgkName,
							AgentHasVersion:      "1.0",
							AgentHasHash:         []byte("hash1"),
							ServerOfferedVersion: "1.2",
							ServerOfferedHash:    []byte("hash2"),
							Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
						},
					},
				}

				mockStateManger := mocks.NewMockStateManager(t)
				mockStateManger.On("SaveStatuses", expectedStatus).Return(expectedErr)

				collectorMonitor := &CollectorMonitor{
					stateManager: mockStateManger,
					currentStatus: &protobufs.PackageStatuses{
						Packages: map[string]*protobufs.PackageStatus{
							pgkName: {
								Name:                 pgkName,
								AgentHasVersion:      "1.0",
								AgentHasHash:         []byte("hash1"),
								ServerOfferedVersion: "1.2",
								ServerOfferedHash:    []byte("hash2"),
								Status:               protobufs.PackageStatusEnum_PackageStatusEnum_InstallPending,
							},
						},
					},
				}

				err := collectorMonitor.SetState("my_package", protobufs.PackageStatusEnum_PackageStatusEnum_Installed, nil)
				assert.ErrorIs(t, err, expectedErr)
				assert.Equal(t, expectedStatus, collectorMonitor.currentStatus)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestCollectorMonitorMonitorForSuccess(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Context is canceled",
			testFunc: func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				mockStateManger := mocks.NewMockStateManager(t)
				collectorMonitor := &CollectorMonitor{
					stateManager: mockStateManger,
				}

				err := collectorMonitor.MonitorForSuccess(ctx, "my_package")
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
		{
			desc: "Package Status Indicates Failed Install",
			testFunc: func(t *testing.T) {
				pgkName := "my_package"
				returnedStatus := &protobufs.PackageStatuses{
					Packages: map[string]*protobufs.PackageStatus{
						pgkName: {
							Name:   pgkName,
							Status: protobufs.PackageStatusEnum_PackageStatusEnum_InstallFailed,
						},
					},
				}

				mockStateManger := mocks.NewMockStateManager(t)
				mockStateManger.On("LoadStatuses").Return(returnedStatus, nil)

				collectorMonitor := &CollectorMonitor{
					stateManager: mockStateManger,
				}

				err := collectorMonitor.MonitorForSuccess(context.Background(), pgkName)
				assert.ErrorIs(t, err, ErrFailedStatus)
			},
		},
		{
			desc: "Package Status Indicates Successful install",
			testFunc: func(t *testing.T) {
				pgkName := "my_package"
				returnedStatus := &protobufs.PackageStatuses{
					Packages: map[string]*protobufs.PackageStatus{
						pgkName: {
							Name:   pgkName,
							Status: protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
						},
					},
				}

				mockStateManger := mocks.NewMockStateManager(t)
				mockStateManger.On("LoadStatuses").Return(returnedStatus, nil)

				collectorMonitor := &CollectorMonitor{
					stateManager: mockStateManger,
				}

				err := collectorMonitor.MonitorForSuccess(context.Background(), pgkName)
				assert.NoError(t, err)
			},
		},
		{
			desc: "File does not exist at first then is successful",
			testFunc: func(t *testing.T) {
				pgkName := "my_package"
				returnedStatus := &protobufs.PackageStatuses{
					Packages: map[string]*protobufs.PackageStatus{
						pgkName: {
							Name:   pgkName,
							Status: protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
						},
					},
				}

				mockStateManger := mocks.NewMockStateManager(t)
				mockStateManger.On("LoadStatuses").Once().Return(nil, os.ErrNotExist)
				mockStateManger.On("LoadStatuses").Return(returnedStatus, nil)

				collectorMonitor := &CollectorMonitor{
					stateManager: mockStateManger,
				}

				err := collectorMonitor.MonitorForSuccess(context.Background(), pgkName)
				assert.NoError(t, err)
			},
		},
		{
			desc: "Error reading file at first first then is successful",
			testFunc: func(t *testing.T) {
				pgkName := "my_package"
				returnedStatus := &protobufs.PackageStatuses{
					Packages: map[string]*protobufs.PackageStatus{
						pgkName: {
							Name:   pgkName,
							Status: protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
						},
					},
				}

				mockStateManger := mocks.NewMockStateManager(t)
				mockStateManger.On("LoadStatuses").Once().Return(nil, errors.New("bad"))
				mockStateManger.On("LoadStatuses").Return(returnedStatus, nil)

				collectorMonitor := &CollectorMonitor{
					stateManager: mockStateManger,
				}

				err := collectorMonitor.MonitorForSuccess(context.Background(), pgkName)
				assert.NoError(t, err)
			},
		},
		{
			desc: "Package is not present at first then is successful",
			testFunc: func(t *testing.T) {
				pgkName := "my_package"
				firstStatus := &protobufs.PackageStatuses{
					Packages: map[string]*protobufs.PackageStatus{},
				}
				secondStatus := &protobufs.PackageStatuses{
					Packages: map[string]*protobufs.PackageStatus{
						pgkName: {
							Name:   pgkName,
							Status: protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
						},
					},
				}

				mockStateManger := mocks.NewMockStateManager(t)
				mockStateManger.On("LoadStatuses").Once().Return(firstStatus, nil)
				mockStateManger.On("LoadStatuses").Return(secondStatus, nil)

				collectorMonitor := &CollectorMonitor{
					stateManager: mockStateManger,
				}

				err := collectorMonitor.MonitorForSuccess(context.Background(), pgkName)
				assert.NoError(t, err)
			},
		},
		{
			desc: "Package is still marked as Installing at first then is successful",
			testFunc: func(t *testing.T) {
				pgkName := "my_package"
				firstStatus := &protobufs.PackageStatuses{
					Packages: map[string]*protobufs.PackageStatus{
						pgkName: {
							Name:   pgkName,
							Status: protobufs.PackageStatusEnum_PackageStatusEnum_InstallPending,
						},
					},
				}
				secondStatus := &protobufs.PackageStatuses{
					Packages: map[string]*protobufs.PackageStatus{
						pgkName: {
							Name:   pgkName,
							Status: protobufs.PackageStatusEnum_PackageStatusEnum_Installed,
						},
					},
				}

				mockStateManger := mocks.NewMockStateManager(t)
				mockStateManger.On("LoadStatuses").Once().Return(firstStatus, nil)
				mockStateManger.On("LoadStatuses").Return(secondStatus, nil)

				collectorMonitor := &CollectorMonitor{
					stateManager: mockStateManger,
				}

				err := collectorMonitor.MonitorForSuccess(context.Background(), pgkName)
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
