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

package packagestate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewPackagesStateManager(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "New PackagesStateManager",
			testFunc: func(t *testing.T) {
				jsonPath := "test.json"
				logger := zap.NewNop()
				actual := NewFileStateManager(logger, jsonPath)

				var expected StateManager = &FileStateManager{
					jsonPath: jsonPath,
					logger:   logger,
				}

				assert.Equal(t, expected, actual)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestLoadStatuses(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "File doesn't exist",
			testFunc: func(t *testing.T) {
				noExistJSON := "garbage.json"
				logger := zap.NewNop()
				p := &FileStateManager{
					logger:   logger,
					jsonPath: noExistJSON,
				}

				actual, err := p.LoadStatuses()

				assert.ErrorIs(t, err, os.ErrNotExist)
				assert.Nil(t, actual)
			},
		},
		{
			desc: "Bad json file",
			testFunc: func(t *testing.T) {
				badJSON := "testdata/package_statuses_bad.json"
				logger := zap.NewNop()
				p := &FileStateManager{
					logger:   logger,
					jsonPath: badJSON,
				}

				actual, err := p.LoadStatuses()

				assert.ErrorContains(t, err, "failed to unmarshal package statuses:")
				assert.Nil(t, actual)
			},
		},
		{
			desc: "Good json file",
			testFunc: func(t *testing.T) {
				goodJSON := "testdata/package_statuses_good.json"
				pkgName := "package"
				agentVersion := "1.0"
				agentHash := []byte("hash1")
				serverVersion := "2.0"
				serverHash := []byte("hash2")
				errMsg := "bad"
				allHash := []byte("hash")
				allErrMsg := "whoops"
				logger := zap.NewNop()
				p := &FileStateManager{
					logger:   logger,
					jsonPath: goodJSON,
				}

				actual, err := p.LoadStatuses()

				assert.NoError(t, err)
				assert.Equal(t, allHash, actual.ServerProvidedAllPackagesHash)
				assert.Equal(t, allErrMsg, actual.ErrorMessage)
				assert.Equal(t, 1, len(actual.Packages))
				assert.Equal(t, pkgName, actual.Packages[pkgName].GetName())
				assert.Equal(t, agentVersion, actual.Packages[pkgName].GetAgentHasVersion())
				assert.Equal(t, agentHash, actual.Packages[pkgName].GetAgentHasHash())
				assert.Equal(t, serverVersion, actual.Packages[pkgName].GetServerOfferedVersion())
				assert.Equal(t, serverHash, actual.Packages[pkgName].GetServerOfferedHash())
				assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallPending, actual.Packages[pkgName].GetStatus())
				assert.Equal(t, errMsg, actual.Packages[pkgName].GetErrorMessage())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestSaveStatuses(t *testing.T) {
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
			desc: "New file",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				testJSON := filepath.Join(tmpDir, "test.json")
				logger := zap.NewNop()
				p := &FileStateManager{
					logger:   logger,
					jsonPath: testJSON,
				}

				packages := map[string]*protobufs.PackageStatus{
					pkgName: {
						Name:                 pkgName,
						AgentHasVersion:      agentVersion,
						AgentHasHash:         agentHash,
						ServerOfferedVersion: serverVersion,
						ServerOfferedHash:    serverHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_InstallPending,
						ErrorMessage:         errMsg,
					},
				}
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					ErrorMessage:                  allErrMsg,
					Packages:                      packages,
				}

				err := p.SaveStatuses(packageStatuses)
				assert.NoError(t, err)

				bytes, err := os.ReadFile(testJSON)
				assert.NoError(t, err)
				var fileStates packageStates
				err = json.Unmarshal(bytes, &fileStates)
				assert.NoError(t, err)
				assert.Equal(t, allHash, fileStates.AllPackagesHash)
				assert.Equal(t, allErrMsg, fileStates.AllErrorMessage)
				assert.Equal(t, 1, len(fileStates.PackageStates))
				assert.Equal(t, pkgName, fileStates.PackageStates[pkgName].Name)
				assert.Equal(t, agentVersion, fileStates.PackageStates[pkgName].AgentVersion)
				assert.Equal(t, agentHash, fileStates.PackageStates[pkgName].AgentHash)
				assert.Equal(t, serverVersion, fileStates.PackageStates[pkgName].ServerVersion)
				assert.Equal(t, serverHash, fileStates.PackageStates[pkgName].ServerHash)
				assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallPending, fileStates.PackageStates[pkgName].Status)
				assert.Equal(t, errMsg, fileStates.PackageStates[pkgName].ErrorMessage)
			},
		},
		{
			desc: "Existing file",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				testJSON := filepath.Join(tmpDir, "test.json")
				os.WriteFile(testJSON, nil, 0600)

				logger := zap.NewNop()
				p := &FileStateManager{
					logger:   logger,
					jsonPath: testJSON,
				}

				packages := map[string]*protobufs.PackageStatus{
					pkgName: {
						Name:                 pkgName,
						AgentHasVersion:      agentVersion,
						AgentHasHash:         agentHash,
						ServerOfferedVersion: serverVersion,
						ServerOfferedHash:    serverHash,
						Status:               protobufs.PackageStatusEnum_PackageStatusEnum_InstallPending,
						ErrorMessage:         errMsg,
					},
				}
				packageStatuses := &protobufs.PackageStatuses{
					ServerProvidedAllPackagesHash: allHash,
					ErrorMessage:                  allErrMsg,
					Packages:                      packages,
				}

				err := p.SaveStatuses(packageStatuses)
				assert.NoError(t, err)

				bytes, err := os.ReadFile(testJSON)
				assert.NoError(t, err)
				var fileStates packageStates
				err = json.Unmarshal(bytes, &fileStates)
				assert.NoError(t, err)
				assert.Equal(t, allHash, fileStates.AllPackagesHash)
				assert.Equal(t, allErrMsg, fileStates.AllErrorMessage)
				assert.Equal(t, 1, len(fileStates.PackageStates))
				assert.Equal(t, pkgName, fileStates.PackageStates[pkgName].Name)
				assert.Equal(t, agentVersion, fileStates.PackageStates[pkgName].AgentVersion)
				assert.Equal(t, agentHash, fileStates.PackageStates[pkgName].AgentHash)
				assert.Equal(t, serverVersion, fileStates.PackageStates[pkgName].ServerVersion)
				assert.Equal(t, serverHash, fileStates.PackageStates[pkgName].ServerHash)
				assert.Equal(t, protobufs.PackageStatusEnum_PackageStatusEnum_InstallPending, fileStates.PackageStates[pkgName].Status)
				assert.Equal(t, errMsg, fileStates.PackageStates[pkgName].ErrorMessage)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
