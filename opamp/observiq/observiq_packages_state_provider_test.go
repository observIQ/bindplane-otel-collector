// Copyright  observIQ, Inc.
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
	"io"
	"os"
	"testing"

	"github.com/observiq/bindplane-otel-collector/packagestate"
	"github.com/observiq/bindplane-otel-collector/packagestate/mocks"
	"github.com/observiq/bindplane-otel-collector/version"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewPackagesStateProvider(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "New PackagesStateProvider",
			testFunc: func(t *testing.T) {
				logger := zap.NewNop()
				actual := newPackagesStateProvider(logger, "test.json")

				packagesStateProvider, ok := actual.(*packagesStateProvider)
				require.True(t, ok)

				assert.Equal(t, logger, packagesStateProvider.logger)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestAllPackagesHash(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Not Implemented",
			testFunc: func(t *testing.T) {
				logger := zap.NewNop()
				p := &packagesStateProvider{
					logger: logger,
				}

				actual, err := p.AllPackagesHash()

				assert.Nil(t, actual)
				assert.ErrorContains(t, err, "method not implemented")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestSetAllPackagesHash(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Not Implemented",
			testFunc: func(t *testing.T) {
				logger := zap.NewNop()
				p := &packagesStateProvider{
					logger: logger,
				}

				err := p.SetAllPackagesHash([]byte("hash"))

				assert.ErrorContains(t, err, "method not implemented")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestPackages(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Not Implemented",
			testFunc: func(t *testing.T) {
				logger := zap.NewNop()
				p := &packagesStateProvider{
					logger: logger,
				}

				actual, err := p.Packages()

				assert.Nil(t, actual)
				assert.ErrorContains(t, err, "method not implemented")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestPackageState(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Not Implemented",
			testFunc: func(t *testing.T) {
				logger := zap.NewNop()
				p := &packagesStateProvider{
					logger: logger,
				}

				actual, err := p.PackageState("name")

				assert.Equal(t, types.PackageState{}, actual)
				assert.ErrorContains(t, err, "method not implemented")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestSetPackageState(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Not Implemented",
			testFunc: func(t *testing.T) {
				logger := zap.NewNop()
				p := &packagesStateProvider{
					logger: logger,
				}

				err := p.SetPackageState("name", types.PackageState{})

				assert.ErrorContains(t, err, "method not implemented")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestCreatePackage(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Not Implemented",
			testFunc: func(t *testing.T) {
				logger := zap.NewNop()
				p := &packagesStateProvider{
					logger: logger,
				}

				err := p.CreatePackage("name", protobufs.PackageType_PackageType_TopLevel)

				assert.ErrorContains(t, err, "method not implemented")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestFileContentHash(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Not Implemented",
			testFunc: func(t *testing.T) {
				logger := zap.NewNop()
				p := &packagesStateProvider{
					logger: logger,
				}

				actual, err := p.FileContentHash("name")

				assert.Nil(t, actual)
				assert.ErrorContains(t, err, "method not implemented")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestUpdateContent(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Not Implemented",
			testFunc: func(t *testing.T) {
				logger := zap.NewNop()
				p := &packagesStateProvider{
					logger: logger,
				}
				var r io.Reader

				err := p.UpdateContent(context.TODO(), "name", r, []byte("hash"), []byte("signature"))

				assert.ErrorContains(t, err, "method not implemented")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestDeletePackage(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Not Implemented",
			testFunc: func(t *testing.T) {
				logger := zap.NewNop()
				p := &packagesStateProvider{
					logger: logger,
				}

				err := p.DeletePackage("name")

				assert.ErrorContains(t, err, "method not implemented")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestLastReportedStatuses(t *testing.T) {
	pkgName := packagestate.CollectorPackageName
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "PackageStateManager returns error for missing file",
			testFunc: func(t *testing.T) {
				mockManager := mocks.NewMockStateManager(t)
				mockManager.On("LoadStatuses").Return(nil, os.ErrNotExist)

				p := &packagesStateProvider{
					packageStateManager: mockManager,
					logger:              zap.NewNop(),
				}

				actual, err := p.LastReportedStatuses()

				assert.NoError(t, err)
				assert.Nil(t, actual.ServerProvidedAllPackagesHash)
				assert.Equal(t, "", actual.ErrorMessage)
				assert.Equal(t, 1, len(actual.Packages))
				assert.Equal(t, pkgName, actual.Packages[pkgName].GetName())
				assert.Equal(t, version.Version(), actual.Packages[pkgName].GetAgentHasVersion())
				assert.Nil(t, actual.Packages[pkgName].GetAgentHasHash())
				assert.Equal(t, "", actual.Packages[pkgName].GetServerOfferedVersion())
				assert.Nil(t, actual.Packages[pkgName].GetServerOfferedHash())
				assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_Installed, actual.Packages[pkgName].GetStatus())
				assert.Equal(t, "", actual.Packages[pkgName].GetErrorMessage())
			},
		},
		{
			desc: "Load Error",
			testFunc: func(t *testing.T) {
				expectedErr := errors.New("bad")
				mockManager := mocks.NewMockStateManager(t)
				mockManager.On("LoadStatuses").Return(nil, expectedErr)

				p := &packagesStateProvider{
					packageStateManager: mockManager,
					logger:              zap.NewNop(),
				}

				actual, err := p.LastReportedStatuses()

				assert.ErrorIs(t, err, expectedErr)
				assert.Nil(t, actual.ServerProvidedAllPackagesHash)
				assert.Equal(t, "", actual.ErrorMessage)
				assert.Equal(t, 1, len(actual.Packages))
				assert.Equal(t, pkgName, actual.Packages[pkgName].GetName())
				assert.Equal(t, version.Version(), actual.Packages[pkgName].GetAgentHasVersion())
				assert.Nil(t, actual.Packages[pkgName].GetAgentHasHash())
				assert.Equal(t, "", actual.Packages[pkgName].GetServerOfferedVersion())
				assert.Nil(t, actual.Packages[pkgName].GetServerOfferedHash())
				assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_Installed, actual.Packages[pkgName].GetStatus())
				assert.Equal(t, "", actual.Packages[pkgName].GetErrorMessage())
			},
		},
		{
			desc: "Successful file read",
			testFunc: func(t *testing.T) {
				expected := &protobufs.PackageStatuses{
					Packages: map[string]*protobufs.PackageStatus{
						"package": {
							Name:                 "package",
							AgentHasVersion:      "1.0",
							AgentHasHash:         []byte("hash1"),
							ServerOfferedVersion: "2.0",
							ServerOfferedHash:    []byte("hash2"),
							Status:               protobufs.PackageStatusEnum_PackageStatusEnum_InstallPending,
							ErrorMessage:         "bad",
						},
					},
					ServerProvidedAllPackagesHash: []byte("hash"),
					ErrorMessage:                  "whoops",
				}

				mockManager := mocks.NewMockStateManager(t)
				mockManager.On("LoadStatuses").Return(expected, nil)

				p := &packagesStateProvider{
					packageStateManager: mockManager,
					logger:              zap.NewNop(),
				}

				actual, err := p.LastReportedStatuses()

				assert.NoError(t, err)
				assert.Equal(t, expected, actual)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestSetLastReportedStatuses(t *testing.T) {
	pkgName := "package"
	agentVersion := "1.0"
	agentHash := []byte("hash1")
	serverVersion := "2.0"
	serverHash := []byte("hash2")
	errMsg := "bad"
	allHash := []byte("hash")
	allErrMsg := "whoops"
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "PackageStateManager Returns error",
			testFunc: func(t *testing.T) {
				expectedErr := errors.New("bad")

				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					ErrorMessage:                  allErrMsg,
					Packages: map[string]*protobufs.PackageStatus{
						pkgName: {
							Name:                 pkgName,
							AgentHasVersion:      agentVersion,
							AgentHasHash:         agentHash,
							ServerOfferedVersion: serverVersion,
							ServerOfferedHash:    serverHash,
							Status:               protobufs.PackageStatusEnum_PackageStatusEnum_InstallPending,
							ErrorMessage:         errMsg,
						},
					},
				}

				mockManager := mocks.NewMockStateManager(t)
				mockManager.On("SaveStatuses", packageStatuses).Return(expectedErr)

				p := &packagesStateProvider{
					packageStateManager: mockManager,
					logger:              zap.NewNop(),
				}

				err := p.SetLastReportedStatuses(packageStatuses)
				assert.ErrorIs(t, err, expectedErr)
			},
		},
		{
			desc: "PackageStateManager No error",
			testFunc: func(t *testing.T) {
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					ErrorMessage:                  allErrMsg,
					Packages: map[string]*protobufs.PackageStatus{
						pkgName: {
							Name:                 pkgName,
							AgentHasVersion:      agentVersion,
							AgentHasHash:         agentHash,
							ServerOfferedVersion: serverVersion,
							ServerOfferedHash:    serverHash,
							Status:               protobufs.PackageStatusEnum_PackageStatusEnum_InstallPending,
							ErrorMessage:         errMsg,
						},
					},
				}

				mockManager := mocks.NewMockStateManager(t)
				mockManager.On("SaveStatuses", packageStatuses).Return(nil)

				p := &packagesStateProvider{
					packageStateManager: mockManager,
					logger:              zap.NewNop(),
				}

				err := p.SetLastReportedStatuses(packageStatuses)
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
